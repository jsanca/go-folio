package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

func registerRoutes(r chi.Router, rt *runtime.GatewayRuntime, logger *slog.Logger) {
	NewProductsHandler(rt, logger).RegisterRoutes(r)

	// ── Admin routes (not yet implemented) ────────────────────────────────────
	r.Get("/admin/products", notImplemented)
	r.Post("/admin/products", notImplemented)
	r.Patch("/admin/products/{sku}", notImplemented)
	r.Put("/admin/inventory/{sku}", notImplemented)
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}
