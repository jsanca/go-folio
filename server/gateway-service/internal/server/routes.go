package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

func registerRoutes(r chi.Router, rt *runtime.GatewayRuntime, logger *slog.Logger) {
	// ── Public routes ─────────────────────────────────────────────────────────
	NewProductsHandler(rt, logger).RegisterRoutes(r)

	// ── Admin routes — require a valid JWT with the "admin" realm role ────────
	r.Group(func(r chi.Router) {
		r.Use(rt.Auth.RequireAuth)
		r.Use(rt.Auth.RequireRole("admin"))
		r.Get("/admin/products", NewAdminProductsHandler(rt, logger).ServeHTTP)
		r.Post("/admin/products", notImplemented)
		r.Patch("/admin/products/{sku}", notImplemented)
		r.Put("/admin/inventory/{sku}", notImplemented)
	})
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}
