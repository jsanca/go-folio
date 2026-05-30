package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
	"github.com/jsanca/go-folio/internal/service"
)

// ── stub catalog service ──────────────────────────────────────────────────────

type stubCatalogSvc struct {
	listProjectionsFn func(ctx context.Context, q domain.SyncQuery) (*domain.ProductProjectionPage, error)
	listInventoryFn   func(ctx context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error)
	createProductFn   func(ctx context.Context, p *domain.Product) (*domain.Product, error)
	updateProductFn   func(ctx context.Context, id int64, update service.ProductUpdate) (*domain.Product, error)
	deleteProductFn   func(ctx context.Context, id int64) error
}

func (s *stubCatalogSvc) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	if s.createProductFn != nil {
		return s.createProductFn(ctx, p)
	}
	return nil, nil
}
func (s *stubCatalogSvc) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return nil, nil
}
func (s *stubCatalogSvc) ListProducts(ctx context.Context) ([]domain.Product, error) { return nil, nil }
func (s *stubCatalogSvc) UpdateProduct(ctx context.Context, id int64, update service.ProductUpdate) (*domain.Product, error) {
	if s.updateProductFn != nil {
		return s.updateProductFn(ctx, id, update)
	}
	return nil, nil
}
func (s *stubCatalogSvc) DeleteProduct(ctx context.Context, id int64) error {
	if s.deleteProductFn != nil {
		return s.deleteProductFn(ctx, id)
	}
	return nil
}
func (s *stubCatalogSvc) AddVariantToProduct(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	return nil, nil
}
func (s *stubCatalogSvc) GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	return nil, nil
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

// ── POST /products handler tests ──────────────────────────────────────────────

func TestProductHandler_Create_HappyPath(t *testing.T) {
	svc := &stubCatalogSvc{
		createProductFn: func(_ context.Context, p *domain.Product) (*domain.Product, error) {
			p.ID = 1
			return p, nil
		},
	}
	body := `{"productCode":"BM-02","title":"Billetera","slug":"billetera","active":true}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m) //nolint:errcheck
	if m["productCode"] != "BM-02" {
		t.Errorf("expected productCode BM-02 in response, got %v", m["productCode"])
	}
}

func TestProductHandler_Create_MissingRequiredField(t *testing.T) {
	svc := &stubCatalogSvc{
		createProductFn: func(_ context.Context, p *domain.Product) (*domain.Product, error) {
			return nil, service.ErrInvalidProduct
		},
	}
	body := `{"title":"No Code","slug":"no-code"}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m) //nolint:errcheck
	errObj, _ := m["error"].(map[string]any)
	if errObj["code"] != "INVALID_REQUEST" {
		t.Errorf("expected INVALID_REQUEST, got %v", errObj["code"])
	}
}

func TestProductHandler_Create_MalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(&stubCatalogSvc{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestProductHandler_Create_DuplicateCode(t *testing.T) {
	svc := &stubCatalogSvc{
		createProductFn: func(_ context.Context, p *domain.Product) (*domain.Product, error) {
			return nil, repository.ErrDuplicateProductCode
		},
	}
	body := `{"productCode":"DUPE","title":"Dupe","slug":"dupe"}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m) //nolint:errcheck
	errObj, _ := m["error"].(map[string]any)
	if errObj["code"] != "CONFLICT" {
		t.Errorf("expected CONFLICT, got %v", errObj["code"])
	}
}

// ── PATCH /products/{id} handler tests ───────────────────────────────────────

func TestProductHandler_Update_HappyPath(t *testing.T) {
	svc := &stubCatalogSvc{
		updateProductFn: func(_ context.Context, id int64, u service.ProductUpdate) (*domain.Product, error) {
			title := ""
			if u.Title != nil {
				title = *u.Title
			}
			return &domain.Product{ID: id, ProductCode: "BM-02", Title: title, Slug: "billetera"}, nil
		},
	}
	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/products/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m) //nolint:errcheck
	if m["title"] != "Updated Title" {
		t.Errorf("expected updated title in response, got %v", m["title"])
	}
}

func TestProductHandler_Update_NotFound(t *testing.T) {
	svc := &stubCatalogSvc{
		updateProductFn: func(_ context.Context, id int64, u service.ProductUpdate) (*domain.Product, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	body := `{"title":"Ghost"}`
	req := httptest.NewRequest(http.MethodPatch, "/products/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProductHandler_Update_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/products/abc", bytes.NewBufferString(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(&stubCatalogSvc{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestProductHandler_Update_DuplicateSlug(t *testing.T) {
	svc := &stubCatalogSvc{
		updateProductFn: func(_ context.Context, id int64, u service.ProductUpdate) (*domain.Product, error) {
			return nil, repository.ErrDuplicateSlug
		},
	}
	body := `{"slug":"existing-slug"}`
	req := httptest.NewRequest(http.MethodPatch, "/products/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	var m map[string]any
	json.Unmarshal(rec.Body.Bytes(), &m) //nolint:errcheck
	errObj, _ := m["error"].(map[string]any)
	if errObj["code"] != "CONFLICT" {
		t.Errorf("expected CONFLICT, got %v", errObj["code"])
	}
}

// ── DELETE /products/{id} handler tests ──────────────────────────────────────

func TestProductHandler_Delete_HappyPath(t *testing.T) {
	svc := &stubCatalogSvc{
		deleteProductFn: func(_ context.Context, id int64) error { return nil },
	}
	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rec.Body.String())
	}
}

func TestProductHandler_Delete_NotFound(t *testing.T) {
	svc := &stubCatalogSvc{
		deleteProductFn: func(_ context.Context, id int64) error { return repository.ErrProductNotFound },
	}
	req := httptest.NewRequest(http.MethodDelete, "/products/999", nil)
	rec := httptest.NewRecorder()
	newCatalogRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestProductHandler_Delete_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/products/xyz", nil)
	rec := httptest.NewRecorder()
	newCatalogRouter(&stubCatalogSvc{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
