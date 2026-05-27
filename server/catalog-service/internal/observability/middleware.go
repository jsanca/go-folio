package observability

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// Default status is 200 if WriteHeader is never explicitly called.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// RequestLogger logs every inbound HTTP request as a structured JSON line.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remoteAddr", r.RemoteAddr,
				"userAgent", r.UserAgent(),
				"status", wrapped.status,
				"durationMs", time.Since(start).Milliseconds(),
			)
		})
	}
}

// PanicRecovery catches panics, logs them as errors, and returns a 500
// using the standard API error envelope — no internal details leak to the client.
func PanicRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					logger.Error("panic recovered",
						"panic", fmt.Sprint(p),
						"method", r.Method,
						"path", r.URL.Path,
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
						"error": map[string]string{
							"code":    "INTERNAL_SERVER_ERROR",
							"message": "internal server error",
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
