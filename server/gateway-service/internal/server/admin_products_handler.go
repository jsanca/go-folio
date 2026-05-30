package server

import (
	"log/slog"
	"net/http"

	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

// AdminProductsHandler handles GET /admin/products, returning the full product
// list in the same shape as GET /products.  Route-level authentication and role
// enforcement are applied by the RequireAuth + RequireRole("admin") middleware
// registered in routes.go.
type AdminProductsHandler struct {
	ph *ProductsHandler
}

// NewAdminProductsHandler creates an AdminProductsHandler wired to the given runtime.
func NewAdminProductsHandler(rt *runtime.GatewayRuntime, logger *slog.Logger) *AdminProductsHandler {
	return &AdminProductsHandler{ph: NewProductsHandler(rt, logger)}
}

// ServeHTTP delegates to the shared listProducts logic, reusing catalog
// hydration and live inventory stock aggregation.
func (h *AdminProductsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ph.listProducts(w, r)
}
