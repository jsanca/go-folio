package server_test

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jsanca/go-folio/gateway-service/internal/server"
	"github.com/jsanca/go-folio/gateway-service/internal/sse"
)

// TestSSEHandler_Headers verifies that GET /admin/events responds with HTTP 200
// and the required SSE headers.
func TestSSEHandler_Headers(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	rt := buildRuntime(t, catalog.URL, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gw.URL+"/admin/events", nil)

	resp, err := http.DefaultClient.Do(req)
	cancel() // disconnect once headers are received
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: want %q, got %q", "text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: want %q, got %q", "no-cache", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection: want %q, got %q", "keep-alive", conn)
	}
}

// TestSSEHandler_StreamsEvent verifies that an event published to the broker
// appears in the SSE response body as a well-formed "data: {...}\n\n" frame.
func TestSSEHandler_StreamsEvent(t *testing.T) {
	catalog := fakeCatalogSrv(t)
	rt := buildRuntime(t, catalog.URL, "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := httptest.NewServer(server.New(rt, logger))
	t.Cleanup(gw.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gw.URL+"/admin/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Publish after a short delay to ensure the handler has subscribed.
	go func() {
		time.Sleep(50 * time.Millisecond)
		rt.Events.Publish(sse.StockEvent{
			SKU:       "BAG-001-BRN",
			Available: 10,
			Reserved:  0,
			Status:    "IN_STOCK",
		})
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if !strings.Contains(payload, `"sku":"BAG-001-BRN"`) {
			t.Errorf("unexpected SSE payload: %q", payload)
		}
		if !strings.Contains(payload, `"status":"IN_STOCK"`) {
			t.Errorf("missing status in SSE payload: %q", payload)
		}
		return // received and validated the event — success
	}

	if ctx.Err() != nil {
		t.Error("timed out waiting for SSE event")
	} else if err := scanner.Err(); err != nil {
		t.Error("scanner error:", err)
	}
}
