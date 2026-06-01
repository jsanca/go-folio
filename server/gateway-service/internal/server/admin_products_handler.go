package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/clients"
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

// addVariant implements a choreographed saga:
//  1. Create the variant in catalog-service.
//  2. Seed the SKU in inventory-service (delta=0).
//     If step 2 fails, compensate by deleting the variant from catalog.
func (h *AdminProductsHandler) addVariant(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid product id")
		return
	}

	// Step 1: create variant in catalog.
	variant, err := h.ph.rt.Catalog.AddVariant(r.Context(), productID, r.Body)
	if err != nil {
		var catErr *clients.CatalogError
		if errors.As(err, &catErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(catErr.Status)
			w.Write(catErr.Body) //nolint:errcheck
			return
		}
		h.ph.logger.Error("saga: catalog add variant failed", "productID", productID, "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to create variant in catalog")
		return
	}

	// Step 2: seed SKU in inventory.
	if err := h.ph.rt.Inventory.SeedSKU(r.Context(), variant.SKU); err != nil {
		h.ph.logger.Error("saga: inventory failed, compensating", "sku", variant.SKU, "productID", productID, "err", err)
		if compErr := h.ph.rt.Catalog.DeleteVariant(r.Context(), productID, variant.SKU); compErr != nil {
			h.ph.logger.Error("saga: compensation failed, manual reconciliation needed", "sku", variant.SKU, "productID", productID, "err", compErr)
		}
		writeError(w, http.StatusServiceUnavailable, "UPSTREAM_ERROR", "failed to register variant in inventory")
		return
	}

	writeJSON(w, http.StatusCreated, variant)
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
