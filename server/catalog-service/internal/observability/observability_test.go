package observability

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- helpers ---

func silentLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func decodeBody(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

// --- /health ---

func TestHealth_Returns200AndStatusOk(t *testing.T) {
	h := NewHealthHandler(nil, silentLogger())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeBody(t, rec.Body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
}

// --- /ready ---

type fakePinger struct{ err error }

func (f *fakePinger) PingContext(_ context.Context) error { return f.err }

func TestReady_Returns200WhenPingSucceeds(t *testing.T) {
	h := NewHealthHandler(&fakePinger{}, silentLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeBody(t, rec.Body)
	if body["status"] != "ready" {
		t.Errorf("expected status ready, got %v", body["status"])
	}
}

func TestReady_Returns503WhenPingFails(t *testing.T) {
	h := NewHealthHandler(&fakePinger{err: errors.New("db down")}, silentLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	body := decodeBody(t, rec.Body)
	if body["status"] != "not_ready" {
		t.Errorf("expected status not_ready, got %v", body["status"])
	}
}

func TestReady_DoesNotLeakDBErrorToClient(t *testing.T) {
	h := NewHealthHandler(&fakePinger{err: errors.New("secret internal error")}, silentLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	raw := rec.Body.String()
	if contains(raw, "secret internal error") {
		t.Error("response body must not contain internal error details")
	}
}

// --- RequestLogger middleware ---

func TestRequestLogger_CapturesStatusCode(t *testing.T) {
	mw := RequestLogger(silentLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestRequestLogger_DefaultsTo200IfWriteHeaderNotCalled(t *testing.T) {
	var capturedStatus int
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			capturedStatus = wrapped.status
		})
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// deliberately do not call WriteHeader
		w.Write([]byte("ok")) //nolint:errcheck
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedStatus != http.StatusOK {
		t.Errorf("expected default status 200, got %d", capturedStatus)
	}
}

// --- PanicRecovery middleware ---

func TestPanicRecovery_Returns500OnPanic(t *testing.T) {
	mw := PanicRecovery(silentLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went very wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestPanicRecovery_ReturnsStandardErrorEnvelope(t *testing.T) {
	mw := PanicRecovery(silentLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("oops")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := decodeBody(t, rec.Body)
	errField, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", body["error"])
	}
	if errField["code"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected INTERNAL_SERVER_ERROR, got %v", errField["code"])
	}
	if errField["message"] != "internal server error" {
		t.Errorf("expected 'internal server error', got %v", errField["message"])
	}
}

func TestPanicRecovery_DoesNotLeakPanicDetails(t *testing.T) {
	mw := PanicRecovery(silentLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("super secret panic reason")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if contains(rec.Body.String(), "super secret panic reason") {
		t.Error("panic details must not be leaked to the client")
	}
}

// --- Metrics ---

func TestMetrics_IsRegisteredAndEndpointReturns200(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	r := chi.NewRouter()
	r.Use(m.Middleware())
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// fire a request to generate a metric sample
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// verify /metrics returns 200 and contains our metric name
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 from /metrics, got %d", rec2.Code)
	}
	body := rec2.Body.String()
	if !contains(body, "http_requests_total") {
		t.Error("expected http_requests_total in /metrics output")
	}
	if !contains(body, "http_request_duration_seconds") {
		t.Error("expected http_request_duration_seconds in /metrics output")
	}
}

func TestMetrics_UsesRoutePatternNotRawPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	r := chi.NewRouter()
	r.Use(m.Middleware())
	r.Get("/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	body := rec2.Body.String()
	if !contains(body, `/products/{id}`) {
		t.Error("expected route pattern /products/{id} in metrics, not the raw path /products/42")
	}
	if contains(body, `"42"`) {
		t.Error("raw product ID 42 must not appear as a metric label")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
