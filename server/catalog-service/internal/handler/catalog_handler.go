package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
	"github.com/jsanca/go-folio/internal/service"
)

type CatalogHandler struct {
	svc service.CatalogService
}

func NewCatalogHandler(svc service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.listProducts)
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
