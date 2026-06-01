package server

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

func registerRoutes(r chi.Router, rt *runtime.GatewayRuntime, logger *slog.Logger) {
	// ── Public routes ─────────────────────────────────────────────────────────
	NewProductsHandler(rt, logger).RegisterRoutes(r)

	// ── Admin routes — require a valid JWT with the "admin" realm role ────────
	adminH := NewAdminProductsHandler(rt, logger)
	adminInvH := NewAdminInventoryHandler(rt, logger)
	sseH := NewSSEHandler(rt, logger)
	r.Group(func(r chi.Router) {
		r.Use(rt.Auth.RequireAuth)
		r.Use(rt.Auth.RequireRole("admin"))
		r.Get("/admin/products", adminH.listProducts)
		r.Post("/admin/products", adminH.createProduct)
		r.Patch("/admin/products/{id}", adminH.updateProduct)
		r.Delete("/admin/products/{id}", adminH.deleteProduct)
		r.Post("/admin/products/{id}/variants", adminH.addVariant)
		r.Get("/admin/inventory", adminInvH.listStock)
		r.Get("/admin/inventory/{sku}", adminInvH.getStock)
		r.Put("/admin/inventory/{sku}", adminInvH.adjustStock)
		r.Get("/admin/events", sseH.streamEvents)
	})
}
