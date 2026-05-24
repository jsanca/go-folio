package repository

import (
	"context"

	"github.com/leatherstore/catalog-service/internal/domain"
)

// ProductRepository defines the persistence contract for LeatherProduct.
type ProductRepository interface {
	FindByID(ctx context.Context, id int64) (*domain.LeatherProduct, error)
	FindBySKU(ctx context.Context, sku string) (*domain.LeatherProduct, error)
	Save(ctx context.Context, p *domain.LeatherProduct) error
	Update(ctx context.Context, p *domain.LeatherProduct) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]domain.LeatherProduct, error)
}
