package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
	"github.com/jsanca/go-folio/internal/service"
)

// CatalogHandler handles HTTP requests for the product catalog.
type CatalogHandler struct {
	svc service.CatalogService
}

// NewCatalogHandler creates a CatalogHandler backed by the given service.
func NewCatalogHandler(svc service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

// RegisterRoutes mounts all catalog routes on the given router.
func (h *CatalogHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Post("/products", h.createProduct)
	r.Patch("/products/{id}", h.updateProduct)
	r.Delete("/products/{id}", h.deleteProduct)
	r.Post("/catalog/products/{id}/variants", h.addVariant)
	r.Get("/catalog/product-projections", h.listProductProjections)
	r.Get("/catalog/variant-inventory", h.listVariantInventory)
	r.Get("/catalog/variants/{sku}", h.getVariantBySKU)
}

func (h *CatalogHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListProductProjections(r.Context(), domain.SyncQuery{PageSize: 500})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, page.Items)
}

func (h *CatalogHandler) getVariantBySKU(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	v, err := h.svc.GetVariantBySKU(r.Context(), sku)
	if errors.Is(err, repository.ErrVariantNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "variant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
		return
	}
	product, err := h.svc.GetProductByID(r.Context(), v.ProductID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, domain.ProductProjection{
		Product:  *product,
		Variants: []domain.ProductVariant{*v},
		Images:   []domain.ProductImage{},
	})
}

func (h *CatalogHandler) listProductProjections(w http.ResponseWriter, r *http.Request) {
	ps, ok := parsePageSizeQP(w, r, "pageSize", 0)
	if !ok {
		return
	}
	updatedSince, ok := parseUpdatedSinceQP(w, r)
	if !ok {
		return
	}

	q := domain.SyncQuery{
		PageSize:     ps,
		Cursor:       r.URL.Query().Get("cursor"),
		UpdatedSince: updatedSince,
	}

	page, err := h.svc.ListProductProjections(r.Context(), q)
	if err != nil {
		handleSyncError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *CatalogHandler) listVariantInventory(w http.ResponseWriter, r *http.Request) {
	ps, ok := parsePageSizeQP(w, r, "pageSize", 0)
	if !ok {
		return
	}
	updatedSince, ok := parseUpdatedSinceQP(w, r)
	if !ok {
		return
	}

	q := domain.SyncQuery{
		PageSize:     ps,
		Cursor:       r.URL.Query().Get("cursor"),
		UpdatedSince: updatedSince,
	}

	page, err := h.svc.ListVariantInventory(r.Context(), q)
	if err != nil {
		handleSyncError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// createProductRequest is the request body for POST /products.
type createProductRequest struct {
	ProductCode      string `json:"productCode"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	ShortDescription string `json:"shortDescription"`
	Department       string `json:"department"`
	Category         string `json:"category"`
	Active           bool   `json:"active"`
}

// updateProductRequest is the request body for PATCH /products/{id}.
type updateProductRequest struct {
	ProductCode      *string `json:"productCode"`
	Title            *string `json:"title"`
	Slug             *string `json:"slug"`
	ShortDescription *string `json:"shortDescription"`
	Department       *string `json:"department"`
	Category         *string `json:"category"`
	Active           *bool   `json:"active"`
}

func (h *CatalogHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	p := &domain.Product{
		ProductCode:      req.ProductCode,
		Title:            req.Title,
		Slug:             req.Slug,
		ShortDescription: req.ShortDescription,
		Department:       req.Department,
		Category:         req.Category,
		Active:           req.Active,
	}
	created, err := h.svc.CreateProduct(r.Context(), p)
	if err != nil {
		handleProductMutationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CatalogHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id")
	if !ok {
		return
	}
	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	update := service.ProductUpdate{
		ProductCode:      req.ProductCode,
		Title:            req.Title,
		Slug:             req.Slug,
		ShortDescription: req.ShortDescription,
		Department:       req.Department,
		Category:         req.Category,
		Active:           req.Active,
	}
	updated, err := h.svc.UpdateProduct(r.Context(), id, update)
	if err != nil {
		handleProductMutationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CatalogHandler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), id); err != nil {
		handleProductMutationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createVariantRequest is the body for POST /catalog/products/{id}/variants.
type createVariantRequest struct {
	SKU              string `json:"sku"`
	ColorName        string `json:"colorName"`
	ColorSlug        string `json:"colorSlug"`
	PrimaryColorHex  string `json:"primaryColorHex"`
	RetailPriceCents *int64 `json:"retailPriceCents"`
	Currency         string `json:"currency"`
	Active           bool   `json:"active"`
}

func (h *CatalogHandler) addVariant(w http.ResponseWriter, r *http.Request) {
	productID, ok := parseIDParam(w, r, "id")
	if !ok {
		return
	}
	var req createVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.SKU == "" || req.RetailPriceCents == nil || req.Currency == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "sku, retailPriceCents, and currency are required")
		return
	}
	if _, err := h.svc.GetProductByID(r.Context(), productID); err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
		return
	}
	v := &domain.ProductVariant{
		ProductID:       productID,
		SKU:             req.SKU,
		ColorName:       req.ColorName,
		ColorSlug:       req.ColorSlug,
		PrimaryColorHex: req.PrimaryColorHex,
		RetailPrice:     domain.Money{AmountCents: *req.RetailPriceCents},
		Currency:        req.Currency,
		Active:          req.Active,
	}
	created, err := h.svc.AddVariantToProduct(r.Context(), v)
	if err != nil {
		handleVariantMutationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// parsePageSizeQP parses the pageSize query param. Returns 0 (meaning "use service default")
// when the param is absent. Returns false and writes a 400 if the value is present but invalid.
func parsePageSizeQP(w http.ResponseWriter, r *http.Request, param string, _ int) (int, bool) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "pageSize must be a positive integer")
		return 0, false
	}
	return n, true
}

// parseUpdatedSinceQP parses the updatedSince query param as an ISO-8601 timestamp.
func parseUpdatedSinceQP(w http.ResponseWriter, r *http.Request) (*time.Time, bool) {
	raw := r.URL.Query().Get("updatedSince")
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "updatedSince must be an ISO-8601 timestamp")
		return nil, false
	}
	utc := t.UTC()
	return &utc, true
}

func handleSyncError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCursor):
		writeError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid cursor")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
	}
}

// parseIDParam parses a positive int64 URL parameter. It writes a 400 and returns false on failure.
func parseIDParam(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// handleProductMutationError maps product mutation errors to HTTP responses.
func handleProductMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrProductNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
	case errors.Is(err, service.ErrInvalidProduct):
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, repository.ErrDuplicateProductCode), errors.Is(err, repository.ErrDuplicateSlug):
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
	}
}

// handleVariantMutationError maps variant mutation errors to HTTP responses.
func handleVariantMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrDuplicateSKU):
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, service.ErrInvalidVariant):
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error")
	}
}
