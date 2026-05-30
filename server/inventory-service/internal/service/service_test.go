package service

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/inventory-service/internal/domain"
	"github.com/jsanca/go-folio/inventory-service/internal/repository"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── fake repository ───────────────────────────────────────────────────────────

// fakeRepository is an in-memory Repository for unit tests.
// WithTx returns itself — the transaction lifecycle is exercised through the
// real *sql.DB passed to NewService; data mutations stay in-memory.
type fakeRepository struct {
	stock        map[string]*domain.Stock
	reservations map[string]*domain.Reservation
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		stock:        make(map[string]*domain.Stock),
		reservations: make(map[string]*domain.Reservation),
	}
}

func (r *fakeRepository) seed(sku string, available, reserved int32) {
	r.stock[sku] = &domain.Stock{SKU: sku, Available: available, Reserved: reserved}
}

func (r *fakeRepository) GetStock(_ context.Context, sku string) (*domain.Stock, error) {
	s, ok := r.stock[sku]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *fakeRepository) AdjustStock(_ context.Context, sku string, delta int32) (*domain.Stock, error) {
	s, ok := r.stock[sku]
	if !ok {
		return nil, repository.ErrNotFound
	}
	newAvailable := s.Available + delta
	if newAvailable < 0 {
		return nil, repository.ErrInsufficientStock
	}
	s.Available = newAvailable
	cp := *s
	return &cp, nil
}

func (r *fakeRepository) ReserveStock(_ context.Context, sku string, quantity int32, orderID string) (*domain.Reservation, error) {
	s, ok := r.stock[sku]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if s.Available < quantity {
		return nil, repository.ErrInsufficientStock
	}
	id := repository.NewID()
	s.Available -= quantity
	s.Reserved += quantity
	res := &domain.Reservation{ID: id, SKU: sku, Quantity: quantity, OrderID: orderID}
	r.reservations[id] = res
	cp := *res
	return &cp, nil
}

func (r *fakeRepository) ReleaseStock(_ context.Context, reservationID string) (*domain.Stock, error) {
	res, ok := r.reservations[reservationID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	s := r.stock[res.SKU]
	s.Available += res.Quantity
	s.Reserved -= res.Quantity
	delete(r.reservations, reservationID)
	cp := *s
	return &cp, nil
}

func (r *fakeRepository) SeedSKU(_ context.Context, sku string, available int32) error {
	if _, ok := r.stock[sku]; !ok {
		r.stock[sku] = &domain.Stock{SKU: sku, Available: available}
	}
	return nil
}

func (r *fakeRepository) HasAnyStock(_ context.Context) (bool, error) {
	return len(r.stock) > 0, nil
}

func (r *fakeRepository) ListStock(_ context.Context) ([]domain.Stock, error) {
	result := make([]domain.Stock, 0, len(r.stock))
	for _, s := range r.stock {
		result = append(result, *s)
	}
	return result, nil
}

// WithTx returns r itself so unit tests remain in-memory while the service's
// BeginTx/Commit/Rollback lifecycle runs on the real SQLite connection.
func (r *fakeRepository) WithTx(_ *sql.Tx) repository.Repository {
	return r
}

// ── test helpers ──────────────────────────────────────────────────────────────

func newTestService(t *testing.T, repo *fakeRepository) *Service {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(db, repo, logger)
}

// ── GetStock ──────────────────────────────────────────────────────────────────

func TestGetStock_ReturnsStockForKnownSKU(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-WALLET-001", 10, 2)
	svc := newTestService(t, repo)

	resp, err := svc.GetStock(context.Background(), &invpb.GetStockRequest{Sku: "LTH-WALLET-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Available != 10 {
		t.Errorf("expected available=10, got %d", resp.Available)
	}
	if resp.Reserved != 2 {
		t.Errorf("expected reserved=2, got %d", resp.Reserved)
	}
	if resp.Sku != "LTH-WALLET-001" {
		t.Errorf("expected sku=LTH-WALLET-001, got %s", resp.Sku)
	}
}

func TestGetStock_ErrNotFound_ForUnknownSKU(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	_, err := svc.GetStock(context.Background(), &invpb.GetStockRequest{Sku: "GHOST-SKU"})
	if err == nil {
		t.Fatal("expected error for unknown SKU, got nil")
	}
}

// ── AdjustStock ───────────────────────────────────────────────────────────────

func TestAdjustStock_PositiveDelta_IncreasesAvailable(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BAG-002", 5, 0)
	svc := newTestService(t, repo)

	resp, err := svc.AdjustStock(context.Background(), &invpb.AdjustStockRequest{Sku: "LTH-BAG-002", Delta: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Available != 15 {
		t.Errorf("expected available=15, got %d", resp.Available)
	}
}

func TestAdjustStock_NegativeDelta_DecreasesAvailable(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BAG-002", 10, 0)
	svc := newTestService(t, repo)

	resp, err := svc.AdjustStock(context.Background(), &invpb.AdjustStockRequest{Sku: "LTH-BAG-002", Delta: -3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Available != 7 {
		t.Errorf("expected available=7, got %d", resp.Available)
	}
}

func TestAdjustStock_ErrNotFound_ForUnknownSKU(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	_, err := svc.AdjustStock(context.Background(), &invpb.AdjustStockRequest{Sku: "GHOST-SKU", Delta: 1})
	if err == nil {
		t.Fatal("expected error for unknown SKU, got nil")
	}
	if !isNotFoundStatus(err) {
		t.Errorf("expected NotFound gRPC status, got: %v", err)
	}
}

func TestAdjustStock_ErrInsufficientStock_WhenDeltaExceedsAvailable(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BAG-002", 3, 0)
	svc := newTestService(t, repo)

	_, err := svc.AdjustStock(context.Background(), &invpb.AdjustStockRequest{Sku: "LTH-BAG-002", Delta: -10})
	if err == nil {
		t.Fatal("expected error for insufficient stock, got nil")
	}
	if !isFailedPreconditionStatus(err) {
		t.Errorf("expected FailedPrecondition gRPC status, got: %v", err)
	}
}

// ── ReserveStock ──────────────────────────────────────────────────────────────

func TestReserveStock_DeductsAvailableAndCreatesReservation(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BELT-003", 8, 0)
	svc := newTestService(t, repo)

	resp, err := svc.ReserveStock(context.Background(), &invpb.ReserveStockRequest{
		Sku:      "LTH-BELT-003",
		Quantity: 3,
		OrderId:  "ORD-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.ReservationId == "" {
		t.Error("expected a non-empty reservation ID")
	}

	stock, _ := repo.GetStock(context.Background(), "LTH-BELT-003")
	if stock.Available != 5 {
		t.Errorf("expected available=5 after reserve, got %d", stock.Available)
	}
	if stock.Reserved != 3 {
		t.Errorf("expected reserved=3 after reserve, got %d", stock.Reserved)
	}
}

func TestReserveStock_ErrNotFound_ForUnknownSKU(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	_, err := svc.ReserveStock(context.Background(), &invpb.ReserveStockRequest{
		Sku: "GHOST-SKU", Quantity: 1, OrderId: "ORD-X",
	})
	if err == nil {
		t.Fatal("expected error for unknown SKU, got nil")
	}
	if !isNotFoundStatus(err) {
		t.Errorf("expected NotFound gRPC status, got: %v", err)
	}
}

func TestReserveStock_ErrInsufficientStock_WhenQtyExceedsAvailable(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BELT-003", 2, 0)
	svc := newTestService(t, repo)

	_, err := svc.ReserveStock(context.Background(), &invpb.ReserveStockRequest{
		Sku: "LTH-BELT-003", Quantity: 5, OrderId: "ORD-002",
	})
	if err == nil {
		t.Fatal("expected error for insufficient stock, got nil")
	}
	if !isFailedPreconditionStatus(err) {
		t.Errorf("expected FailedPrecondition gRPC status, got: %v", err)
	}
}

// ── ReleaseStock ──────────────────────────────────────────────────────────────

func TestReleaseStock_RestoresAvailableAndDeletesReservation(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-CARD-004", 10, 0)
	svc := newTestService(t, repo)

	reserveResp, err := svc.ReserveStock(context.Background(), &invpb.ReserveStockRequest{
		Sku: "LTH-CARD-004", Quantity: 4, OrderId: "ORD-003",
	})
	if err != nil {
		t.Fatalf("setup reserve failed: %v", err)
	}

	releaseResp, err := svc.ReleaseStock(context.Background(), &invpb.ReleaseStockRequest{
		ReservationId: reserveResp.ReservationId,
	})
	if err != nil {
		t.Fatalf("unexpected error on release: %v", err)
	}
	if !releaseResp.Success {
		t.Error("expected success=true")
	}

	stock, _ := repo.GetStock(context.Background(), "LTH-CARD-004")
	if stock.Available != 10 {
		t.Errorf("expected available restored to 10, got %d", stock.Available)
	}
	if stock.Reserved != 0 {
		t.Errorf("expected reserved=0 after release, got %d", stock.Reserved)
	}
}

func TestReleaseStock_ErrNotFound_ForUnknownReservationID(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	_, err := svc.ReleaseStock(context.Background(), &invpb.ReleaseStockRequest{
		ReservationId: "non-existent-reservation-id",
	})
	if err == nil {
		t.Fatal("expected error for unknown reservation, got nil")
	}
	if !isNotFoundStatus(err) {
		t.Errorf("expected NotFound gRPC status, got: %v", err)
	}
}

// ── ListStock ─────────────────────────────────────────────────────────────────

func TestListStock_ReturnsAllRecords(t *testing.T) {
	repo := newFakeRepository()
	repo.seed("LTH-BAG-001", 10, 2)
	repo.seed("LTH-BELT-002", 5, 0)
	svc := newTestService(t, repo)

	resp, err := svc.ListStock(context.Background(), &invpb.ListStockRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
}

func TestListStock_EmptyInventory_ReturnsEmptyList(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	resp, err := svc.ListStock(context.Background(), &invpb.ListStockRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

// ── gRPC status helpers ───────────────────────────────────────────────────────

func isNotFoundStatus(err error) bool {
	return status.Code(err) == codes.NotFound
}

func isFailedPreconditionStatus(err error) bool {
	return status.Code(err) == codes.FailedPrecondition
}
