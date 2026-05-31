package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jsanca/go-folio/internal/domain"
	"github.com/jsanca/go-folio/internal/repository"
)

// ── no-op SQL driver for transaction lifecycle in tests ───────────────────────

type noopDriver struct{}
type noopConn struct{}
type noopTx struct{}
type noopStmt struct{}
type noopResult struct{}

func (noopDriver) Open(_ string) (driver.Conn, error)         { return noopConn{}, nil }
func (noopConn) Prepare(_ string) (driver.Stmt, error)        { return noopStmt{}, nil }
func (noopConn) Close() error                                  { return nil }
func (noopConn) Begin() (driver.Tx, error)                    { return noopTx{}, nil }
func (noopTx) Commit() error                                   { return nil }
func (noopTx) Rollback() error                                 { return nil }
func (noopStmt) Close() error                                  { return nil }
func (noopStmt) NumInput() int                                 { return -1 }
func (noopStmt) Exec(_ []driver.Value) (driver.Result, error) { return noopResult{}, nil }
func (noopStmt) Query(_ []driver.Value) (driver.Rows, error)  { return nil, errors.New("noop") }
func (noopResult) LastInsertId() (int64, error)                { return 0, nil }
func (noopResult) RowsAffected() (int64, error)                { return 0, nil }

var registerNoopOnce sync.Once

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	registerNoopOnce.Do(func() {
		sql.Register("noop", noopDriver{})
	})
	db, err := sql.Open("noop", "")
	if err != nil {
		t.Fatalf("open noop db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ── fake repositories ─────────────────────────────────────────────────────────

type fakeCatalogProductRepo struct {
	products map[int64]*domain.Product
	nextID   int64
}

func newFakeCatalogProductRepo() *fakeCatalogProductRepo {
	return &fakeCatalogProductRepo{products: make(map[int64]*domain.Product), nextID: 1}
}

func (f *fakeCatalogProductRepo) CreateProduct(_ context.Context, p *domain.Product) (*domain.Product, error) {
	for _, existing := range f.products {
		if existing.ProductCode == p.ProductCode {
			return nil, repository.ErrDuplicateProductCode
		}
	}
	saved := *p
	saved.ID = f.nextID
	f.nextID++
	f.products[saved.ID] = &saved
	return &saved, nil
}

func (f *fakeCatalogProductRepo) GetProductByID(_ context.Context, id int64) (*domain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, repository.ErrProductNotFound
	}
	cp := *p
	return &cp, nil
}

// GetProductByIDForUpdate delegates to GetProductByID; no row locking in the fake.
func (f *fakeCatalogProductRepo) GetProductByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error) {
	return f.GetProductByID(ctx, id)
}

func (f *fakeCatalogProductRepo) ListProducts(_ context.Context) ([]domain.Product, error) {
	var list []domain.Product
	for _, p := range f.products {
		list = append(list, *p)
	}
	return list, nil
}

func (f *fakeCatalogProductRepo) UpdateProduct(_ context.Context, id int64, p *domain.Product) (*domain.Product, error) {
	if _, ok := f.products[id]; !ok {
		return nil, repository.ErrProductNotFound
	}
	for existingID, existing := range f.products {
		if existingID != id && existing.ProductCode == p.ProductCode {
			return nil, repository.ErrDuplicateProductCode
		}
		if existingID != id && existing.Slug == p.Slug {
			return nil, repository.ErrDuplicateSlug
		}
	}
	saved := *p
	saved.ID = id
	f.products[id] = &saved
	return &saved, nil
}

func (f *fakeCatalogProductRepo) DeleteProduct(_ context.Context, id int64) error {
	if _, ok := f.products[id]; !ok {
		return repository.ErrProductNotFound
	}
	delete(f.products, id)
	return nil
}

// WithTx returns the same fake; transactions are no-ops in tests.
func (f *fakeCatalogProductRepo) WithTx(_ *sql.Tx) repository.CatalogProductRepository {
	return f
}

type fakeVariantRepo struct {
	variants map[string]*domain.ProductVariant
	byID     map[int64]*domain.ProductVariant
	nextID   int64
}

func newFakeVariantRepo() *fakeVariantRepo {
	return &fakeVariantRepo{
		variants: make(map[string]*domain.ProductVariant),
		byID:     make(map[int64]*domain.ProductVariant),
		nextID:   1,
	}
}

func (f *fakeVariantRepo) AddVariant(_ context.Context, v *domain.ProductVariant) (*domain.ProductVariant, error) {
	if _, exists := f.variants[v.SKU]; exists {
		return nil, repository.ErrDuplicateSKU
	}
	saved := *v
	saved.ID = f.nextID
	f.nextID++
	f.variants[saved.SKU] = &saved
	f.byID[saved.ID] = &saved
	return &saved, nil
}

func (f *fakeVariantRepo) GetVariantBySKU(_ context.Context, sku string) (*domain.ProductVariant, error) {
	v, ok := f.variants[sku]
	if !ok {
		return nil, repository.ErrVariantNotFound
	}
	cp := *v
	return &cp, nil
}

func (f *fakeVariantRepo) GetVariantByID(_ context.Context, id int64) (*domain.ProductVariant, error) {
	v, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrVariantNotFound
	}
	cp := *v
	return &cp, nil
}

func (f *fakeVariantRepo) UpdateVariantPricing(_ context.Context, sku string, retail domain.Money, sale *domain.Money, currency string) error {
	v, ok := f.variants[sku]
	if !ok {
		return repository.ErrVariantNotFound
	}
	v.RetailPrice = retail
	v.SalePrice = sale
	v.Currency = currency
	f.byID[v.ID] = v
	return nil
}

func (f *fakeVariantRepo) ListVariantsByProductID(_ context.Context, productID int64) ([]domain.ProductVariant, error) {
	var list []domain.ProductVariant
	for _, v := range f.variants {
		if v.ProductID == productID {
			list = append(list, *v)
		}
	}
	return list, nil
}

type fakeImageRepo struct {
	images []domain.ProductImage
	nextID int64
}

func newFakeImageRepo() *fakeImageRepo {
	return &fakeImageRepo{nextID: 1}
}

func (f *fakeImageRepo) AddImage(_ context.Context, img *domain.ProductImage) (*domain.ProductImage, error) {
	saved := *img
	saved.ID = f.nextID
	f.nextID++
	f.images = append(f.images, saved)
	return &saved, nil
}

func (f *fakeImageRepo) ListImagesByProductID(_ context.Context, productID int64) ([]domain.ProductImage, error) {
	var list []domain.ProductImage
	for _, img := range f.images {
		if img.ProductID == productID {
			list = append(list, img)
		}
	}
	return list, nil
}

func (f *fakeImageRepo) ListImagesByVariantID(_ context.Context, variantID int64) ([]domain.ProductImage, error) {
	var list []domain.ProductImage
	for _, img := range f.images {
		if img.VariantID != nil && *img.VariantID == variantID {
			list = append(list, img)
		}
	}
	return list, nil
}

// ── fake sync repository ──────────────────────────────────────────────────────

type fakeSyncRepo struct {
	projections   []domain.ProductProjection
	inventory     []domain.VariantInventoryRecord
	capturedQuery domain.SyncQuery
}

func (f *fakeSyncRepo) ListProductProjectionPage(_ context.Context, q domain.SyncQuery) ([]domain.ProductProjection, bool, error) {
	f.capturedQuery = q
	if len(f.projections) > q.PageSize {
		return f.projections[:q.PageSize], true, nil
	}
	return f.projections, false, nil
}

func (f *fakeSyncRepo) ListVariantInventoryPage(_ context.Context, q domain.SyncQuery) ([]domain.VariantInventoryRecord, bool, error) {
	f.capturedQuery = q
	if len(f.inventory) > q.PageSize {
		return f.inventory[:q.PageSize], true, nil
	}
	return f.inventory, false, nil
}

// ── test setup helpers ────────────────────────────────────────────────────────

func newTestCatalogService(t *testing.T) (CatalogService, *fakeCatalogProductRepo, *fakeVariantRepo, *fakeImageRepo) {
	t.Helper()
	pr := newFakeCatalogProductRepo()
	vr := newFakeVariantRepo()
	ir := newFakeImageRepo()
	sr := &fakeSyncRepo{}
	return NewCatalogService(newTestDB(t), pr, vr, ir, sr), pr, vr, ir
}

func newSyncService(t *testing.T, sr *fakeSyncRepo) CatalogService {
	t.Helper()
	return NewCatalogService(newTestDB(t), newFakeCatalogProductRepo(), newFakeVariantRepo(), newFakeImageRepo(), sr)
}

// ── Product tests ─────────────────────────────────────────────────────────────

func TestCatalog_CreateProduct_ValidProduct(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	p := &domain.Product{ProductCode: "BM-02", Title: "Billetera Colores", Slug: "billetera-colores"}
	got, err := svc.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
}

func TestCatalog_CreateProduct_RejectsInvalidProduct(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	_, err := svc.CreateProduct(context.Background(), &domain.Product{})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCatalog_CreateProduct_ValidatesWithoutSKUOrPriceOrStock(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	p := &domain.Product{ProductCode: "X-01", Title: "Test", Slug: "test"}
	_, err := svc.CreateProduct(context.Background(), p)
	if err != nil {
		t.Errorf("product without SKU/price/stock should be valid: %v", err)
	}
}

func TestCatalog_GetProductByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	_, err := svc.GetProductByID(context.Background(), 999)
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

// ── Variant tests ─────────────────────────────────────────────────────────────

func TestCatalog_AddVariant_RequiresSKU(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	v := &domain.ProductVariant{ProductID: 1, Currency: "CRC", StockStatus: domain.StockStatusInStock}
	_, err := svc.AddVariantToProduct(context.Background(), v)
	if !errors.Is(err, ErrInvalidVariant) {
		t.Errorf("expected ErrInvalidVariant, got %v", err)
	}
}

func TestCatalog_AddVariant_Valid(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	v := &domain.ProductVariant{
		ProductID:   1,
		SKU:         "BM-02-COL-CO-CA",
		Currency:    "CRC",
		RetailPrice: domain.Money{AmountCents: 1500000},
		StockStatus: domain.StockStatusInStock,
	}
	got, err := svc.AddVariantToProduct(context.Background(), v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestCatalog_UpdateVariantPricing_BySKU(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService(t)
	_ = seedVariant(vr, "BM-02-COL-CO-MI", 1)

	retail := domain.Money{AmountCents: 2000000}
	sale := domain.Money{AmountCents: 1800000}
	err := svc.UpdateVariantPricing(context.Background(), "BM-02-COL-CO-MI", retail, &sale, "CRC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, _ := vr.GetVariantBySKU(context.Background(), "BM-02-COL-CO-MI")
	if v.RetailPrice.AmountCents != 2000000 {
		t.Errorf("expected retail 2000000, got %d", v.RetailPrice.AmountCents)
	}
	if v.SalePrice == nil || v.SalePrice.AmountCents != 1800000 {
		t.Error("expected sale price 1800000")
	}
}

func TestCatalog_UpdateVariantPricing_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	err := svc.UpdateVariantPricing(context.Background(), "GHOST", domain.Money{}, nil, "CRC")
	if !errors.Is(err, repository.ErrVariantNotFound) {
		t.Errorf("expected ErrVariantNotFound, got %v", err)
	}
}

func TestCatalog_ListVariantsByProductID(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService(t)
	_ = seedVariant(vr, "BM-02-COL-CO-CA", 10)
	_ = seedVariant(vr, "BM-02-COL-CO-NE", 10)
	_ = seedVariant(vr, "OTHER-SKU", 99)

	list, err := svc.ListVariantsByProductID(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 variants for product 10, got %d", len(list))
	}
}

// ── Image tests ───────────────────────────────────────────────────────────────

func TestCatalog_AddProductImage_ProductLevel(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	img := &domain.ProductImage{ProductID: 1, URL: "https://example.com/img.jpg"}
	got, err := svc.AddProductImage(context.Background(), img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if got.VariantID != nil {
		t.Error("expected nil variant ID for product-level image")
	}
}

func TestCatalog_AddProductImage_VariantLevel(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	vid := int64(7)
	img := &domain.ProductImage{ProductID: 1, VariantID: &vid, URL: "https://example.com/v.jpg"}
	got, err := svc.AddProductImage(context.Background(), img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.VariantID == nil || *got.VariantID != 7 {
		t.Errorf("expected variant ID 7, got %v", got.VariantID)
	}
}

func TestCatalog_AddProductImage_RejectsInvalid(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	_, err := svc.AddProductImage(context.Background(), &domain.ProductImage{})
	if !errors.Is(err, ErrInvalidImage) {
		t.Errorf("expected ErrInvalidImage, got %v", err)
	}
}

func TestCatalog_ListProductImagesByProductID(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 1, URL: "https://a.com/1.jpg"}) //nolint:errcheck
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 1, URL: "https://a.com/2.jpg"}) //nolint:errcheck
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 2, URL: "https://a.com/3.jpg"}) //nolint:errcheck

	list, err := svc.ListProductImagesByProductID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 images for product 1, got %d", len(list))
	}
}

func TestCatalog_ListProductImagesByVariantID(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	vid1 := int64(3)
	vid2 := int64(4)
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 1, VariantID: &vid1, URL: "https://a.com/v3.jpg"}) //nolint:errcheck
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 1, VariantID: &vid2, URL: "https://a.com/v4.jpg"}) //nolint:errcheck
	svc.AddProductImage(context.Background(), &domain.ProductImage{ProductID: 1, URL: "https://a.com/base.jpg"})                 //nolint:errcheck

	list, err := svc.ListProductImagesByVariantID(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 image for variant 3, got %d", len(list))
	}
}

// ── Sync tests ────────────────────────────────────────────────────────────────

func TestSync_DefaultPageSizeForProjections(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	_, err := svc.ListProductProjections(context.Background(), domain.SyncQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sr.capturedQuery.PageSize != defaultProjectionPageSize {
		t.Errorf("expected default pageSize %d, got %d", defaultProjectionPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_DefaultPageSizeForInventory(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	_, err := svc.ListVariantInventory(context.Background(), domain.SyncQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sr.capturedQuery.PageSize != defaultInventoryPageSize {
		t.Errorf("expected default pageSize %d, got %d", defaultInventoryPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_MaxPageSizeClamped_Projections(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	_, _ = svc.ListProductProjections(context.Background(), domain.SyncQuery{PageSize: 99999})
	if sr.capturedQuery.PageSize != maxProjectionPageSize {
		t.Errorf("expected pageSize clamped to %d, got %d", maxProjectionPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_MaxPageSizeClamped_Inventory(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	_, _ = svc.ListVariantInventory(context.Background(), domain.SyncQuery{PageSize: 99999})
	if sr.capturedQuery.PageSize != maxInventoryPageSize {
		t.Errorf("expected pageSize clamped to %d, got %d", maxInventoryPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_UpdatedSincePassedToRepo(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	ts := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, _ = svc.ListVariantInventory(context.Background(), domain.SyncQuery{UpdatedSince: &ts})
	if sr.capturedQuery.UpdatedSince == nil || !sr.capturedQuery.UpdatedSince.Equal(ts) {
		t.Errorf("expected updatedSince to be passed to repo, got %v", sr.capturedQuery.UpdatedSince)
	}
}

func TestSync_HasNextTrueWhenMoreRecordsExist(t *testing.T) {
	// Fake repo has 3 records; we request pageSize=2 → hasNext should be true.
	sr := &fakeSyncRepo{
		inventory: []domain.VariantInventoryRecord{
			{SKU: "SKU-1", VariantID: 1, UpdatedAt: time.Now()},
			{SKU: "SKU-2", VariantID: 2, UpdatedAt: time.Now()},
			{SKU: "SKU-3", VariantID: 3, UpdatedAt: time.Now()},
		},
	}
	svc := newSyncService(t, sr)
	page, err := svc.ListVariantInventory(context.Background(), domain.SyncQuery{PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page.HasNext {
		t.Error("expected hasNext=true")
	}
	if len(page.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(page.Items))
	}
}

func TestSync_NextCursorSetWhenHasNext(t *testing.T) {
	now := time.Now().UTC()
	sr := &fakeSyncRepo{
		inventory: []domain.VariantInventoryRecord{
			{SKU: "SKU-1", VariantID: 1, UpdatedAt: now},
			{SKU: "SKU-2", VariantID: 2, UpdatedAt: now},
			{SKU: "SKU-3", VariantID: 3, UpdatedAt: now},
		},
	}
	svc := newSyncService(t, sr)
	page, _ := svc.ListVariantInventory(context.Background(), domain.SyncQuery{PageSize: 2})
	if page.NextCursor == "" {
		t.Error("expected non-empty nextCursor when hasNext=true")
	}
	// Verify nextCursor decodes to the last returned item.
	c, err := decodeCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("nextCursor should be a valid cursor: %v", err)
	}
	if c.ID != 2 {
		t.Errorf("expected cursor ID 2 (last returned item), got %d", c.ID)
	}
}

func TestSync_EmptyResultsReturnNonNilItems(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)

	projPage, _ := svc.ListProductProjections(context.Background(), domain.SyncQuery{})
	if projPage.Items == nil {
		t.Error("ProductProjectionPage.Items must not be nil")
	}

	invPage, _ := svc.ListVariantInventory(context.Background(), domain.SyncQuery{})
	if invPage.Items == nil {
		t.Error("VariantInventoryPage.Items must not be nil")
	}
}

func TestSync_ProjectionsIncludeVariantsAndImages(t *testing.T) {
	sr := &fakeSyncRepo{
		projections: []domain.ProductProjection{
			{
				Product:  domain.Product{ID: 1, ProductCode: "BM-02", UpdatedAt: time.Now()},
				Variants: []domain.ProductVariant{{SKU: "BM-02-COL-CO-CA"}},
				Images:   []domain.ProductImage{{URL: "https://example.com/img.jpg"}},
			},
		},
	}
	svc := newSyncService(t, sr)
	page, err := svc.ListProductProjections(context.Background(), domain.SyncQuery{PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 projection, got %d", len(page.Items))
	}
	if len(page.Items[0].Variants) != 1 {
		t.Errorf("expected 1 variant in projection, got %d", len(page.Items[0].Variants))
	}
	if len(page.Items[0].Images) != 1 {
		t.Errorf("expected 1 image in projection, got %d", len(page.Items[0].Images))
	}
}

func TestSync_ProjectionsAvoidNilSlices(t *testing.T) {
	// Fake repo returns a projection with nil variants and images.
	sr := &fakeSyncRepo{
		projections: []domain.ProductProjection{
			{Product: domain.Product{ID: 1, ProductCode: "X", UpdatedAt: time.Now()}},
		},
	}
	svc := newSyncService(t, sr)
	page, _ := svc.ListProductProjections(context.Background(), domain.SyncQuery{PageSize: 10})
	if page.Items[0].Variants == nil {
		t.Error("Variants must not be nil — would serialize as JSON null")
	}
	if page.Items[0].Images == nil {
		t.Error("Images must not be nil — would serialize as JSON null")
	}
}

func TestSync_InvalidCursor_ReturnsErrInvalidCursor(t *testing.T) {
	svc := newSyncService(t, &fakeSyncRepo{})
	_, err := svc.ListProductProjections(context.Background(), domain.SyncQuery{Cursor: "!!!not-a-cursor!!!"})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestSync_CursorDecodedAndPassedToRepo(t *testing.T) {
	ts := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	validCursor := encodeCursor(domain.SyncCursor{UpdatedAt: ts, ID: 42})

	sr := &fakeSyncRepo{}
	svc := newSyncService(t, sr)
	_, _ = svc.ListVariantInventory(context.Background(), domain.SyncQuery{Cursor: validCursor})
	if sr.capturedQuery.AfterAt == nil || !sr.capturedQuery.AfterAt.Equal(ts) {
		t.Errorf("expected AfterAt=%v, got %v", ts, sr.capturedQuery.AfterAt)
	}
	if sr.capturedQuery.AfterID != 42 {
		t.Errorf("expected AfterID=42, got %d", sr.capturedQuery.AfterID)
	}
}

func TestSync_VariantInventoryReturnsLightweightRecord(t *testing.T) {
	sr := &fakeSyncRepo{
		inventory: []domain.VariantInventoryRecord{
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
	}
	svc := newSyncService(t, sr)
	page, err := svc.ListVariantInventory(context.Background(), domain.SyncQuery{PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	rec := page.Items[0]
	if rec.ProductCode != "BM-02" {
		t.Errorf("expected productCode BM-02, got %s", rec.ProductCode)
	}
	if rec.SKU != "BM-02-COL-CO-NE" {
		t.Errorf("expected SKU BM-02-COL-CO-NE, got %s", rec.SKU)
	}
	if rec.RetailPrice.AmountCents != 2439000 {
		t.Errorf("expected 2439000 amountCents, got %d", rec.RetailPrice.AmountCents)
	}
	if rec.Currency != "CRC" {
		t.Errorf("expected currency CRC, got %s", rec.Currency)
	}
}

// ── UpdateProduct tests ───────────────────────────────────────────────────────

func TestCatalog_UpdateProduct_HappyPath(t *testing.T) {
	svc, pr, _, _ := newTestCatalogService(t)
	created, _ := svc.CreateProduct(context.Background(), &domain.Product{
		ProductCode: "BM-02", Title: "Original Title", Slug: "original-slug", Department: "Hombre",
	})

	newTitle := "Updated Title"
	newDept := "Mujer"
	got, err := svc.UpdateProduct(context.Background(), created.ID, ProductUpdate{
		Title:      &newTitle,
		Department: &newDept,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, got.Title)
	}
	if got.Department != newDept {
		t.Errorf("expected department %q, got %q", newDept, got.Department)
	}
	_ = pr
}

func TestCatalog_UpdateProduct_PartialUpdate(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	created, _ := svc.CreateProduct(context.Background(), &domain.Product{
		ProductCode: "BM-03", Title: "Stable Title", Slug: "stable-slug", Department: "Hombre",
	})

	newTitle := "New Title Only"
	got, err := svc.UpdateProduct(context.Background(), created.ID, ProductUpdate{Title: &newTitle})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, got.Title)
	}
	if got.ProductCode != "BM-03" {
		t.Errorf("expected unchanged product code BM-03, got %q", got.ProductCode)
	}
	if got.Slug != "stable-slug" {
		t.Errorf("expected unchanged slug stable-slug, got %q", got.Slug)
	}
}

func TestCatalog_UpdateProduct_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	_, err := svc.UpdateProduct(context.Background(), 999, ProductUpdate{})
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestCatalog_UpdateProduct_DuplicateCode(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	svc.CreateProduct(context.Background(), &domain.Product{ProductCode: "FIRST", Title: "First", Slug: "first"})           //nolint:errcheck
	second, _ := svc.CreateProduct(context.Background(), &domain.Product{ProductCode: "SECOND", Title: "Second", Slug: "second"}) //nolint:errcheck

	code := "FIRST"
	_, err := svc.UpdateProduct(context.Background(), second.ID, ProductUpdate{ProductCode: &code})
	if !errors.Is(err, repository.ErrDuplicateProductCode) {
		t.Errorf("expected ErrDuplicateProductCode, got %v", err)
	}
}

func TestCatalog_UpdateProduct_ClearRequiredField(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	created, _ := svc.CreateProduct(context.Background(), &domain.Product{
		ProductCode: "BM-05", Title: "Title", Slug: "slug",
	})

	empty := ""
	_, err := svc.UpdateProduct(context.Background(), created.ID, ProductUpdate{ProductCode: &empty})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCatalog_UpdateProduct_RollsBackOnError(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	created, _ := svc.CreateProduct(context.Background(), &domain.Product{
		ProductCode: "RB-01", Title: "Rollback Test", Slug: "rollback-test",
	})

	// Attempt to clear the product code — validation fails before any write.
	empty := ""
	_, err := svc.UpdateProduct(context.Background(), created.ID, ProductUpdate{ProductCode: &empty})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("expected ErrInvalidProduct, got %v", err)
	}

	// Product must be unchanged after the failed transaction.
	got, err := svc.GetProductByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching product: %v", err)
	}
	if got.ProductCode != "RB-01" {
		t.Errorf("expected product code RB-01 after rollback, got %q", got.ProductCode)
	}
}

func TestCatalog_DeleteProduct_HappyPath(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	created, _ := svc.CreateProduct(context.Background(), &domain.Product{
		ProductCode: "DEL-01", Title: "To Delete", Slug: "to-delete",
	})

	if err := svc.DeleteProduct(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetProductByID(context.Background(), created.ID)
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound after delete, got %v", err)
	}
}

func TestCatalog_DeleteProduct_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	err := svc.DeleteProduct(context.Background(), 999)
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

// ── ListProducts tests ────────────────────────────────────────────────────────

func TestCatalog_ListProducts_HappyPath(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	svc.CreateProduct(context.Background(), &domain.Product{ProductCode: "P-01", Title: "Product One", Slug: "product-one"})   //nolint:errcheck
	svc.CreateProduct(context.Background(), &domain.Product{ProductCode: "P-02", Title: "Product Two", Slug: "product-two"})   //nolint:errcheck

	list, err := svc.ListProducts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 products, got %d", len(list))
	}
}

func TestCatalog_ListProducts_Empty(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	list, err := svc.ListProducts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

// ── GetVariantBySKU tests ─────────────────────────────────────────────────────

func TestCatalog_GetVariantBySKU_HappyPath(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService(t)
	_ = seedVariant(vr, "BM-02-COL-CO-CA", 1)

	v, err := svc.GetVariantBySKU(context.Background(), "BM-02-COL-CO-CA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.SKU != "BM-02-COL-CO-CA" {
		t.Errorf("expected SKU BM-02-COL-CO-CA, got %s", v.SKU)
	}
	if v.ProductID != 1 {
		t.Errorf("expected productID 1, got %d", v.ProductID)
	}
}

func TestCatalog_GetVariantBySKU_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService(t)
	_, err := svc.GetVariantBySKU(context.Background(), "GHOST-SKU")
	if !errors.Is(err, repository.ErrVariantNotFound) {
		t.Errorf("expected ErrVariantNotFound, got %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func seedVariant(vr *fakeVariantRepo, sku string, productID int64) *domain.ProductVariant {
	v := &domain.ProductVariant{
		ProductID:   productID,
		SKU:         sku,
		Currency:    "CRC",
		RetailPrice: domain.Money{AmountCents: 1000000},
		StockStatus: domain.StockStatusInStock,
	}
	saved, _ := vr.AddVariant(context.Background(), v)
	return saved
}
