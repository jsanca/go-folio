package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jsanca/go-folio/internal/domain"
)

// SQLiteCatalogRepository implements CatalogProductRepository,
// ProductVariantRepository, and ProductImageRepository over SQLite.
type SQLiteCatalogRepository struct {
	db *sql.DB
}

func NewSQLiteCatalogRepository(db *sql.DB) *SQLiteCatalogRepository {
	return &SQLiteCatalogRepository{db: db}
}

// ── CatalogProductRepository ────────────────────────────────────────────────

func (r *SQLiteCatalogRepository) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	tags, err := json.Marshal(tagsOrEmpty(p.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal tags: %w", err)
	}

	const q = `
		INSERT INTO catalog_products
			(product_code, external_product_id, title, slug, short_description,
			 description, additional_info, department, category, subcategory,
			 tags, base_sku, active)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`

	res, err := r.db.ExecContext(ctx, q,
		p.ProductCode, nullableString(p.ExternalProductID), p.Title, p.Slug,
		p.ShortDescription, p.Description, p.AdditionalInfo,
		p.Department, p.Category, p.Subcategory,
		string(tags), nullableString(p.BaseSKU), boolToInt(p.Active),
	)
	if err != nil {
		return nil, mapCatalogSQLiteError(err, "create product")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetProductByID(ctx, id)
}

func (r *SQLiteCatalogRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	const q = `
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products WHERE id = ?`

	row := r.db.QueryRowContext(ctx, q, id)
	p, err := scanCatalogProduct(row)
	if err == sql.ErrNoRows {
		return nil, ErrProductNotFound
	}
	return p, err
}

func (r *SQLiteCatalogRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	const q = `
		SELECT id, product_code, external_product_id, title, slug,
		       short_description, description, additional_info,
		       department, category, subcategory, tags, base_sku,
		       active, created_at, updated_at, last_synced_at
		FROM catalog_products ORDER BY id`

	rows, err := r.db.QueryContext(ctx, q)
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

func (r *SQLiteCatalogRepository) AddVariant(ctx context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	const q = `
		INSERT INTO product_variants
			(product_id, sku, external_variant_id, color_slug, color_name,
			 primary_color_name, secondary_color_name, primary_color_hex, secondary_color_hex,
			 retail_price_cents, sale_price_cents, currency,
			 stock_quantity, stock_status, warehouse_code, variant_image_url, active)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	res, err := r.db.ExecContext(ctx, q,
		v.ProductID, v.SKU, nullableString(v.ExternalVariantID),
		v.ColorSlug, v.ColorName, v.PrimaryColorName, v.SecondaryColorName,
		v.PrimaryColorHex, v.SecondaryColorHex,
		v.RetailPrice.AmountCents, nullableInt64(v.SalePrice),
		v.Currency, v.StockQuantity, string(v.StockStatus),
		v.WarehouseCode, v.VariantImageURL, boolToInt(v.Active),
	)
	if err != nil {
		return nil, mapCatalogSQLiteError(err, "add variant")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetVariantByID(ctx, id)
}

func (r *SQLiteCatalogRepository) GetVariantBySKU(ctx context.Context, sku string) (*domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE sku = ?`

	row := r.db.QueryRowContext(ctx, q, sku)
	v, err := scanCatalogVariant(row)
	if err == sql.ErrNoRows {
		return nil, ErrVariantNotFound
	}
	return v, err
}

func (r *SQLiteCatalogRepository) GetVariantByID(ctx context.Context, id int64) (*domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE id = ?`

	row := r.db.QueryRowContext(ctx, q, id)
	v, err := scanCatalogVariant(row)
	if err == sql.ErrNoRows {
		return nil, ErrVariantNotFound
	}
	return v, err
}

func (r *SQLiteCatalogRepository) UpdateVariantInventory(ctx context.Context, sku string, qty int, status domain.StockStatus) error {
	const q = `UPDATE product_variants SET stock_quantity = ?, stock_status = ?, updated_at = ? WHERE sku = ?`
	res, err := r.db.ExecContext(ctx, q, qty, string(status), time.Now().UTC(), sku)
	if err != nil {
		return fmt.Errorf("update inventory: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrVariantNotFound
	}
	return nil
}

func (r *SQLiteCatalogRepository) UpdateVariantPricing(ctx context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error {
	const q = `UPDATE product_variants SET retail_price_cents = ?, sale_price_cents = ?, currency = ?, updated_at = ? WHERE sku = ?`
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

func (r *SQLiteCatalogRepository) ListVariantsByProductID(ctx context.Context, productID int64) ([]domain.ProductVariant, error) {
	const q = `
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE product_id = ? ORDER BY id`

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

func (r *SQLiteCatalogRepository) AddImage(ctx context.Context, img *domain.ProductImage) (*domain.ProductImage, error) {
	const q = `
		INSERT INTO product_images (product_id, variant_id, url, alt_text, sort_order, is_main, width, height)
		VALUES (?,?,?,?,?,?,?,?)`

	res, err := r.db.ExecContext(ctx, q,
		img.ProductID, img.VariantID, img.URL, img.AltText,
		img.SortOrder, boolToInt(img.IsMain), img.Width, img.Height,
	)
	if err != nil {
		return nil, fmt.Errorf("add image: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	saved := *img
	saved.ID = id
	return &saved, nil
}

func (r *SQLiteCatalogRepository) ListImagesByProductID(ctx context.Context, productID int64) ([]domain.ProductImage, error) {
	const q = `
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE product_id = ? ORDER BY sort_order, id`
	return r.queryImages(ctx, q, productID)
}

func (r *SQLiteCatalogRepository) ListImagesByVariantID(ctx context.Context, variantID int64) ([]domain.ProductImage, error) {
	const q = `
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE variant_id = ? ORDER BY sort_order, id`
	return r.queryImages(ctx, q, variantID)
}

func (r *SQLiteCatalogRepository) queryImages(ctx context.Context, q string, arg any) ([]domain.ProductImage, error) {
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

func (r *SQLiteCatalogRepository) ListProductProjectionPage(ctx context.Context, q domain.SyncQuery) ([]domain.ProductProjection, bool, error) {
	var conds []string
	var args []any

	if q.UpdatedSince != nil {
		conds = append(conds, "updated_at > ?")
		args = append(args, q.UpdatedSince.UTC())
	}
	if q.AfterAt != nil {
		conds = append(conds, "(updated_at > ? OR (updated_at = ? AND id > ?))")
		args = append(args, q.AfterAt.UTC(), q.AfterAt.UTC(), q.AfterID)
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
		ORDER BY updated_at ASC, id ASC LIMIT ?`, where)

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

func (r *SQLiteCatalogRepository) ListVariantInventoryPage(ctx context.Context, q domain.SyncQuery) ([]domain.VariantInventoryRecord, bool, error) {
	var conds []string
	var args []any

	if q.UpdatedSince != nil {
		conds = append(conds, "pv.updated_at > ?")
		args = append(args, q.UpdatedSince.UTC())
	}
	if q.AfterAt != nil {
		conds = append(conds, "(pv.updated_at > ? OR (pv.updated_at = ? AND pv.id > ?))")
		args = append(args, q.AfterAt.UTC(), q.AfterAt.UTC(), q.AfterID)
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
		ORDER BY pv.updated_at ASC, pv.id ASC LIMIT ?`, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list variant inventory page: %w", err)
	}
	defer rows.Close()

	var records []domain.VariantInventoryRecord
	for rows.Next() {
		var rec domain.VariantInventoryRecord
		var saleCents sql.NullInt64
		var activeInt int
		var lastSynced sql.NullTime

		if err := rows.Scan(
			&rec.ProductCode, &rec.ProductID, &rec.VariantID, &rec.SKU,
			&rec.RetailPrice.AmountCents, &saleCents, &rec.Currency,
			&rec.StockQuantity, &rec.StockStatus, &activeInt,
			&rec.UpdatedAt, &lastSynced,
		); err != nil {
			return nil, false, err
		}
		rec.Active = activeInt != 0
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

func (r *SQLiteCatalogRepository) loadVariantsForProducts(ctx context.Context, ids []int64) (map[int64][]domain.ProductVariant, error) {
	query := fmt.Sprintf(`
		SELECT id, product_id, sku, external_variant_id,
		       color_slug, color_name, primary_color_name, secondary_color_name,
		       primary_color_hex, secondary_color_hex,
		       retail_price_cents, sale_price_cents, currency,
		       stock_quantity, stock_status, warehouse_code, variant_image_url,
		       active, created_at, updated_at, last_synced_at
		FROM product_variants WHERE product_id IN %s ORDER BY id`, inClause(len(ids)))

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

func (r *SQLiteCatalogRepository) loadImagesForProducts(ctx context.Context, ids []int64) (map[int64][]domain.ProductImage, error) {
	query := fmt.Sprintf(`
		SELECT id, product_id, variant_id, url, alt_text, sort_order, is_main, width, height, created_at, updated_at
		FROM product_images WHERE product_id IN %s ORDER BY sort_order, id`, inClause(len(ids)))

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

func inClause(n int) string {
	placeholders := make([]string, n)
	for i := range placeholders {
		placeholders[i] = "?"
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
	var activeInt int
	var lastSynced sql.NullTime

	err := s.Scan(
		&p.ID, &p.ProductCode, &externalID, &p.Title, &p.Slug,
		&p.ShortDescription, &p.Description, &p.AdditionalInfo,
		&p.Department, &p.Category, &p.Subcategory, &tagsJSON, &baseSKU,
		&activeInt, &p.CreatedAt, &p.UpdatedAt, &lastSynced,
	)
	if err != nil {
		return nil, err
	}

	p.ExternalProductID = externalID.String
	p.BaseSKU = baseSKU.String
	p.Active = activeInt != 0
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
	var activeInt int
	var lastSynced sql.NullTime

	err := s.Scan(
		&v.ID, &v.ProductID, &v.SKU, &externalID,
		&v.ColorSlug, &v.ColorName, &v.PrimaryColorName, &v.SecondaryColorName,
		&v.PrimaryColorHex, &v.SecondaryColorHex,
		&v.RetailPrice.AmountCents, &saleCents, &v.Currency,
		&v.StockQuantity, &v.StockStatus, &v.WarehouseCode, &v.VariantImageURL,
		&activeInt, &v.CreatedAt, &v.UpdatedAt, &lastSynced,
	)
	if err != nil {
		return nil, err
	}

	v.ExternalVariantID = externalID.String
	v.Active = activeInt != 0
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
	var isMainInt int

	err := s.Scan(
		&img.ID, &img.ProductID, &variantID, &img.URL, &img.AltText,
		&img.SortOrder, &isMainInt, &width, &height,
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
	img.IsMain = isMainInt != 0
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func mapCatalogSQLiteError(err error, op string) error {
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed: catalog_products.product_code") {
		return ErrDuplicateProductCode
	}
	if strings.Contains(msg, "UNIQUE constraint failed: catalog_products.slug") {
		return ErrDuplicateSlug
	}
	if strings.Contains(msg, "UNIQUE constraint failed: product_variants.sku") {
		return ErrDuplicateSKU
	}
	return fmt.Errorf("%s: %w", op, err)
}
