package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

// SSEHandler serves the GET /admin/events endpoint.
type SSEHandler struct {
	rt     *runtime.GatewayRuntime
	logger *slog.Logger
}

// NewSSEHandler creates an SSEHandler wired to the given runtime.
func NewSSEHandler(rt *runtime.GatewayRuntime, logger *slog.Logger) *SSEHandler {
	return &SSEHandler{rt: rt, logger: logger}
}

// streamEvents handles GET /admin/events.
// It subscribes to the SSE broker and pushes StockEvents as text/event-stream
// until the client disconnects.
func (h *SSEHandler) streamEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("SSE: ResponseWriter does not support flushing")
		return
	}
	flusher.Flush()

	ch := h.rt.Events.Subscribe()
	defer h.rt.Events.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("SSE: marshal event", "err", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
