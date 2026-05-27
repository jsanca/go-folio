package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds the Prometheus instruments for the HTTP layer.
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewMetrics registers the HTTP metrics into reg and returns the Metrics instance.
// Pass prometheus.DefaultRegisterer in production; pass prometheus.NewRegistry() in tests.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests by method, path pattern, and status.",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds by method and path pattern.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}
	reg.MustRegister(m.requestsTotal, m.requestDuration)
	return m
}

// Middleware records http_requests_total and http_request_duration_seconds.
// Route patterns from chi (e.g. /products/{id}) are used as labels to avoid
// high-cardinality labels from real IDs or SKU values.
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := wrapResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			pattern := routePattern(r)
			duration := time.Since(start).Seconds()

			m.requestsTotal.WithLabelValues(r.Method, pattern, fmt.Sprint(wrapped.status)).Inc()
			m.requestDuration.WithLabelValues(r.Method, pattern).Observe(duration)
		})
	}
}

// routePattern extracts the chi route pattern (e.g. /products/{id}) from the
// request context. Falls back to the raw path for unmatched routes.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}
