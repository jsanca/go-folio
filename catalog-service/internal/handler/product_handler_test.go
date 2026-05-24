package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/leatherstore/catalog-service/internal/domain"
	"github.com/leatherstore/catalog-service/internal/repository"
	"github.com/leatherstore/catalog-service/internal/service"
)

// --- fake service ---

type fakeProductService struct {
	createProduct   func(context.Context, *domain.LeatherProduct) (*domain.LeatherProduct, error)
	updateProduct   func(context.Context, *domain.LeatherProduct) (*domain.LeatherProduct, error)
	getProductByID  func(context.Context, int64) (*domain.LeatherProduct, error)
	getProductBySKU func(context.Context, string) (*domain.LeatherProduct, error)
	listProducts    func(context.Context) ([]domain.LeatherProduct, error)
	deleteProduct   func(context.Context, int64) error
	activateProduct   func(context.Context, int64) error
	deactivateProduct func(context.Context, int64) error
	updateInventory func(context.Context, string, int, domain.StockStatus) (*domain.LeatherProduct, error)
	updatePricing   func(context.Context, string, domain.Money, *domain.Money, string) (*domain.LeatherProduct, error)
}

func (f *fakeProductService) CreateProduct(ctx context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
	return f.createProduct(ctx, p)
}
func (f *fakeProductService) UpdateProduct(ctx context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
	return f.updateProduct(ctx, p)
}
func (f *fakeProductService) GetProductByID(ctx context.Context, id int64) (*domain.LeatherProduct, error) {
	return f.getProductByID(ctx, id)
}
func (f *fakeProductService) GetProductBySKU(ctx context.Context, sku string) (*domain.LeatherProduct, error) {
	return f.getProductBySKU(ctx, sku)
}
func (f *fakeProductService) ListProducts(ctx context.Context) ([]domain.LeatherProduct, error) {
	return f.listProducts(ctx)
}
func (f *fakeProductService) DeleteProduct(ctx context.Context, id int64) error {
	return f.deleteProduct(ctx, id)
}
func (f *fakeProductService) ActivateProduct(ctx context.Context, id int64) error {
	return f.activateProduct(ctx, id)
}
func (f *fakeProductService) DeactivateProduct(ctx context.Context, id int64) error {
	return f.deactivateProduct(ctx, id)
}
func (f *fakeProductService) UpdateInventory(ctx context.Context, sku string, qty int, status domain.StockStatus) (*domain.LeatherProduct, error) {
	return f.updateInventory(ctx, sku, qty, status)
}
func (f *fakeProductService) UpdatePricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) (*domain.LeatherProduct, error) {
	return f.updatePricing(ctx, sku, retail, sale, currency)
}

// --- test helpers ---

func newRouter(svc service.ProductService) *chi.Mux {
	r := chi.NewRouter()
	NewProductHandler(svc).RegisterRoutes(r)
	return r
}

func sampleProduct() *domain.LeatherProduct {
	return &domain.LeatherProduct{
		ID:            1,
		SKU:           "LTH-001",
		Title:         "Bolso de Cuero",
		Slug:          "bolso-de-cuero",
		Currency:      "USD",
		RetailPrice:   domain.Money{AmountCents: 19999},
		StockQuantity: 10,
		StockStatus:   domain.StockStatusInStock,
		Active:        true,
	}
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func decodeErrorResponse(t *testing.T, body *bytes.Buffer) apiErrorResponse {
	t.Helper()
	var resp apiErrorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return resp
}

// --- POST /products ---

func TestCreateProduct_Success_Returns201(t *testing.T) {
	svc := &fakeProductService{
		createProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			p.ID = 1
			return p, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", toJSON(t, sampleProduct()))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	var p domain.LeatherProduct
	json.NewDecoder(rec.Body).Decode(&p)
	if p.ID != 1 {
		t.Errorf("expected product ID 1, got %d", p.ID)
	}
}

func TestCreateProduct_InvalidProduct_Returns400(t *testing.T) {
	svc := &fakeProductService{
		createProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			return nil, errors.Join(service.ErrInvalidProduct, errors.New("SKU is required"))
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", toJSON(t, domain.LeatherProduct{}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "INVALID_PRODUCT" {
		t.Errorf("expected INVALID_PRODUCT, got %s", resp.Error.Code)
	}
}

func TestCreateProduct_DuplicateSKU_Returns409(t *testing.T) {
	svc := &fakeProductService{
		createProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			return nil, repository.ErrDuplicateSKU
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", toJSON(t, sampleProduct()))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "DUPLICATE_SKU" {
		t.Errorf("expected DUPLICATE_SKU, got %s", resp.Error.Code)
	}
}

func TestCreateProduct_DuplicateSlug_Returns409(t *testing.T) {
	svc := &fakeProductService{
		createProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			return nil, repository.ErrDuplicateSlug
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", toJSON(t, sampleProduct()))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "DUPLICATE_SLUG" {
		t.Errorf("expected DUPLICATE_SLUG, got %s", resp.Error.Code)
	}
}

func TestCreateProduct_MalformedJSON_Returns400WithInvalidJSON(t *testing.T) {
	svc := &fakeProductService{}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("{broken json"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "INVALID_JSON" {
		t.Errorf("expected INVALID_JSON, got %s", resp.Error.Code)
	}
}

func TestCreateProduct_UnknownField_Returns400(t *testing.T) {
	svc := &fakeProductService{}
	r := newRouter(svc)

	body := bytes.NewBufferString(`{"unknownField": "value", "anotherUnknown": 42}`)
	req := httptest.NewRequest(http.MethodPost, "/products", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "INVALID_JSON" {
		t.Errorf("expected INVALID_JSON, got %s", resp.Error.Code)
	}
}

// --- GET /products/{id} ---

func TestGetProductByID_Found_Returns200(t *testing.T) {
	svc := &fakeProductService{
		getProductByID: func(_ context.Context, id int64) (*domain.LeatherProduct, error) {
			p := sampleProduct()
			p.ID = id
			return p, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetProductByID_NotFound_Returns404(t *testing.T) {
	svc := &fakeProductService{
		getProductByID: func(_ context.Context, id int64) (*domain.LeatherProduct, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/99", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "PRODUCT_NOT_FOUND" {
		t.Errorf("expected PRODUCT_NOT_FOUND, got %s", resp.Error.Code)
	}
}

func TestGetProductByID_InvalidID_Returns400(t *testing.T) {
	svc := &fakeProductService{}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- GET /products/sku/{sku} ---

func TestGetProductBySKU_Found_Returns200(t *testing.T) {
	svc := &fakeProductService{
		getProductBySKU: func(_ context.Context, sku string) (*domain.LeatherProduct, error) {
			p := sampleProduct()
			p.SKU = sku
			return p, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/sku/LTH-001", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetProductBySKU_NotFound_Returns404(t *testing.T) {
	svc := &fakeProductService{
		getProductBySKU: func(_ context.Context, sku string) (*domain.LeatherProduct, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products/sku/NOPE", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- GET /products ---

func TestListProducts_Returns200WithArray(t *testing.T) {
	svc := &fakeProductService{
		listProducts: func(_ context.Context) ([]domain.LeatherProduct, error) {
			return []domain.LeatherProduct{*sampleProduct()}, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var products []domain.LeatherProduct
	json.NewDecoder(rec.Body).Decode(&products)
	if len(products) != 1 {
		t.Errorf("expected 1 product, got %d", len(products))
	}
}

func TestListProducts_EmptyList_ReturnsEmptyArray(t *testing.T) {
	svc := &fakeProductService{
		listProducts: func(_ context.Context) ([]domain.LeatherProduct, error) {
			return nil, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var products []domain.LeatherProduct
	json.NewDecoder(rec.Body).Decode(&products)
	if len(products) != 0 {
		t.Errorf("expected empty array, got %d items", len(products))
	}
}

// --- PUT /products/{id} ---

func TestUpdateProduct_Success_Returns200(t *testing.T) {
	svc := &fakeProductService{
		updateProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			return p, nil
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/products/1", toJSON(t, sampleProduct()))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestUpdateProduct_NotFound_Returns404(t *testing.T) {
	svc := &fakeProductService{
		updateProduct: func(_ context.Context, p *domain.LeatherProduct) (*domain.LeatherProduct, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/products/99", toJSON(t, sampleProduct()))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- DELETE /products/{id} ---

func TestDeleteProduct_Success_Returns204(t *testing.T) {
	svc := &fakeProductService{
		deleteProduct: func(_ context.Context, id int64) error { return nil },
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestDeleteProduct_NotFound_Returns404(t *testing.T) {
	svc := &fakeProductService{
		deleteProduct: func(_ context.Context, id int64) error {
			return repository.ErrProductNotFound
		},
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/products/99", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- PATCH /products/{id}/activate ---

func TestActivateProduct_Success_Returns204(t *testing.T) {
	svc := &fakeProductService{
		activateProduct: func(_ context.Context, id int64) error { return nil },
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPatch, "/products/1/activate", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

// --- PATCH /products/{id}/deactivate ---

func TestDeactivateProduct_Success_Returns204(t *testing.T) {
	svc := &fakeProductService{
		deactivateProduct: func(_ context.Context, id int64) error { return nil },
	}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodPatch, "/products/1/deactivate", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

// --- PATCH /products/sku/{sku}/inventory ---

func TestUpdateInventory_Success_Returns200(t *testing.T) {
	svc := &fakeProductService{
		updateInventory: func(_ context.Context, sku string, qty int, status domain.StockStatus) (*domain.LeatherProduct, error) {
			p := sampleProduct()
			p.StockQuantity = qty
			p.StockStatus = status
			return p, nil
		},
	}
	r := newRouter(svc)

	body := toJSON(t, updateInventoryRequest{Quantity: 5, StockStatus: domain.StockStatusLowStock})
	req := httptest.NewRequest(http.MethodPatch, "/products/sku/LTH-001/inventory", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var p domain.LeatherProduct
	json.NewDecoder(rec.Body).Decode(&p)
	if p.StockQuantity != 5 {
		t.Errorf("expected quantity 5, got %d", p.StockQuantity)
	}
	if p.StockStatus != domain.StockStatusLowStock {
		t.Errorf("expected LOW_STOCK, got %s", p.StockStatus)
	}
}

func TestUpdateInventory_InvalidStock_Returns400(t *testing.T) {
	svc := &fakeProductService{
		updateInventory: func(_ context.Context, sku string, qty int, status domain.StockStatus) (*domain.LeatherProduct, error) {
			return nil, errors.Join(service.ErrInvalidProduct, errors.New("stock quantity must not be negative"))
		},
	}
	r := newRouter(svc)

	body := toJSON(t, updateInventoryRequest{Quantity: -1, StockStatus: domain.StockStatusInStock})
	req := httptest.NewRequest(http.MethodPatch, "/products/sku/LTH-001/inventory", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateInventory_ProductNotFound_Returns404(t *testing.T) {
	svc := &fakeProductService{
		updateInventory: func(_ context.Context, sku string, qty int, status domain.StockStatus) (*domain.LeatherProduct, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	r := newRouter(svc)

	body := toJSON(t, updateInventoryRequest{Quantity: 5, StockStatus: domain.StockStatusInStock})
	req := httptest.NewRequest(http.MethodPatch, "/products/sku/NOPE/inventory", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- PATCH /products/sku/{sku}/pricing ---

func TestUpdatePricing_Success_Returns200(t *testing.T) {
	svc := &fakeProductService{
		updatePricing: func(_ context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) (*domain.LeatherProduct, error) {
			p := sampleProduct()
			p.RetailPrice = retail
			p.SalePrice = sale
			p.Currency = currency
			return p, nil
		},
	}
	r := newRouter(svc)

	sale := domain.Money{AmountCents: 14999}
	body := toJSON(t, updatePricingRequest{
		RetailPrice: domain.Money{AmountCents: 25000},
		SalePrice:   &sale,
		Currency:    "EUR",
	})
	req := httptest.NewRequest(http.MethodPatch, "/products/sku/LTH-001/pricing", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var p domain.LeatherProduct
	json.NewDecoder(rec.Body).Decode(&p)
	if p.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", p.Currency)
	}
	if p.RetailPrice.AmountCents != 25000 {
		t.Errorf("expected retail 25000 cents, got %d", p.RetailPrice.AmountCents)
	}
}

func TestUpdatePricing_UnknownError_Returns500(t *testing.T) {
	svc := &fakeProductService{
		updatePricing: func(_ context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) (*domain.LeatherProduct, error) {
			return nil, errors.New("database connection lost")
		},
	}
	r := newRouter(svc)

	body := toJSON(t, updatePricingRequest{RetailPrice: domain.Money{AmountCents: 100}, Currency: "USD"})
	req := httptest.NewRequest(http.MethodPatch, "/products/sku/LTH-001/pricing", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	resp := decodeErrorResponse(t, rec.Body)
	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %s", resp.Error.Code)
	}
}
