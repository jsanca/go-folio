// Package service implements the gRPC InventoryServiceServer.
//
// Transaction ownership:
//   - The service layer owns transaction boundaries (unit of work).
//   - Repositories own SQL operations but never open or commit transactions.
//   - A repository method must never call db.BeginTx or tx.Commit.
//   - The service calls WithTx to bind repository operations to a transaction
//     it controls.
//
// This separation keeps business consistency rules in the service layer
// and keeps repositories focused on persistence mechanics.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"github.com/jsanca/go-folio/inventory-service/internal/domain"
	"github.com/jsanca/go-folio/inventory-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the gRPC InventoryServiceServer interface.
// It owns the unit of work: transactional operations begin a *sql.Tx,
// bind the repository to it via WithTx, and commit or roll back.
type Service struct {
	invpb.UnimplementedInventoryServiceServer
	db     *sql.DB
	repo   repository.Repository
	logger *slog.Logger
}

// NewService creates a Service backed by the given repository and database.
// db is used exclusively to begin transactions; repo handles all data access.
func NewService(db *sql.DB, repo repository.Repository, logger *slog.Logger) *Service {
	return &Service{db: db, repo: repo, logger: logger}
}

// GetStock returns the available and reserved quantities for a SKU.
func (s *Service) GetStock(ctx context.Context, req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
	stock, err := s.repo.GetStock(ctx, req.GetSku())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "sku not found: %s", req.GetSku())
		}
		s.logger.Error("get stock", "sku", req.GetSku(), "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &invpb.GetStockResponse{
		Sku:       stock.SKU,
		Available: stock.Available,
		Reserved:  stock.Reserved,
	}, nil
}

// AdjustStock applies a delta to available stock for a SKU.
// A positive delta adds stock; a negative delta removes it.
func (s *Service) AdjustStock(ctx context.Context, req *invpb.AdjustStockRequest) (*invpb.AdjustStockResponse, error) {
	stock, err := s.inTx(ctx, func(r repository.Repository) (*domain.Stock, error) {
		return r.AdjustStock(ctx, req.GetSku(), req.GetDelta())
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "sku not found: %s", req.GetSku())
		}
		if errors.Is(err, repository.ErrInsufficientStock) {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock: %v", err)
		}
		s.logger.Error("adjust stock", "sku", req.GetSku(), "delta", req.GetDelta(), "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &invpb.AdjustStockResponse{Sku: stock.SKU, Available: stock.Available}, nil
}

// ReserveStock moves quantity from available to reserved and returns a reservation ID.
func (s *Service) ReserveStock(ctx context.Context, req *invpb.ReserveStockRequest) (*invpb.ReserveStockResponse, error) {
	res, err := s.inTxReservation(ctx, func(r repository.Repository) (*domain.Reservation, error) {
		return r.ReserveStock(ctx, req.GetSku(), req.GetQuantity(), req.GetOrderId())
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "sku not found: %s", req.GetSku())
		}
		if errors.Is(err, repository.ErrInsufficientStock) {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock: %v", err)
		}
		s.logger.Error("reserve stock", "sku", req.GetSku(), "quantity", req.GetQuantity(), "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &invpb.ReserveStockResponse{Success: true, ReservationId: res.ID}, nil
}

// ReleaseStock cancels a reservation and returns the quantity to available stock.
func (s *Service) ReleaseStock(ctx context.Context, req *invpb.ReleaseStockRequest) (*invpb.ReleaseStockResponse, error) {
	if _, err := s.inTx(ctx, func(r repository.Repository) (*domain.Stock, error) {
		return r.ReleaseStock(ctx, req.GetReservationId())
	}); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "reservation not found: %s", req.GetReservationId())
		}
		s.logger.Error("release stock", "reservation_id", req.GetReservationId(), "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &invpb.ReleaseStockResponse{Success: true}, nil
}

// ListStock returns all SKUs with their current available and reserved quantities.
func (s *Service) ListStock(ctx context.Context, _ *invpb.ListStockRequest) (*invpb.ListStockResponse, error) {
	stocks, err := s.repo.ListStock(ctx)
	if err != nil {
		s.logger.Error("list stock", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	items := make([]*invpb.StockRecord, 0, len(stocks))
	for _, st := range stocks {
		items = append(items, &invpb.StockRecord{
			Sku:       st.SKU,
			Available: st.Available,
			Reserved:  st.Reserved,
		})
	}
	return &invpb.ListStockResponse{Items: items}, nil
}

// inTx is a service-layer helper that owns the transaction boundary for stock mutations.
// It begins a transaction, binds the repository to it via WithTx, runs fn, and commits.
// defer tx.Rollback() is called unconditionally; after a successful Commit it becomes a no-op.
// Repositories must never call BeginTx or Commit — that responsibility lives here.
func (s *Service) inTx(ctx context.Context, fn func(repository.Repository) (*domain.Stock, error)) (*domain.Stock, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	result, err := fn(s.repo.WithTx(tx))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

// inTxReservation is inTx for operations that return a *domain.Reservation (no domain.Stock value).
// The same transaction ownership rules apply: the service begins and commits;
// defer tx.Rollback() is a no-op after a successful Commit.
func (s *Service) inTxReservation(ctx context.Context, fn func(repository.Repository) (*domain.Reservation, error)) (*domain.Reservation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	result, err := fn(s.repo.WithTx(tx))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}
