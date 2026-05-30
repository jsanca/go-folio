package repository_test

import (
	"context"
	"database/sql"
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
	return repository.NewSQLiteRepository(db)
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
