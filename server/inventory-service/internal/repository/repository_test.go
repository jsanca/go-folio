package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jsanca/go-folio/inventory-service/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

func newTestRepo(t *testing.T) repository.Repository {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE stock (
			sku       TEXT    PRIMARY KEY,
			available INTEGER NOT NULL DEFAULT 0,
			reserved  INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE reservations (
			id         TEXT PRIMARY KEY,
			sku        TEXT NOT NULL REFERENCES stock(sku),
			quantity   INTEGER NOT NULL,
			order_id   TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return repository.NewPostgresRepository(db)
}

// ── ListStock ─────────────────────────────────────────────────────────────────

func TestListStock_ReturnsAllRecordsOrderedBySKU(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "WAL-001", 20); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := repo.SeedSKU(ctx, "BAG-001", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := repo.SeedSKU(ctx, "BEL-001", 5); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stocks, err := repo.ListStock(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stocks) != 3 {
		t.Fatalf("expected 3 records, got %d", len(stocks))
	}

	// Results must be ordered by SKU ascending.
	wantOrder := []string{"BAG-001", "BEL-001", "WAL-001"}
	for i, want := range wantOrder {
		if stocks[i].SKU != want {
			t.Errorf("position %d: want SKU %q, got %q", i, want, stocks[i].SKU)
		}
	}

	// Spot-check fields are populated.
	bagIdx := 0
	if stocks[bagIdx].Available != 10 {
		t.Errorf("BAG-001 available: want 10, got %d", stocks[bagIdx].Available)
	}
	if stocks[bagIdx].Reserved != 0 {
		t.Errorf("BAG-001 reserved: want 0, got %d", stocks[bagIdx].Reserved)
	}
}

func TestListStock_EmptyTable_ReturnsNilSlice(t *testing.T) {
	repo := newTestRepo(t)

	stocks, err := repo.ListStock(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stocks) != 0 {
		t.Errorf("expected empty result, got %d records", len(stocks))
	}
}

// ── GetStock ──────────────────────────────────────────────────────────────────

func TestGetStock_HappyPath(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "WAL-001", 15); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s, err := repo.GetStock(ctx, "WAL-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.SKU != "WAL-001" {
		t.Errorf("SKU: want WAL-001, got %s", s.SKU)
	}
	if s.Available != 15 {
		t.Errorf("available: want 15, got %d", s.Available)
	}
	if s.Reserved != 0 {
		t.Errorf("reserved: want 0, got %d", s.Reserved)
	}
}

func TestGetStock_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetStock(context.Background(), "GHOST-SKU")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── AdjustStock ───────────────────────────────────────────────────────────────

func TestAdjustStock_PositiveDelta(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "BAG-001", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s, err := repo.AdjustStock(ctx, "BAG-001", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Available != 15 {
		t.Errorf("available: want 15, got %d", s.Available)
	}
}

func TestAdjustStock_NegativeDelta(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "BAG-002", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s, err := repo.AdjustStock(ctx, "BAG-002", -3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Available != 7 {
		t.Errorf("available: want 7, got %d", s.Available)
	}
}

func TestAdjustStock_InsufficientStock(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "BAG-003", 3); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := repo.AdjustStock(ctx, "BAG-003", -5)
	if !errors.Is(err, repository.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}

	// Row must be unchanged — the atomic UPDATE must not have mutated the row.
	s, err := repo.GetStock(ctx, "BAG-003")
	if err != nil {
		t.Fatalf("get stock after failed adjust: %v", err)
	}
	if s.Available != 3 {
		t.Errorf("available must be unchanged: want 3, got %d", s.Available)
	}
}

func TestAdjustStock_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.AdjustStock(context.Background(), "GHOST-SKU", 1)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── ReserveStock ──────────────────────────────────────────────────────────────

func TestReserveStock_HappyPath(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "BEL-001", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, err := repo.ReserveStock(ctx, "BEL-001", 3, "ORD-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID == "" {
		t.Error("expected non-empty reservation ID")
	}
	if res.SKU != "BEL-001" {
		t.Errorf("SKU: want BEL-001, got %s", res.SKU)
	}
	if res.Quantity != 3 {
		t.Errorf("quantity: want 3, got %d", res.Quantity)
	}
	if res.OrderID != "ORD-001" {
		t.Errorf("orderID: want ORD-001, got %s", res.OrderID)
	}

	// Stock levels must reflect the reservation.
	s, err := repo.GetStock(ctx, "BEL-001")
	if err != nil {
		t.Fatalf("get stock after reserve: %v", err)
	}
	if s.Available != 7 {
		t.Errorf("available after reserve: want 7, got %d", s.Available)
	}
	if s.Reserved != 3 {
		t.Errorf("reserved after reserve: want 3, got %d", s.Reserved)
	}
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "BEL-002", 2); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := repo.ReserveStock(ctx, "BEL-002", 5, "ORD-002")
	if !errors.Is(err, repository.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}

	// Stock must be unchanged — the atomic UPDATE must not have mutated the row.
	s, err := repo.GetStock(ctx, "BEL-002")
	if err != nil {
		t.Fatalf("get stock after failed reserve: %v", err)
	}
	if s.Available != 2 {
		t.Errorf("available must be unchanged: want 2, got %d", s.Available)
	}
	if s.Reserved != 0 {
		t.Errorf("reserved must be unchanged: want 0, got %d", s.Reserved)
	}
}

func TestReserveStock_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.ReserveStock(context.Background(), "GHOST-SKU", 1, "ORD-003")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── ReleaseStock ──────────────────────────────────────────────────────────────

func TestReleaseStock_HappyPath(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "WAL-002", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res, err := repo.ReserveStock(ctx, "WAL-002", 4, "ORD-004")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	s, err := repo.ReleaseStock(ctx, res.ID)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if s.Available != 10 {
		t.Errorf("available after release: want 10, got %d", s.Available)
	}
	if s.Reserved != 0 {
		t.Errorf("reserved after release: want 0, got %d", s.Reserved)
	}
}

func TestReleaseStock_ReservationNotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.ReleaseStock(context.Background(), "non-existent-reservation-id")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReleaseStock_SecondRelease_ReturnsErrNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	if err := repo.SeedSKU(ctx, "WAL-003", 10); err != nil {
		t.Fatalf("seed: %v", err)
	}
	reservation, err := repo.ReserveStock(ctx, "WAL-003", 3, "ORD-005")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	// First release succeeds.
	if _, err := repo.ReleaseStock(ctx, reservation.ID); err != nil {
		t.Fatalf("first release: %v", err)
	}

	// Second release of the same ID must return ErrNotFound.
	_, err = repo.ReleaseStock(ctx, reservation.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("second release: expected ErrNotFound, got %v", err)
	}

	// Stock must not be double-credited.
	s, err := repo.GetStock(ctx, "WAL-003")
	if err != nil {
		t.Fatalf("get stock after double release: %v", err)
	}
	if s.Available != 10 {
		t.Errorf("available after double release: want 10, got %d", s.Available)
	}
	if s.Reserved != 0 {
		t.Errorf("reserved after double release: want 0, got %d", s.Reserved)
	}
}
