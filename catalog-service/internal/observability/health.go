package observability

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Pinger is satisfied by *sql.DB and any test stub that can check connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler serves the /health and /ready endpoints.
type HealthHandler struct {
	db     Pinger
	logger *slog.Logger
}

// NewHealthHandler creates a HealthHandler using the provided Pinger and logger.
// *sql.DB satisfies Pinger and can be passed directly.
func NewHealthHandler(p Pinger, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{db: p, logger: logger}
}

// Health reports that the process is alive. No external checks.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}

// Ready checks database connectivity. Returns 503 if the DB is unreachable.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		h.logger.Error("readiness check failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"}) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"}) //nolint:errcheck
}
