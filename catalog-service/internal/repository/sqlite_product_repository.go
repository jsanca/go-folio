package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jsanca/go-folio/internal/domain"
)

type SQLiteProductRepository struct {
	db *sql.DB
}

func NewSQLiteProductRepository(db *sql.DB) *SQLiteProductRepository {
	return &SQLiteProductRepository{db: db}
}

const productColumns = `
	id, sku, external_system_id, title, slug, short_description, description,
	category, tags, main_image_url, retail_price_cents, sale_price_cents, currency,
	stock_quantity, stock_status, warehouse_code, active,
	created_at, updated_at, last_synced_at`

// mapSQLiteError translates known SQLite constraint errors into semantic repository errors.
func mapSQLiteError(err error, op string) error {
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed: products.sku") {
		return ErrDuplicateSKU
	}
	if strings.Contains(msg, "UNIQUE constraint failed: products.slug") {
		return ErrDuplicateSlug
	}
	return fmt.Errorf("%s: %w", op, err)
}

func scanProduct(row interface {
	Scan(dest ...any) error
}) (*domain.LeatherProduct, error) {
	var p domain.LeatherProduct
	var tagsJSON string
	var salePriceCents sql.NullInt64
	var lastSyncedAt sql.NullTime

	err := row.Scan(
		&p.ID,
		&p.SKU,
		&p.ExternalSystemID,
		&p.Title,
		&p.Slug,
		&p.ShortDescription,
		&p.Description,
		&p.Category,
		&tagsJSON,
		&p.MainImageURL,
		&p.RetailPrice.AmountCents,
		&salePriceCents,
		&p.Currency,
		&p.StockQuantity,
		&p.StockStatus,
		&p.WarehouseCode,
		&p.Active,
		&p.CreatedAt,
		&p.UpdatedAt,
		&lastSyncedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(tagsJSON), &p.Tags); err != nil || p.Tags == nil {
		p.Tags = []string{}
	}
	if salePriceCents.Valid {
		p.SalePrice = &domain.Money{AmountCents: salePriceCents.Int64}
	}
	if lastSyncedAt.Valid {
		t := lastSyncedAt.Time
		p.LastSyncedAt = &t
	}

	return &p, nil
}

func (r *SQLiteProductRepository) FindByID(ctx context.Context, id int64) (*domain.LeatherProduct, error) {
	query := fmt.Sprintf("SELECT %s FROM products WHERE id = ?", productColumns)
	p, err := scanProduct(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find product by id: %w", err)
	}
	return p, nil
}

func (r *SQLiteProductRepository) FindBySKU(ctx context.Context, sku string) (*domain.LeatherProduct, error) {
	query := fmt.Sprintf("SELECT %s FROM products WHERE sku = ?", productColumns)
	p, err := scanProduct(r.db.QueryRowContext(ctx, query, sku))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find product by sku: %w", err)
	}
	return p, nil
}

func (r *SQLiteProductRepository) Save(ctx context.Context, p *domain.LeatherProduct) error {
	tagsJSON, err := json.Marshal(p.Tags)
	if err != nil {
		return fmt.Errorf("save product: marshal tags: %w", err)
	}

	var salePriceCents *int64
	if p.SalePrice != nil {
		salePriceCents = &p.SalePrice.AmountCents
	}

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	query := `
		INSERT INTO products (
			sku, external_system_id, title, slug, short_description, description,
			category, tags, main_image_url, retail_price_cents, sale_price_cents, currency,
			stock_quantity, stock_status, warehouse_code, active, last_synced_at,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?
		)`

	result, err := r.db.ExecContext(ctx, query,
		p.SKU, p.ExternalSystemID, p.Title, p.Slug, p.ShortDescription, p.Description,
		p.Category, string(tagsJSON), p.MainImageURL, p.RetailPrice.AmountCents, salePriceCents, p.Currency,
		p.StockQuantity, p.StockStatus, p.WarehouseCode, p.Active, p.LastSyncedAt,
		now, now,
	)
	if err != nil {
		return mapSQLiteError(err, "save product")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("save product: %w", err)
	}
	p.ID = id
	return nil
}

func (r *SQLiteProductRepository) Update(ctx context.Context, p *domain.LeatherProduct) error {
	tagsJSON, err := json.Marshal(p.Tags)
	if err != nil {
		return fmt.Errorf("update product: marshal tags: %w", err)
	}

	var salePriceCents *int64
	if p.SalePrice != nil {
		salePriceCents = &p.SalePrice.AmountCents
	}

	p.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE products SET
			sku = ?, external_system_id = ?, title = ?, slug = ?,
			short_description = ?, description = ?, category = ?, tags = ?,
			main_image_url = ?, retail_price_cents = ?, sale_price_cents = ?, currency = ?,
			stock_quantity = ?, stock_status = ?, warehouse_code = ?,
			active = ?, last_synced_at = ?, updated_at = ?
		WHERE id = ?`

	_, err = r.db.ExecContext(ctx, query,
		p.SKU, p.ExternalSystemID, p.Title, p.Slug,
		p.ShortDescription, p.Description, p.Category, string(tagsJSON),
		p.MainImageURL, p.RetailPrice.AmountCents, salePriceCents, p.Currency,
		p.StockQuantity, p.StockStatus, p.WarehouseCode,
		p.Active, p.LastSyncedAt, p.UpdatedAt,
		p.ID,
	)
	if err != nil {
		return mapSQLiteError(err, "update product")
	}
	return nil
}

func (r *SQLiteProductRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

func (r *SQLiteProductRepository) List(ctx context.Context, limit, offset int) ([]domain.LeatherProduct, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM products ORDER BY id ASC LIMIT ? OFFSET ?",
		productColumns,
	)

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []domain.LeatherProduct
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("list products scan: %w", err)
		}
		products = append(products, *p)
	}
	return products, rows.Err()
}
