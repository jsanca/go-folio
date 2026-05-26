package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/service"
)

type CatalogHandler struct {
	svc service.CatalogService
}

func NewCatalogHandler(svc service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) RegisterRoutes(r chi.Router) {
	r.Get("/catalog/product-projections", h.listProductProjections)
	r.Get("/catalog/variant-inventory", h.listVariantInventory)
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
