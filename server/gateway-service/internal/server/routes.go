package server

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

func registerRoutes(r chi.Router, rt *runtime.GatewayRuntime, logger *slog.Logger) {
	sseH := NewSSEHandler(rt, logger)

	// ── /public — store-facing routes, no authentication required ─────────────
	r.Route("/public", func(r chi.Router) {
		NewProductsHandler(rt, logger).RegisterRoutes(r)
		r.Get("/events", sseH.streamEvents)
	})

	// ── /admin — internal staff UI, requires JWT with "admin" realm role ──────
	r.Route("/admin", func(r chi.Router) {
		r.Use(rt.Auth.RequireAuth)
		r.Use(rt.Auth.RequireRole("admin"))
		adminH := NewAdminProductsHandler(rt, logger)
		adminInvH := NewAdminInventoryHandler(rt, logger)
		r.Get("/products", adminH.listProducts)
		r.Post("/products", adminH.createProduct)
		r.Patch("/products/{id}", adminH.updateProduct)
		r.Delete("/products/{id}", adminH.deleteProduct)
		r.Post("/products/{id}/variants", adminH.addVariant)
		r.Get("/inventory", adminInvH.listStock)
		r.Get("/inventory/{sku}", adminInvH.getStock)
		r.Put("/inventory/{sku}", adminInvH.adjustStock)
		r.Get("/events", sseH.streamEvents)
	})

	// ── /account — Keycloak-authenticated buyer routes (future) ───────────────
	// Customers authenticate via Keycloak social login (Google, Facebook); distinct
	// from /admin which is for internal staff with the "admin" realm role.
	// Populated when cart, checkout, order history, and profile features are built.
	r.Route("/account", func(r chi.Router) {
		r.Use(rt.Auth.RequireAuth)
		// TODO: cart, checkout, order history, profile routes go here
	})
}
