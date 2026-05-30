package server

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
)

// adminProductResponse is the full product shape returned by GET /admin/products.
// It includes fields omitted from the public productResponse (shortDescription,
// department, category, active) needed by the admin catalog management UI.
type adminProductResponse struct {
	ID               int64             `json:"id"`
	ProductCode      string            `json:"productCode"`
	Title            string            `json:"title"`
	Slug             string            `json:"slug"`
	ShortDescription string            `json:"shortDescription"`
	Department       string            `json:"department"`
	Category         string            `json:"category"`
	Active           bool              `json:"active"`
	Variants         []variantResponse `json:"variants"`
}

// AdminProductsHandler handles admin product routes on the gateway.
// Route-level authentication and role enforcement are applied by the
// RequireAuth + RequireRole("admin") middleware registered in routes.go.
type AdminProductsHandler struct {
	ph *ProductsHandler
}

// NewAdminProductsHandler creates an AdminProductsHandler wired to the given runtime.
func NewAdminProductsHandler(rt *runtime.GatewayRuntime, logger *slog.Logger) *AdminProductsHandler {
	return &AdminProductsHandler{ph: NewProductsHandler(rt, logger)}
}

// ServeHTTP handles GET /admin/products.
func (h *AdminProductsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.listProducts(w, r)
}

// listProducts aggregates all catalog products with live inventory stock,
// returning the full admin product shape including metadata fields.
func (h *AdminProductsHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	projections, err := h.ph.rt.Catalog.ListProducts(r.Context())
	if err != nil {
		h.ph.logger.Error("list admin products from catalog", "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to fetch products")
		return
	}

	result := make([]adminProductResponse, 0, len(projections))
	for _, proj := range projections {
		variants := make([]variantResponse, 0, len(proj.Variants))
		for _, v := range proj.Variants {
			stock := h.ph.fetchStock(r.Context(), v.SKU)
			variants = append(variants, variantResponse{
				SKU:             v.SKU,
				ColorName:       v.ColorName,
				ColorSlug:       v.ColorSlug,
				PrimaryColorHex: v.PrimaryColorHex,
				RetailPrice:     money{AmountCents: v.RetailPrice.AmountCents},
				Currency:        v.Currency,
				Stock:           stock,
				Active:          v.Active,
			})
		}
		result = append(result, adminProductResponse{
			ID:               proj.Product.ID,
			ProductCode:      proj.Product.ProductCode,
			Title:            proj.Product.Title,
			Slug:             proj.Product.Slug,
			ShortDescription: proj.Product.ShortDescription,
			Department:       proj.Product.Department,
			Category:         proj.Product.Category,
			Active:           proj.Product.Active,
			Variants:         variants,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// createProduct proxies POST /admin/products → catalog-service POST /products.
func (h *AdminProductsHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	h.proxyTo(w, r, http.MethodPost, "/products", r.Body)
}

// updateProduct proxies PATCH /admin/products/{id} → catalog-service PATCH /products/{id}.
func (h *AdminProductsHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.proxyTo(w, r, http.MethodPatch, "/products/"+id, r.Body)
}

// deleteProduct proxies DELETE /admin/products/{id} → catalog-service DELETE /products/{id}.
func (h *AdminProductsHandler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.proxyTo(w, r, http.MethodDelete, "/products/"+id, nil)
}

// proxyTo forwards a request to the catalog-service and writes the upstream
// status code and body directly to the response writer.
func (h *AdminProductsHandler) proxyTo(w http.ResponseWriter, r *http.Request, method, path string, body io.Reader) {
	statusCode, respBody, err := h.ph.rt.Catalog.ProxyRequest(r.Context(), method, path, body)
	if err != nil {
		h.ph.logger.Error("catalog proxy", "method", method, "path", path, "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "upstream request failed")
		return
	}
	if len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(statusCode)
	if len(respBody) > 0 {
		w.Write(respBody) //nolint:errcheck
	}
}
