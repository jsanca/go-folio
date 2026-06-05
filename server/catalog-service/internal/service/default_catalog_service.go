package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
)

const (
	defaultProjectionPageSize = 100
	maxProjectionPageSize     = 500
	defaultInventoryPageSize  = 500
	maxInventoryPageSize      = 1000
)

type defaultCatalogService struct {
	db       *sql.DB
	products repository.CatalogProductRepository
	variants repository.ProductVariantRepository
	images   repository.ProductImageRepository
	syncRepo repository.CatalogSyncRepository
}

// NewCatalogService creates a CatalogService backed by the given repositories.
// db is used exclusively to begin transactions for mutating operations.
func NewCatalogService(
	db *sql.DB,
	products repository.CatalogProductRepository,
	variants repository.ProductVariantRepository,
	images repository.ProductImageRepository,
	syncRepo repository.CatalogSyncRepository,
) CatalogService {
	return &defaultCatalogService{
		db:       db,
		products: products,
		variants: variants,
		images:   images,
		syncRepo: syncRepo,
	}
}

// inTx begins a transaction, binds the product repository to it, runs fn, and commits.
func (s *defaultCatalogService) inTx(ctx context.Context, fn func(repository.CatalogProductRepository) (*domain.Product, error)) (*domain.Product, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	result, err := fn(s.products.WithTx(tx))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

// inTxVoid is inTx for operations that return only an error.
func (s *defaultCatalogService) inTxVoid(ctx context.Context, fn func(repository.CatalogProductRepository) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err = fn(s.products.WithTx(tx)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ── Product ──────────────────────────────────────────────────────────────────

func (s *defaultCatalogService) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
	}
	return s.inTx(ctx, func(r repository.CatalogProductRepository) (*domain.Product, error) {
		return r.CreateProduct(ctx, p)
	})
}

func (s *defaultCatalogService) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return s.products.GetProductByID(ctx, id)
}

func (s *defaultCatalogService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	return s.products.GetProductBySlug(ctx, slug)
}

func (s *defaultCatalogService) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.products.ListProducts(ctx)
}

// UpdateProduct applies non-nil fields from update onto the existing product, validates, and persists.
// A SELECT FOR UPDATE is used on the initial fetch to prevent races between concurrent edits.
func (s *defaultCatalogService) UpdateProduct(ctx context.Context, id int64, update ProductUpdate) (*domain.Product, error) {
	return s.inTx(ctx, func(r repository.CatalogProductRepository) (*domain.Product, error) {
		existing, err := r.GetProductByIDForUpdate(ctx, id)
		if err != nil {
			return nil, err
		}
		if update.ProductCode != nil {
			existing.ProductCode = *update.ProductCode
		}
		if update.Title != nil {
			existing.Title = *update.Title
		}
		if update.Slug != nil {
			existing.Slug = *update.Slug
		}
		if update.ShortDescription != nil {
			existing.ShortDescription = *update.ShortDescription
		}
		if update.Department != nil {
			existing.Department = *update.Department
		}
		if update.Category != nil {
			existing.Category = *update.Category
		}
		if update.PrimaryImageURL != nil {
			existing.PrimaryImageURL = *update.PrimaryImageURL
		}
		if update.Active != nil {
			existing.Active = *update.Active
		}
		if err := existing.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
		}
		return r.UpdateProduct(ctx, id, existing)
	})
}

// DeleteProduct removes the product with the given id.
func (s *defaultCatalogService) DeleteProduct(ctx context.Context, id int64) error {
	return s.inTxVoid(ctx, func(r repository.CatalogProductRepository) error {
		return r.DeleteProduct(ctx, id)
	})
}

// ── Variant ──────────────────────────────────────────────────────────────────

func (s *defaultCatalogService) AddVariantToProduct(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidVariant, err)
	}
	return s.variants.AddVariant(ctx, v)
}

func (s *defaultCatalogService) GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	return s.variants.GetVariantBySKU(ctx, sku)
}

func (s *defaultCatalogService) UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error {
	if retail.AmountCents < 0 {
		return fmt.Errorf("%w: retail price must not be negative", ErrInvalidVariant)
	}
	if sale != nil && sale.AmountCents < 0 {
		return fmt.Errorf("%w: sale price must not be negative", ErrInvalidVariant)
	}
	if currency == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidVariant)
	}
	return s.variants.UpdateVariantPricing(ctx, sku, retail, sale, currency)
}

func (s *defaultCatalogService) ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error) {
	return s.variants.ListVariantsByProductID(ctx, productID)
}

// ── Image ────────────────────────────────────────────────────────────────────

func (s *defaultCatalogService) AddProductImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error) {
	if err := img.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	return s.images.AddImage(ctx, img)
}

func (s *defaultCatalogService) ListProductImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error) {
	return s.images.ListImagesByProductID(ctx, productID)
}

func (s *defaultCatalogService) ListProductImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error) {
	return s.images.ListImagesByVariantID(ctx, variantID)
}

// ── Sync ──────────────────────────────────────────────────────────────────────

func (s *defaultCatalogService) ListProductProjections(ctx context.Context, q domain.SyncQuery) (*domain.ProductProjectionPage, error) {
	// Capture syncToken before any DB query so concurrent updates are not lost.
	syncToken := time.Now().UTC()

	if q.PageSize <= 0 {
		q.PageSize = defaultProjectionPageSize
	}
	if q.PageSize > maxProjectionPageSize {
		q.PageSize = maxProjectionPageSize
	}

	if q.Cursor != "" {
		c, err := decodeCursor(q.Cursor)
		if err != nil {
			return nil, err
		}
		q.AfterAt = &c.UpdatedAt
		q.AfterID = c.ID
	}

	items, hasNext, err := s.syncRepo.ListProductProjectionPage(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list product projections: %w", err)
	}

	// Guarantee non-nil slices so JSON serializes as [] not null.
	if items == nil {
		items = []domain.ProductProjection{}
	}
	for i := range items {
		if items[i].Variants == nil {
			items[i].Variants = []domain.ProductVariant{}
		}
		if items[i].Images == nil {
			items[i].Images = []domain.ProductImage{}
		}
	}

	var nextCursor string
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCursor(domain.SyncCursor{
			UpdatedAt: last.Product.UpdatedAt,
			ID:        last.Product.ID,
		})
	}

	return &domain.ProductProjectionPage{
		Items:      items,
		PageSize:   q.PageSize,
		NextCursor: nextCursor,
		HasNext:    hasNext,
		SyncToken:  syncToken,
	}, nil
}

func (s *defaultCatalogService) ListVariantInventory(ctx context.Context, q domain.SyncQuery) (*domain.VariantInventoryPage, error) {
	syncToken := time.Now().UTC()

	if q.PageSize <= 0 {
		q.PageSize = defaultInventoryPageSize
	}
	if q.PageSize > maxInventoryPageSize {
		q.PageSize = maxInventoryPageSize
	}

	if q.Cursor != "" {
		c, err := decodeCursor(q.Cursor)
		if err != nil {
			return nil, err
		}
		q.AfterAt = &c.UpdatedAt
		q.AfterID = c.ID
	}

	items, hasNext, err := s.syncRepo.ListVariantInventoryPage(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list variant inventory: %w", err)
	}

	if items == nil {
		items = []domain.VariantInventoryRecord{}
	}

	var nextCursor string
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = encodeCursor(domain.SyncCursor{
			UpdatedAt: last.UpdatedAt,
			ID:        last.VariantID,
		})
	}

	return &domain.VariantInventoryPage{
		Items:      items,
		PageSize:   q.PageSize,
		NextCursor: nextCursor,
		HasNext:    hasNext,
		SyncToken:  syncToken,
	}, nil
}
