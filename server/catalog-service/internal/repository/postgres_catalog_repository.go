package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jsanca/go-folio/internal/domain"
)

// querier is the minimal interface shared by *sql.DB and *sql.Tx,
// allowing product functions to execute within or outside a transaction.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// PostgresCatalogRepository implements CatalogProductRepository,
// ProductVariantRepository, ProductImageRepository, and CatalogSyncRepository
// over a PostgreSQL database.
type PostgresCatalogRepository struct {
	db *sql.DB
}

// NewSQLiteCatalogRepository creates a PostgresCatalogRepository backed by the given connection.
// The name is kept for backwards compatibility with existing call sites.
func NewSQLiteCatalogRepository(db *sql.DB) *PostgresCatalogRepository {
	return &PostgresCatalogRepository{db: db}
}

// ── CatalogProductRepository ────────────────────────────────────────────────

// CreateProduct inserts a new product and returns the persisted record.
func (r *PostgresCatalogRepository) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	return createProduct(ctx, r.db, p)
}

// GetProductByID returns the product with the given id.
func (r *PostgresCatalogRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return getProductByID(ctx, r.db, id)
}

// GetProductByIDForUpdate returns the product with the given id and acquires a row lock (SELECT … FOR UPDATE).
func (r *PostgresCatalogRepository) GetProductByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error) {
	return getProductByIDForUpdate(ctx, r.db, id)
}

// UpdateProduct updates all mutable fields of the product identified by id and returns the persisted record.
func (r *PostgresCatalogRepository) UpdateProduct(ctx context.Context, id int64, p *domain.Product) (*domain.Product, error) {
	return updateProduct(ctx, r.db, id, p)
}

// DeleteProduct removes the product identified by id.
func (r *PostgresCatalogRepository) DeleteProduct(ctx context.Context, id int64) error {
	return deleteProduct(ctx, r.db, id)
}

// ListProducts returns all products ordered by id.
func (r *PostgresCatalogRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return listProducts(ctx, r.db)
}

// WithTx returns a CatalogProductRepository scoped to the given transaction.
func (r *PostgresCatalogRepository) WithTx(tx *sql.Tx) CatalogProductRepository {
	return &pgTxCatalogRepository{tx: tx}
}

// ── pgTxCatalogRepository ────────────────────────────────────────────────────

// pgTxCatalogRepository is a transaction-scoped CatalogProductRepository.
type pgTxCatalogRepository struct {
	tx *sql.Tx
}

// CreateProduct inserts a new product within the transaction and returns the persisted record.
func (r *pgTxCatalogRepository) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	return createProduct(ctx, r.tx, p)
}

// GetProductByID returns the product with the given id within the transaction.
func (r *pgTxCatalogRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return getProductByID(ctx, r.tx, id)
}

// GetProductByIDForUpdate returns the product with the given id and acquires a row lock within the transaction.
func (r *pgTxCatalogRepository) GetProductByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error) {
	return getProductByIDForUpdate(ctx, r.tx, id)
}

// UpdateProduct updates all mutable fields of the product identified by id within the transaction.
func (r *pgTxCatalogRepository) UpdateProduct(ctx context.Context, id int64, p *domain.Product) (*domain.Product, error) {
	return updateProduct(ctx, r.tx, id, p)
}

// DeleteProduct removes the product identified by id within the transaction.
func (r *pgTxCatalogRepository) DeleteProduct(ctx context.Context, id int64) error {
	return deleteProduct(ctx, r.tx, id)
}

// ListProducts returns all products ordered by id within the transaction.
func (r *pgTxCatalogRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return listProducts(ctx, r.tx)
}

// WithTx returns a new transaction-scoped CatalogProductRepository bound to tx.
func (r *pgTxCatalogRepository) WithTx(tx *sql.Tx) CatalogProductRepository {
	return &pgTxCatalogRepository{tx: tx}
}

// ── package-level product functions ──────────────────────────────────────────

func createProduct(ctx context.Context, q querier, p *domain.Product) (*domain.Product, error) {
	tags, err := json.Marshal(tagsOrEmpty(p.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal tags: %w", err)
	}

	const query = `
		INSERT INTO catalog_products
			(product_code, external_product_id, title, slug, short_description,
			 description, additional_info, department, category, subcategory,
			 tags, base_sku, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`

	var id int64
	err = q.QueryRowContext(ctx, query,
		p.ProductCode, nullableString(p.ExternalProductID), p.Title, p.Slug,
		p.ShortDescription, p.Description, p.AdditionalInfo,
		p.Department, p.Category, p.Subcategory,
		string(tags), nullableString(p.BaseSKU), p.Active,
	).Scan(&id)
	if err != nil {
		return nil, mapCatalogPgError(err, "create product")
	}
	return getProductByID(ctx, q, id)
}

func getProductByID(ctx context.Context, q querier, id int64) (*domain.Product, error) {
	const query = `
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products WHERE id = $1`

	row := q.QueryRowContext(ctx, query, id)
	p, err := scanCatalogProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	return p, err
}

func getProductByIDForUpdate(ctx context.Context, q querier, id int64) (*domain.Product, error) {
	const query = `
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products WHERE id = $1 FOR UPDATE`

	row := q.QueryRowContext(ctx, query, id)
	p, err := scanCatalogProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	return p, err
}

func updateProduct(ctx context.Context, q querier, id int64, p *domain.Product) (*domain.Product, error) {
	tags, err := json.Marshal(tagsOrEmpty(p.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal tags: %w", err)
	}

	const query = `
		UPDATE catalog_products SET
		    product_code = $1, title = $2, slug = $3,
		    short_description = $4, description = $5, additional_info = $6,
		    department = $7, category = $8, subcategory = $9,
		    tags = $10, base_sku = $11, active = $12, updated_at = NOW()
		WHERE id = $13`

	res, err := q.ExecContext(ctx, query,
		p.ProductCode, p.Title, p.Slug,
		p.ShortDescription, p.Description, p.AdditionalInfo,
		p.Department, p.Category, p.Subcategory,
		string(tags), nullableString(p.BaseSKU), p.Active, id,
	)
	if err != nil {
		return nil, mapCatalogPgError(err, "update product")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrProductNotFound
	}
	return getProductByID(ctx, q, id)
}

func deleteProduct(ctx context.Context, q querier, id int64) error {
	const query = `DELETE FROM catalog_products WHERE id = $1`
	res, err := q.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProductNotFound
	}
	return nil
}

func listProducts(ctx context.Context, q querier) ([]domain.Product, error) {
	const query = `
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products ORDER BY id`

	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		p, err := scanCatalogProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *p)
	}
	return products, rows.Err()
}

// ── ProductVariantRepository ─────────────────────────────────────────────────

// AddVariant inserts a new variant and returns the persisted record.
func (r *PostgresCatalogRepository) AddVariant(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	const q = `
		INSERT INTO product_variants
			(product_id, sku, external_variant_id, color_slug, color_name,
			 primary_color_name, secondary_color_name, primary_color_hex, secondary_color_hex,
			 retail_price_cents, sale_price_cents, currency,
			 stock_quantity, stock_status, warehouse_code, variant_image_url, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		v.ProductID, v.SKU, nullableString(v.ExternalVariantID),
		v.ColorSlug, v.ColorName, v.PrimaryColorName, v.SecondaryColorName,
		v.PrimaryColorHex, v.SecondaryColorHex,
		v.RetailPrice.AmountCents, nullableInt64(v.SalePrice),
		v.Currency, v.StockQuantity, string(v.StockStatus),
		v.WarehouseCode, v.VariantImageURL, v.Active,
	).Scan(&id)
	if err != nil {
		return nil, mapCatalogPgError(err, "add variant")
	}
	return r.GetVariantByID(ctx, id)
}

// GetVariantBySKU returns the variant with the given SKU.
func (r *PostgresCatalogRepository) GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE sku = $1`

	row := r.db.QueryRowContext(ctx, q, sku)
	v, err := scanCatalogVariant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVariantNotFound
	}
	return v, err
}

// GetVariantByID returns the variant with the given id.
func (r *PostgresCatalogRepository) GetVariantByID(ctx context.Context, id int64) (*domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE id = $1`

	row := r.db.QueryRowContext(ctx, q, id)
	v, err := scanCatalogVariant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVariantNotFound
	}
	return v, err
}

// UpdateVariantPricing updates price fields for the variant identified by sku.
func (r *PostgresCatalogRepository) UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error {
	const q = `UPDATE product_variants SET retail_price_cents = $1, sale_price_cents = $2, currency = $3, updated_at = $4 WHERE sku = $5`
	res, err := r.db.ExecContext(ctx, q, retail.AmountCents, nullableInt64(sale), currency, time.Now().UTC(), sku)
	if err != nil {
		return fmt.Errorf("update pricing: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrVariantNotFound
	}
	return nil
}

// ListVariantsByProductID returns all variants for the given product, ordered by id.
func (r *PostgresCatalogRepository) ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE product_id = $1 ORDER BY id`

	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var variants []domain.ProductVariant
	for rows.Next() {
		v, err := scanCatalogVariant(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, *v)
	}
	return variants, rows.Err()
}

// ── ProductImageRepository ───────────────────────────────────────────────────

// AddImage inserts a new image and returns the persisted record.
func (r *PostgresCatalogRepository) AddImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error) {
	const q = `
		INSERT INTO product_images (product_id, variant_id, url, alt_text, sort_order, is_main, width, height)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		img.ProductID, img.VariantID, img.URL, img.AltText,
		img.SortOrder, img.IsMain, img.Width, img.Height,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("add image: %w", err)
	}
	saved := *img
	saved.ID = id
	return &saved, nil
}

// ListImagesByProductID returns all images for the given product, ordered by sort_order then id.
func (r *PostgresCatalogRepository) ListImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error) {
	const q = `
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE product_id = $1 ORDER BY sort_order, id`
	return r.queryImages(ctx, q, productID)
}

// ListImagesByVariantID returns all images for the given variant, ordered by sort_order then id.
func (r *PostgresCatalogRepository) ListImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error) {
	const q = `
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE variant_id = $1 ORDER BY sort_order, id`
	return r.queryImages(ctx, q, variantID)
}

func (r *PostgresCatalogRepository) queryImages(ctx context.Context, q string, arg any) ([]domain.ProductImage, error) {
	rows, err := r.db.QueryContext(ctx, q, arg)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}
	defer rows.Close()

	var images []domain.ProductImage
	for rows.Next() {
		img, err := scanCatalogImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, *img)
	}
	return images, rows.Err()
}

// ── CatalogSyncRepository ────────────────────────────────────────────────────

// ListProductProjectionPage returns one cursor-paginated page of product projections.
func (r *PostgresCatalogRepository) ListProductProjectionPage(ctx context.Context, q domain.SyncQuery) ([]domain.ProductProjection, bool, error) {
	var conds []string
	var args []any
	n := 1

	if q.UpdatedSince != nil {
		conds = append(conds, fmt.Sprintf("updated_at > $%d", n))
		args = append(args, q.UpdatedSince.UTC())
		n++
	}
	if q.AfterAt != nil {
		conds = append(conds, fmt.Sprintf("(updated_at > $%d OR (updated_at = $%d AND id > $%d))", n, n+1, n+2))
		args = append(args, q.AfterAt.UTC(), q.AfterAt.UTC(), q.AfterID)
		n += 3
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, q.PageSize+1)

	query := fmt.Sprintf(`
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products%s
		ORDER BY updated_at ASC, id ASC LIMIT $%d`, where, n)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list product projection page: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, err := scanCatalogProduct(rows)
		if err != nil {
			return nil, false, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNext := len(products) > q.PageSize
	if hasNext {
		products = products[:q.PageSize]
	}

	if len(products) == 0 {
		return []domain.ProductProjection{}, false, nil
	}

	ids := make([]int64, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}

	variantsByProduct, err := r.loadVariantsForProducts(ctx, ids)
	if err != nil {
		return nil, false, err
	}
	imagesByProduct, err := r.loadImagesForProducts(ctx, ids)
	if err != nil {
		return nil, false, err
	}

	projections := make([]domain.ProductProjection, len(products))
	for i, p := range products {
		variants := variantsByProduct[p.ID]
		if variants == nil {
			variants = []domain.ProductVariant{}
		}
		images := imagesByProduct[p.ID]
		if images == nil {
			images = []domain.ProductImage{}
		}
		projections[i] = domain.ProductProjection{
			Product:  *p,
			Variants: variants,
			Images:   images,
		}
	}
	return projections, hasNext, nil
}

// ListVariantInventoryPage returns one cursor-paginated page of variant inventory records.
func (r *PostgresCatalogRepository) ListVariantInventoryPage(ctx context.Context, q domain.SyncQuery) ([]domain.VariantInventoryRecord, bool, error) {
	var conds []string
	var args []any
	n := 1

	if q.UpdatedSince != nil {
		conds = append(conds, fmt.Sprintf("pv.updated_at > $%d", n))
		args = append(args, q.UpdatedSince.UTC())
		n++
	}
	if q.AfterAt != nil {
		conds = append(conds, fmt.Sprintf("(pv.updated_at > $%d OR (pv.updated_at = $%d AND pv.id > $%d))", n, n+1, n+2))
		args = append(args, q.AfterAt.UTC(), q.AfterAt.UTC(), q.AfterID)
		n += 3
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, q.PageSize+1)

	query := fmt.Sprintf(`
		SELECT cp.product_code, pv.product_id, pv.id, pv.sku,
		       pv.retail_price_cents, pv.sale_price_cents, pv.currency,
		       pv.stock_quantity, pv.stock_status, pv.active,
		       pv.updated_at, pv.last_synced_at
		FROM product_variants pv
		JOIN catalog_products cp ON cp.id = pv.product_id%s
		ORDER BY pv.updated_at ASC, pv.id ASC LIMIT $%d`, where, n)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list variant inventory page: %w", err)
	}
	defer rows.Close()

	var records []domain.VariantInventoryRecord
	for rows.Next() {
		var rec domain.VariantInventoryRecord
		var saleCents sql.NullInt64
		var lastSynced sql.NullTime

		if err := rows.Scan(
			&rec.ProductCode, &rec.ProductID, &rec.VariantID, &rec.SKU,
			&rec.RetailPrice.AmountCents, &saleCents, &rec.Currency,
			&rec.StockQuantity, &rec.StockStatus, &rec.Active,
			&rec.UpdatedAt, &lastSynced,
		); err != nil {
			return nil, false, err
		}
		if saleCents.Valid {
			m := domain.Money{AmountCents: saleCents.Int64}
			rec.SalePrice = &m
		}
		if lastSynced.Valid {
			t := lastSynced.Time
			rec.LastSyncedAt = &t
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNext := len(records) > q.PageSize
	if hasNext {
		records = records[:q.PageSize]
	}
	if records == nil {
		records = []domain.VariantInventoryRecord{}
	}
	return records, hasNext, nil
}

func (r *PostgresCatalogRepository) loadVariantsForProducts(ctx context.Context, ids []int64) (map[int64][]domain.ProductVariant, error) {
	query := fmt.Sprintf(`
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE product_id IN %s ORDER BY id`, pgInClause(1, len(ids)))

	rows, err := r.db.QueryContext(ctx, query, int64sToAny(ids)...)
	if err != nil {
		return nil, fmt.Errorf("load variants for products: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]domain.ProductVariant)
	for rows.Next() {
		v, err := scanCatalogVariant(rows)
		if err != nil {
			return nil, err
		}
		result[v.ProductID] = append(result[v.ProductID], *v)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogRepository) loadImagesForProducts(ctx context.Context, ids []int64) (map[int64][]domain.ProductImage, error) {
	query := fmt.Sprintf(`
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE product_id IN %s ORDER BY sort_order, id`, pgInClause(1, len(ids)))

	rows, err := r.db.QueryContext(ctx, query, int64sToAny(ids)...)
	if err != nil {
		return nil, fmt.Errorf("load images for products: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]domain.ProductImage)
	for rows.Next() {
		img, err := scanCatalogImage(rows)
		if err != nil {
			return nil, err
		}
		result[img.ProductID] = append(result[img.ProductID], *img)
	}
	return result, rows.Err()
}

// pgInClause builds a PostgreSQL IN clause starting at parameter $start.
// e.g. pgInClause(1, 3) → "($1,$2,$3)"
func pgInClause(start, n int) string {
	placeholders := make([]string, n)
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", start+i)
	}
	return "(" + strings.Join(placeholders, ",") + ")"
}

func int64sToAny(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// ── scan helpers ─────────────────────────────────────────────────────────────

type catalogScanner interface {
	Scan(dest ...any) error
}

func scanCatalogProduct(s catalogScanner) (*domain.Product, error) {
	var p domain.Product
	var externalID sql.NullString
	var baseSKU sql.NullString
	var tagsJSON string
	var lastSynced sql.NullTime

	err := s.Scan(
		&p.ID, &p.ProductCode, &externalID, &p.Title, &p.Slug,
		&p.ShortDescription, &p.Description, &p.AdditionalInfo,
		&p.Department, &p.Category, &p.Subcategory, &tagsJSON, &baseSKU,
		&p.Active, &p.CreatedAt, &p.UpdatedAt, &lastSynced,
	)
	if err != nil {
		return nil, err
	}

	p.ExternalProductID = externalID.String
	p.BaseSKU = baseSKU.String
	if lastSynced.Valid {
		t := lastSynced.Time
		p.LastSyncedAt = &t
	}
	if err := json.Unmarshal([]byte(tagsJSON), &p.Tags); err != nil {
		p.Tags = nil
	}
	return &p, nil
}

func scanCatalogVariant(s catalogScanner) (*domain.ProductVariant, error) {
	var v domain.ProductVariant
	var externalID sql.NullString
	var saleCents sql.NullInt64
	var lastSynced sql.NullTime

	err := s.Scan(
		&v.ID, &v.ProductID, &v.SKU, &externalID,
		&v.ColorSlug, &v.ColorName, &v.PrimaryColorName, &v.SecondaryColorName,
		&v.PrimaryColorHex, &v.SecondaryColorHex,
		&v.RetailPrice.AmountCents, &saleCents, &v.Currency,
		&v.StockQuantity, &v.StockStatus, &v.WarehouseCode, &v.VariantImageURL,
		&v.Active, &v.CreatedAt, &v.UpdatedAt, &lastSynced,
	)
	if err != nil {
		return nil, err
	}

	v.ExternalVariantID = externalID.String
	if saleCents.Valid {
		m := domain.Money{AmountCents: saleCents.Int64}
		v.SalePrice = &m
	}
	if lastSynced.Valid {
		t := lastSynced.Time
		v.LastSyncedAt = &t
	}
	return &v, nil
}

func scanCatalogImage(s catalogScanner) (*domain.ProductImage, error) {
	var img domain.ProductImage
	var variantID sql.NullInt64
	var width, height sql.NullInt64

	err := s.Scan(
		&img.ID, &img.ProductID, &variantID, &img.URL, &img.AltText,
		&img.SortOrder, &img.IsMain, &width, &height,
		&img.CreatedAt, &img.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if variantID.Valid {
		id := variantID.Int64
		img.VariantID = &id
	}
	if width.Valid {
		w := int(width.Int64)
		img.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		img.Height = &h
	}
	return &img, nil
}

// ── utility helpers ───────────────────────────────────────────────────────────

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt64(m *domain.Money) any {
	if m == nil {
		return nil
	}
	return m.AmountCents
}

func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func mapCatalogPgError(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "catalog_products_product_code_key":
			return ErrDuplicateProductCode
		case "catalog_products_slug_key":
			return ErrDuplicateSlug
		case "product_variants_sku_key":
			return ErrDuplicateSKU
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
