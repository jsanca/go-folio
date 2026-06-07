package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jsanca/go-folio/gateway-service/internal/runtime"
	"github.com/jsanca/go-folio/gateway-service/internal/sse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// inventoryItemResponse is the shape returned by GET /admin/inventory and GET /admin/inventory/{sku}.
type inventoryItemResponse struct {
	SKU       string `json:"sku"`
	Available int    `json:"available"`
	Reserved  int    `json:"reserved"`
	Status    string `json:"status"`
}

// adjustStockRequest is the body for PUT /admin/inventory/{sku}.
type adjustStockRequest struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
}

// adjustStockResponse is the shape returned by PUT /admin/inventory/{sku}.
type adjustStockResponse struct {
	SKU       string `json:"sku"`
	Available int    `json:"available"`
	Status    string `json:"status"`
}

// AdminInventoryHandler handles admin inventory routes on the gateway.
// Route-level authentication and role enforcement are applied by the
// RequireAuth + RequireRole("admin") middleware registered in routes.go.
type AdminInventoryHandler struct {
	rt     *runtime.GatewayRuntime
	logger *slog.Logger
}

// NewAdminInventoryHandler creates an AdminInventoryHandler wired to the given runtime.
func NewAdminInventoryHandler(rt *runtime.GatewayRuntime, logger *slog.Logger) *AdminInventoryHandler {
	return &AdminInventoryHandler{rt: rt, logger: logger}
}

// listStock handles GET /admin/inventory — returns all SKUs with current stock levels.
func (h *AdminInventoryHandler) listStock(w http.ResponseWriter, r *http.Request) {
	items, err := h.rt.Inventory.ListStock(r.Context())
	if err != nil {
		h.logger.Error("list stock from inventory", "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to fetch stock")
		return
	}
	result := make([]inventoryItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, inventoryItemResponse{
			SKU:       item.SKU,
			Available: item.Available,
			Reserved:  item.Reserved,
			Status:    deriveStockStatus(item.Available, h.rt.LowStockThreshold),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// getStock handles GET /admin/inventory/{sku} — returns stock for a single SKU.
func (h *AdminInventoryHandler) getStock(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	stock, err := h.rt.Inventory.GetStock(r.Context(), sku)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "sku not found")
			return
		}
		h.logger.Error("get stock from inventory", "sku", sku, "err", err)
		writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to fetch stock")
		return
	}
	writeJSON(w, http.StatusOK, inventoryItemResponse{
		SKU:       stock.SKU,
		Available: stock.Available,
		Reserved:  stock.Reserved,
		Status:    deriveStockStatus(stock.Available, h.rt.LowStockThreshold),
	})
}

// adjustStock handles PUT /admin/inventory/{sku} — applies a stock delta.
func (h *AdminInventoryHandler) adjustStock(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	var req adjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	stock, err := h.rt.Inventory.AdjustStock(r.Context(), sku, req.Delta, req.Reason)
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "sku not found")
		case codes.FailedPrecondition:
			writeError(w, http.StatusUnprocessableEntity, "INSUFFICIENT_STOCK", "insufficient stock")
		default:
			h.logger.Error("adjust stock", "sku", sku, "delta", req.Delta, "err", err)
			writeError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "failed to adjust stock")
		}
		return
	}
	// Broadcast a stock change notification to all connected SSE clients.
	// Stock mutations always originate in inventory-service; the gateway's role
	// is delivery to the browser layer only.
	h.rt.Events.Publish(sse.StockEvent{
		EventType:  deriveEventType(stock.Available, h.rt.LowStockThreshold),
		SKU:        sku,
		Available:  int32(stock.Available),
		Reserved:   0,
		Status:     deriveStockStatus(stock.Available, h.rt.LowStockThreshold),
		OccurredAt: time.Now().UTC(),
	})
	writeJSON(w, http.StatusOK, adjustStockResponse{
		SKU:       stock.SKU,
		Available: stock.Available,
		Status:    deriveStockStatus(stock.Available, h.rt.LowStockThreshold),
	})
}

// deriveEventType maps available stock to a named event type.
func deriveEventType(available int, threshold int) string {
	switch {
	case available == 0:
		return "stock.out"
	case available <= threshold:
		return "stock.low"
	default:
		return "stock.updated"
	}
}
