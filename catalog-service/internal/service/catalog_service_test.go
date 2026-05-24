package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leatherstore/catalog-service/internal/domain"
	"github.com/leatherstore/catalog-service/internal/repository"
)

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

func (f *fakeCatalogProductRepo) ListProducts(_ context.Context) ([]domain.Product, error) {
	var list []domain.Product
	for _, p := range f.products {
		list = append(list, *p)
	}
	return list, nil
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

func (f *fakeVariantRepo) UpdateVariantInventory(_ context.Context, sku string, qty int, status domain.StockStatus) error {
	v, ok := f.variants[sku]
	if !ok {
		return repository.ErrVariantNotFound
	}
	v.StockQuantity = qty
	v.StockStatus = status
	f.byID[v.ID] = v
	return nil
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

func newTestCatalogService() (CatalogService, *fakeCatalogProductRepo, *fakeVariantRepo, *fakeImageRepo) {
	pr := newFakeCatalogProductRepo()
	vr := newFakeVariantRepo()
	ir := newFakeImageRepo()
	sr := &fakeSyncRepo{}
	return NewCatalogService(pr, vr, ir, sr), pr, vr, ir
}

func newSyncService(sr *fakeSyncRepo) CatalogService {
	return NewCatalogService(newFakeCatalogProductRepo(), newFakeVariantRepo(), newFakeImageRepo(), sr)
}

// ── Product tests ─────────────────────────────────────────────────────────────

func TestCatalog_CreateProduct_ValidProduct(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
	_, err := svc.CreateProduct(context.Background(), &domain.Product{})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Errorf("expected ErrInvalidProduct, got %v", err)
	}
}

func TestCatalog_CreateProduct_ValidatesWithoutSKUOrPriceOrStock(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
	p := &domain.Product{ProductCode: "X-01", Title: "Test", Slug: "test"}
	_, err := svc.CreateProduct(context.Background(), p)
	if err != nil {
		t.Errorf("product without SKU/price/stock should be valid: %v", err)
	}
}

func TestCatalog_GetProductByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
	_, err := svc.GetProductByID(context.Background(), 999)
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

// ── Variant tests ─────────────────────────────────────────────────────────────

func TestCatalog_AddVariant_RequiresSKU(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
	v := &domain.ProductVariant{ProductID: 1, Currency: "CRC", StockStatus: domain.StockStatusInStock}
	_, err := svc.AddVariantToProduct(context.Background(), v)
	if !errors.Is(err, ErrInvalidVariant) {
		t.Errorf("expected ErrInvalidVariant, got %v", err)
	}
}

func TestCatalog_AddVariant_Valid(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
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

func TestCatalog_UpdateVariantInventory_BySKU(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService()
	_ = seedVariant(vr, "BM-02-COL-CO-NE", 1)

	err := svc.UpdateVariantInventory(context.Background(), "BM-02-COL-CO-NE", 5, domain.StockStatusLowStock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, _ := vr.GetVariantBySKU(context.Background(), "BM-02-COL-CO-NE")
	if v.StockQuantity != 5 {
		t.Errorf("expected qty 5, got %d", v.StockQuantity)
	}
	if v.StockStatus != domain.StockStatusLowStock {
		t.Errorf("expected LOW_STOCK, got %s", v.StockStatus)
	}
}

func TestCatalog_UpdateVariantInventory_NotFound(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
	err := svc.UpdateVariantInventory(context.Background(), "GHOST-SKU", 0, domain.StockStatusOutOfStock)
	if !errors.Is(err, repository.ErrVariantNotFound) {
		t.Errorf("expected ErrVariantNotFound, got %v", err)
	}
}

func TestCatalog_UpdateVariantInventory_RejectsNegativeQty(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
	err := svc.UpdateVariantInventory(context.Background(), "ANY", -1, domain.StockStatusInStock)
	if !errors.Is(err, ErrInvalidVariant) {
		t.Errorf("expected ErrInvalidVariant, got %v", err)
	}
}

func TestCatalog_UpdateVariantPricing_BySKU(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
	err := svc.UpdateVariantPricing(context.Background(), "GHOST", domain.Money{}, nil, "CRC")
	if !errors.Is(err, repository.ErrVariantNotFound) {
		t.Errorf("expected ErrVariantNotFound, got %v", err)
	}
}

func TestCatalog_ListVariantsByProductID(t *testing.T) {
	svc, _, vr, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
	_, err := svc.AddProductImage(context.Background(), &domain.ProductImage{})
	if !errors.Is(err, ErrInvalidImage) {
		t.Errorf("expected ErrInvalidImage, got %v", err)
	}
}

func TestCatalog_ListProductImagesByProductID(t *testing.T) {
	svc, _, _, _ := newTestCatalogService()
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
	svc, _, _, _ := newTestCatalogService()
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
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
	_, _ = svc.ListProductProjections(context.Background(), domain.SyncQuery{PageSize: 99999})
	if sr.capturedQuery.PageSize != maxProjectionPageSize {
		t.Errorf("expected pageSize clamped to %d, got %d", maxProjectionPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_MaxPageSizeClamped_Inventory(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(sr)
	_, _ = svc.ListVariantInventory(context.Background(), domain.SyncQuery{PageSize: 99999})
	if sr.capturedQuery.PageSize != maxInventoryPageSize {
		t.Errorf("expected pageSize clamped to %d, got %d", maxInventoryPageSize, sr.capturedQuery.PageSize)
	}
}

func TestSync_UpdatedSincePassedToRepo(t *testing.T) {
	sr := &fakeSyncRepo{}
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)

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
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
	page, _ := svc.ListProductProjections(context.Background(), domain.SyncQuery{PageSize: 10})
	if page.Items[0].Variants == nil {
		t.Error("Variants must not be nil — would serialize as JSON null")
	}
	if page.Items[0].Images == nil {
		t.Error("Images must not be nil — would serialize as JSON null")
	}
}

func TestSync_InvalidCursor_ReturnsErrInvalidCursor(t *testing.T) {
	svc := newSyncService(&fakeSyncRepo{})
	_, err := svc.ListProductProjections(context.Background(), domain.SyncQuery{Cursor: "!!!not-a-cursor!!!"})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestSync_CursorDecodedAndPassedToRepo(t *testing.T) {
	ts := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	validCursor := encodeCursor(domain.SyncCursor{UpdatedAt: ts, ID: 42})

	sr := &fakeSyncRepo{}
	svc := newSyncService(sr)
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
	svc := newSyncService(sr)
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
