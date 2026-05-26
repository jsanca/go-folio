package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
	"github.com/jsanca/go-folio/internal/service"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// RegisterRoutes mounts all product endpoints onto the given router.
// Literal-segment routes (sku/) are registered before parameterized ones ({id})
// so chi's trie matches them first.
func (h *ProductHandler) RegisterRoutes(r chi.Router) {
	r.Post("/products", h.createProduct)
	r.Get("/products", h.listProducts)
	r.Get("/products/sku/{sku}", h.getProductBySKU)
	r.Get("/products/{id}", h.getProductByID)
	r.Put("/products/{id}", h.updateProduct)
	r.Delete("/products/{id}", h.deleteProduct)
	r.Patch("/products/{id}/activate", h.activateProduct)
	r.Patch("/products/{id}/deactivate", h.deactivateProduct)
	r.Patch("/products/sku/{sku}/inventory", h.updateInventory)
	r.Patch("/products/sku/{sku}/pricing", h.updatePricing)
}

// --- request types ---

type updateInventoryRequest struct {
	Quantity    int                `json:"quantity"`
	StockStatus domain.StockStatus `json:"stockStatus"`
}

type updatePricingRequest struct {
	RetailPrice domain.Money  `json:"retailPrice"`
	SalePrice   *domain.Money `json:"salePrice,omitempty"`
	Currency    string        `json:"currency"`
}

// --- error response ---

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorResponse{Error: apiError{Code: code, Message: message}})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidProduct):
		writeError(w, http.StatusBadRequest, "INVALID_PRODUCT", err.Error())
	case errors.Is(err, repository.ErrProductNotFound):
		writeError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", err.Error())
	case errors.Is(err, repository.ErrDuplicateSKU):
		writeError(w, http.StatusConflict, "DUPLICATE_SKU", err.Error())
	case errors.Is(err, repository.ErrDuplicateSlug):
		writeError(w, http.StatusConflict, "DUPLICATE_SLUG", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "product ID must be an integer")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return false
	}
	return true
}

// --- handlers ---

func (h *ProductHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var p domain.LeatherProduct
	if !decodeJSON(w, r, &p) {
		return
	}
	created, err := h.svc.CreateProduct(r.Context(), &p)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *ProductHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.ListProducts(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if products == nil {
		products = []domain.LeatherProduct{}
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) getProductByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := h.svc.GetProductByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProductHandler) getProductBySKU(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	p, err := h.svc.GetProductBySKU(r.Context(), sku)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProductHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var p domain.LeatherProduct
	if !decodeJSON(w, r, &p) {
		return
	}
	p.ID = id
	updated, err := h.svc.UpdateProduct(r.Context(), &p)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ProductHandler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) activateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.ActivateProduct(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) deactivateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeactivateProduct(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) updateInventory(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	var req updateInventoryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := h.svc.UpdateInventory(r.Context(), sku, req.Quantity, req.StockStatus)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProductHandler) updatePricing(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	var req updatePricingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := h.svc.UpdatePricing(r.Context(), sku, req.RetailPrice, req.SalePrice, req.Currency)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
