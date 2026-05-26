package service

import (
	"context"

	"github.com/jsanca/go-folio/internal/domain"
)

// ProductService defines the application-level use cases for products.
type ProductService interface {
	CreateProduct(ctx context.Context, product *domain.LeatherProduct) (*domain.LeatherProduct, error)
	UpdateProduct(ctx context.Context, product *domain.LeatherProduct) (*domain.LeatherProduct, error)
	GetProductByID(ctx context.Context, id int64) (*domain.LeatherProduct, error)
	GetProductBySKU(ctx context.Context, sku string) (*domain.LeatherProduct, error)
	ListProducts(ctx context.Context) ([]domain.LeatherProduct, error)
	DeleteProduct(ctx context.Context, id int64) error
	ActivateProduct(ctx context.Context, id int64) error
	DeactivateProduct(ctx context.Context, id int64) error
	UpdateInventory(ctx context.Context, sku string, quantity int, stockStatus domain.StockStatus) (*domain.LeatherProduct, error)
	UpdatePricing(ctx context.Context, sku string, retailPrice domain.Money, salePrice *domain.Money, currency string) (*domain.LeatherProduct, error)
}
