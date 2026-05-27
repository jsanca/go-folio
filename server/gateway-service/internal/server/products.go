package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── Response types ────────────────────────────────────────────────────────────

type stockInfo struct {
	Available int32  `json:"available"`
	Reserved  int32  `json:"reserved"`
	Status    string `json:"stockStatus"`
}

type money struct {
	AmountCents int64 `json:"amountCents"`
}

type variantResponse struct {
	SKU             string    `json:"sku"`
	ColorName       string    `json:"colorName,omitempty"`
	ColorSlug       string    `json:"colorSlug,omitempty"`
	PrimaryColorHex string    `json:"primaryColorHex,omitempty"`
	RetailPrice     money     `json:"retailPrice"`
	Currency        string    `json:"currency"`
	Stock           stockInfo `json:"stock"`
	Active          bool      `json:"active"`
}

type productResponse struct {
	ID          int64             `json:"id"`
	ProductCode string            `json:"productCode"`
	Title       string            `json:"title"`
	Slug        string            `json:"slug"`
	Variants    []variantResponse `json:"variants"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

// ProductsHandler handles client-facing product routes on the gateway.
type ProductsHandler struct {
	rt     *runtime.GatewayRuntime
	logger *slog.Logger
}

// NewProductsHandler creates a ProductsHandler wired to the given runtime.
func NewProductsHandler(rt *runtime.GatewayRuntime, logger *slog.Logger) *ProductsHandler {
	return &ProductsHandler{rt: rt, logger: logger}
}

// RegisterRoutes wires /products and /products/{sku}.
func (h *ProductsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Get("/products/{sku}", h.getProductBySKU)
}

// listProducts aggregates all catalog products with live inventory stock.
func (h *ProductsHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	projections, err := h.rt.Catalog.ListProducts(r.Context())
	if err != nil {
		h.logger.Error("list products from catalog", "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to fetch products")
		return
	}

	result := make([]productResponse, 0, len(projections))
	for _, proj := range projections {
		variants := make([]variantResponse, 0, len(proj.Variants))
		for _, v := range proj.Variants {
			stock := h.fetchStock(r.Context(), v.SKU)
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
		result = append(result, productResponse{
			ID:          proj.Product.ID,
			ProductCode: proj.Product.ProductCode,
			Title:       proj.Product.Title,
			Slug:        proj.Product.Slug,
			Variants:    variants,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// getProductBySKU fetches a single variant by SKU and hydrates it with live inventory stock.
func (h *ProductsHandler) getProductBySKU(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	proj, err := h.rt.Catalog.GetVariantBySKU(r.Context(), sku)
	if err != nil {
		if errors.Is(err, clients.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "variant not found")
			return
		}
		h.logger.Error("get variant from catalog", "sku", sku, "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to fetch variant")
		return
	}
	if len(proj.Variants) == 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "variant not found")
		return
	}
	v := proj.Variants[0]
	stock := h.fetchStock(r.Context(), v.SKU)

	type singleResponse struct {
		ID          int64     `json:"id"`
		ProductCode string    `json:"productCode"`
		Title       string    `json:"title"`
		Slug        string    `json:"slug"`
		SKU         string    `json:"sku"`
		ColorName   string    `json:"colorName,omitempty"`
		ColorSlug   string    `json:"colorSlug,omitempty"`
		RetailPrice money     `json:"retailPrice"`
		Currency    string    `json:"currency"`
		Stock       stockInfo `json:"stock"`
	}
	writeJSON(w, http.StatusOK, singleResponse{
		ID:          proj.Product.ID,
		ProductCode: proj.Product.ProductCode,
		Title:       proj.Product.Title,
		Slug:        proj.Product.Slug,
		SKU:         v.SKU,
		ColorName:   v.ColorName,
		ColorSlug:   v.ColorSlug,
		RetailPrice: money{AmountCents: v.RetailPrice.AmountCents},
		Currency:    v.Currency,
		Stock:       stock,
	})
}

// fetchStock calls inventory-service for the given SKU.
// Falls back to available=0 / OUT_OF_STOCK on NotFound or any other error.
func (h *ProductsHandler) fetchStock(ctx context.Context, sku string) stockInfo {
	resp, err := h.rt.Inventory.Svc.GetStock(ctx, &invpb.GetStockRequest{Sku: sku})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			h.logger.Warn("get stock from inventory", "sku", sku, "err", err)
		}
		return stockInfo{Available: 0, Reserved: 0, Status: "OUT_OF_STOCK"}
	}
	return stockInfo{
		Available: resp.Available,
		Reserved:  resp.Reserved,
		Status:    deriveStockStatus(resp.Available, h.rt.LowStockThreshold),
	}
}

// deriveStockStatus computes status from available quantity and the configured threshold.
// available == 0 → OUT_OF_STOCK, available <= threshold → LOW_STOCK, else IN_STOCK.
func deriveStockStatus(available int32, threshold int) string {
	switch {
	case available == 0:
		return "OUT_OF_STOCK"
	case int(available) <= threshold:
		return "LOW_STOCK"
	default:
		return "IN_STOCK"
	}
}
