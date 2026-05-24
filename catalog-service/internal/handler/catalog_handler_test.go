package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/leatherstore/catalog-service/internal/domain"
	"github.com/leatherstore/catalog-service/internal/service"
)

// ── stub catalog service ──────────────────────────────────────────────────────

type stubCatalogSvc struct {
	listProjectionsFn func(ctx context.Context, q domain.SyncQuery) (*domain.ProductProjectionPage, error)
	listInventoryFn   func(ctx context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error)
}

func (s *stubCatalogSvc) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	return nil, nil
}
func (s *stubCatalogSvc) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return nil, nil
}
func (s *stubCatalogSvc) ListProducts(ctx context.Context) ([]domain.Product, error) { return nil, nil }
func (s *stubCatalogSvc) AddVariantToProduct(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	return nil, nil
}
func (s *stubCatalogSvc) GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	return nil, nil
}
func (s *stubCatalogSvc) UpdateVariantInventory(ctx context.Context, sku string, qty int, status domain.StockStatus) error {
	return nil
}
func (s *stubCatalogSvc) UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error {
	return nil
}
func (s *stubCatalogSvc) ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error) {
	return nil, nil
}
func (s *stubCatalogSvc) AddProductImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error) {
	return nil, nil
}
func (s *stubCatalogSvc) ListProductImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error) {
	return nil, nil
}
func (s *stubCatalogSvc) ListProductImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error) {
	return nil, nil
}

func (s *stubCatalogSvc) ListProductProjections(ctx context.Context, q domain.SyncQuery) (*domain.ProductProjectionPage, error) {
	if s.listProjectionsFn != nil {
		return s.listProjectionsFn(ctx, q)
	}
	return &domain.ProductProjectionPage{Items: []domain.ProductProjection{}}, nil
}

func (s *stubCatalogSvc) ListVariantInventory(ctx context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error) {
	if s.listInventoryFn != nil {
		return s.listInventoryFn(ctx, q)
	}
	return &domain.VariantInventoryPage{Items: []domain.VariantInventoryRecord{}}, nil
}

// ── test helpers ──────────────────────────────────────────────────────────────

func newCatalogRouter(svc service.CatalogService) *chi.Mux {
	r := chi.NewRouter()
	NewCatalogHandler(svc).RegisterRoutes(r)
	return r
}

func decodeSyncBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

// ── product-projections tests ─────────────────────────────────────────────────

func TestCatalogHandler_ProductProjections_Returns200WithShape(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/product-projections", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeSyncBody(t, rec.Body.Bytes())
	for _, key := range []string{"items", "pageSize", "hasNext", "nextCursor", "syncToken"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response missing field %q", key)
		}
	}
}

func TestCatalogHandler_ProductProjections_ItemsIsArray(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/product-projections", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := decodeSyncBody(t, rec.Body.Bytes())
	if _, ok := body["items"].([]any); !ok {
		t.Errorf("expected items to be a JSON array, got %T", body["items"])
	}
}

func TestCatalogHandler_ProductProjections_InvalidUpdatedSince_Returns400(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/product-projections?updatedSince=not-a-date", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	body := decodeSyncBody(t, rec.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST, got %v", errObj["code"])
	}
}

func TestCatalogHandler_ProductProjections_InvalidPageSize_Returns400(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/product-projections?pageSize=abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCatalogHandler_ProductProjections_InvalidCursor_Returns400(t *testing.T) {
	svc := &stubCatalogSvc{
		listProjectionsFn: func(_ context.Context, _ domain.SyncQuery) (*domain.ProductProjectionPage, error) {
			return nil, service.ErrInvalidCursor
		},
	}
	r := newCatalogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/catalog/product-projections?cursor=bogus", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	body := decodeSyncBody(t, rec.Body.Bytes())
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "INVALID_CURSOR" {
		t.Errorf("expected INVALID_CURSOR, got %v", errObj["code"])
	}
}

// ── variant-inventory tests ───────────────────────────────────────────────────

func TestCatalogHandler_VariantInventory_Returns200WithShape(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/variant-inventory", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeSyncBody(t, rec.Body.Bytes())
	for _, key := range []string{"items", "pageSize", "hasNext", "nextCursor", "syncToken"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response missing field %q", key)
		}
	}
}

func TestCatalogHandler_VariantInventory_PageSizePassedToService(t *testing.T) {
	var capturedPageSize int
	svc := &stubCatalogSvc{
		listInventoryFn: func(_ context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error) {
			capturedPageSize = q.PageSize
			return &domain.VariantInventoryPage{Items: []domain.VariantInventoryRecord{}, PageSize: q.PageSize}, nil
		},
	}
	r := newCatalogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/catalog/variant-inventory?pageSize=42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedPageSize != 42 {
		t.Errorf("expected pageSize 42 passed to service, got %d", capturedPageSize)
	}
}

func TestCatalogHandler_VariantInventory_InvalidUpdatedSince_Returns400(t *testing.T) {
	r := newCatalogRouter(&stubCatalogSvc{})
	req := httptest.NewRequest(http.MethodGet, "/catalog/variant-inventory?updatedSince=yesterday", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCatalogHandler_VariantInventory_InvalidCursor_Returns400(t *testing.T) {
	svc := &stubCatalogSvc{
		listInventoryFn: func(_ context.Context, _ domain.SyncQuery) (*domain.VariantInventoryPage, error) {
			return nil, service.ErrInvalidCursor
		},
	}
	r := newCatalogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/catalog/variant-inventory?cursor=!!!", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCatalogHandler_ResponseFieldsAreCamelCase(t *testing.T) {
	svc := &stubCatalogSvc{
		listInventoryFn: func(_ context.Context, _ domain.SyncQuery) (*domain.VariantInventoryPage, error) {
			return &domain.VariantInventoryPage{
				Items: []domain.VariantInventoryRecord{
					{
						ProductCode:   "BM-02",
						ProductID:     1,
						VariantID:     10,
						SKU:           "BM-02-COL-CO-NE",
						RetailPrice:   domain.Money{AmountCents: 2439000},
						Currency:      "CRC",
						StockQuantity: 13,
						StockStatus:   domain.StockStatusInStock,
						Active:        true,
						UpdatedAt:     time.Now(),
					},
				},
				PageSize:  1,
				SyncToken: time.Now(),
			}, nil
		},
	}
	r := newCatalogRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/catalog/variant-inventory", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	raw := rec.Body.String()
	for _, camelKey := range []string{"productCode", "productId", "variantId", "stockQuantity", "stockStatus", "retailPrice", "amountCents", "syncToken", "nextCursor", "hasNext", "pageSize"} {
		if !strings.Contains(raw, camelKey) {
			t.Errorf("expected camelCase field %q in response", camelKey)
		}
	}
}
