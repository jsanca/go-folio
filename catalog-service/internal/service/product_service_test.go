package service

import (
	"context"
	"errors"
	"testing"

	"github.com/leatherstore/catalog-service/internal/domain"
	"github.com/leatherstore/catalog-service/internal/repository"
)

// fakeProductRepository is an in-memory ProductRepository for unit tests.
type fakeProductRepository struct {
	products map[int64]*domain.LeatherProduct
	nextID   int64
}

func newFakeRepo() *fakeProductRepository {
	return &fakeProductRepository{
		products: make(map[int64]*domain.LeatherProduct),
		nextID:   1,
	}
}

func (r *fakeProductRepository) FindByID(_ context.Context, id int64) (*domain.LeatherProduct, error) {
	p, ok := r.products[id]
	if !ok {
		return nil, repository.ErrProductNotFound
	}
	clone := *p
	return &clone, nil
}

func (r *fakeProductRepository) FindBySKU(_ context.Context, sku string) (*domain.LeatherProduct, error) {
	for _, p := range r.products {
		if p.SKU == sku {
			clone := *p
			return &clone, nil
		}
	}
	return nil, repository.ErrProductNotFound
}

func (r *fakeProductRepository) Save(_ context.Context, p *domain.LeatherProduct) error {
	p.ID = r.nextID
	r.nextID++
	clone := *p
	r.products[clone.ID] = &clone
	return nil
}

func (r *fakeProductRepository) Update(_ context.Context, p *domain.LeatherProduct) error {
	if _, ok := r.products[p.ID]; !ok {
		return repository.ErrProductNotFound
	}
	clone := *p
	r.products[clone.ID] = &clone
	return nil
}

func (r *fakeProductRepository) Delete(_ context.Context, id int64) error {
	if _, ok := r.products[id]; !ok {
		return repository.ErrProductNotFound
	}
	delete(r.products, id)
	return nil
}

func (r *fakeProductRepository) List(_ context.Context, limit, offset int) ([]domain.LeatherProduct, error) {
	result := make([]domain.LeatherProduct, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, *p)
	}
	return result, nil
}

// helpers

func validProduct() *domain.LeatherProduct {
	return &domain.LeatherProduct{
		SKU:           "LTH-001",
		Title:         "Bolso de Cuero Premium",
		Slug:          "bolso-cuero-premium",
		Currency:      "USD",
		RetailPrice:   domain.Money{AmountCents: 19999},
		StockQuantity: 10,
		StockStatus:   domain.StockStatusInStock,
	}
}

func newService() ProductService {
	return NewProductService(newFakeRepo())
}

// --- CreateProduct ---

func TestCreateProduct_ValidProduct_Saves(t *testing.T) {
	svc := newService()
	p, err := svc.CreateProduct(context.Background(), validProduct())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected product to have an assigned ID")
	}
}

func TestCreateProduct_InvalidProduct_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	bad := validProduct()
	bad.SKU = ""
	_, err := svc.CreateProduct(context.Background(), bad)
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

// --- GetProductBySKU ---

func TestGetProductBySKU_EmptySKU_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	_, err := svc.GetProductBySKU(context.Background(), "   ")
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

func TestGetProductBySKU_NotFound_PropagatesErrProductNotFound(t *testing.T) {
	svc := newService()
	_, err := svc.GetProductBySKU(context.Background(), "NONEXISTENT")
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got: %v", err)
	}
}

func TestGetProductBySKU_Found_ReturnsProduct(t *testing.T) {
	svc := newService()
	_, _ = svc.CreateProduct(context.Background(), validProduct())
	p, err := svc.GetProductBySKU(context.Background(), "LTH-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.SKU != "LTH-001" {
		t.Errorf("expected SKU LTH-001, got: %s", p.SKU)
	}
}

// --- UpdateInventory ---

func TestUpdateInventory_UpdatesQuantityAndStockStatus(t *testing.T) {
	svc := newService()
	_, _ = svc.CreateProduct(context.Background(), validProduct())

	p, err := svc.UpdateInventory(context.Background(), "LTH-001", 5, domain.StockStatusLowStock)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.StockQuantity != 5 {
		t.Errorf("expected quantity 5, got: %d", p.StockQuantity)
	}
	if p.StockStatus != domain.StockStatusLowStock {
		t.Errorf("expected LOW_STOCK, got: %s", p.StockStatus)
	}
}

func TestUpdateInventory_NegativeQuantity_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	_, err := svc.UpdateInventory(context.Background(), "LTH-001", -1, domain.StockStatusInStock)
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

func TestUpdateInventory_InvalidStockStatus_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	_, err := svc.UpdateInventory(context.Background(), "LTH-001", 10, "UNKNOWN")
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

func TestUpdateInventory_SKUNotFound_ReturnsErrProductNotFound(t *testing.T) {
	svc := newService()
	_, err := svc.UpdateInventory(context.Background(), "NONEXISTENT", 10, domain.StockStatusInStock)
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got: %v", err)
	}
}

// --- UpdatePricing ---

func TestUpdatePricing_UpdatesFields(t *testing.T) {
	svc := newService()
	_, _ = svc.CreateProduct(context.Background(), validProduct())

	sale := domain.Money{AmountCents: 14999}
	p, err := svc.UpdatePricing(context.Background(), "LTH-001", domain.Money{AmountCents: 25000}, &sale, "EUR")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.RetailPrice.AmountCents != 25000 {
		t.Errorf("expected retail 25000 cents, got: %d", p.RetailPrice.AmountCents)
	}
	if p.SalePrice == nil || p.SalePrice.AmountCents != 14999 {
		t.Error("expected sale price 14999 cents")
	}
	if p.Currency != "EUR" {
		t.Errorf("expected currency EUR, got: %s", p.Currency)
	}
}

func TestUpdatePricing_NegativeRetailPrice_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	_, err := svc.UpdatePricing(context.Background(), "LTH-001", domain.Money{AmountCents: -1}, nil, "USD")
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

func TestUpdatePricing_EmptyCurrency_ReturnsErrInvalidProduct(t *testing.T) {
	svc := newService()
	_, err := svc.UpdatePricing(context.Background(), "LTH-001", domain.Money{AmountCents: 100}, nil, "")
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got: %v", err)
	}
}

// --- ActivateProduct / DeactivateProduct ---

func TestActivateProduct_SetsActiveTrue(t *testing.T) {
	svc := newService()
	product := validProduct()
	product.Active = false
	created, _ := svc.CreateProduct(context.Background(), product)

	if err := svc.ActivateProduct(context.Background(), created.ID); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := svc.GetProductByID(context.Background(), created.ID)
	if !got.Active {
		t.Error("expected product to be active")
	}
}

func TestDeactivateProduct_SetsActiveFalse(t *testing.T) {
	svc := newService()
	created, _ := svc.CreateProduct(context.Background(), validProduct())

	if err := svc.DeactivateProduct(context.Background(), created.ID); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := svc.GetProductByID(context.Background(), created.ID)
	if got.Active {
		t.Error("expected product to be inactive")
	}
}
