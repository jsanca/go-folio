// Package inventory contains the domain types, repository interface, and
// service implementation for inventory management.
package inventory

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
)

// Stock represents the current stock levels for a single SKU.
type Stock struct {
	SKU       string
	Available int32
	Reserved  int32
}

// Reservation represents a held quantity of a SKU against an order.
type Reservation struct {
	ID       string
	SKU      string
	Quantity int32
	OrderID  string
}

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrInsufficientStock is returned when available stock is too low to fulfill a request.
var ErrInsufficientStock = errors.New("insufficient stock")

// Repository defines pure data-access operations for inventory management.
// Implementations must not manage transaction boundaries — callers own
// the unit of work and may pass a tx-scoped repository via WithTx.
type Repository interface {
	GetStock(ctx context.Context, sku string) (*Stock, error)
	AdjustStock(ctx context.Context, sku string, delta int32) (*Stock, error)
	ReserveStock(ctx context.Context, sku string, quantity int32, orderID string) (*Reservation, error)
	ReleaseStock(ctx context.Context, reservationID string) (*Stock, error)
	SeedSKU(ctx context.Context, sku string, available int32) error
	HasAnyStock(ctx context.Context) (bool, error)
	// WithTx returns a transaction-scoped Repository that executes all
	// operations within tx. The caller is responsible for Commit/Rollback.
	WithTx(tx *sql.Tx) Repository
}

// PostgresRepository implements Repository using a *sql.DB connection.
type PostgresRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a Repository backed by the given connection.
// The name is kept for backwards compatibility with existing call sites.
func NewSQLiteRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// WithTx returns a Repository that executes within tx.
func (r *PostgresRepository) WithTx(tx *sql.Tx) Repository {
	return &pgTxRepository{tx: tx}
}

// GetStock returns the current stock record for the given SKU.
func (r *PostgresRepository) GetStock(ctx context.Context, sku string) (*Stock, error) {
	return queryStock(ctx, r.db, sku)
}

// AdjustStock applies delta to available stock for sku.
func (r *PostgresRepository) AdjustStock(ctx context.Context, sku string, delta int32) (*Stock, error) {
	return adjustStock(ctx, r.db, sku, delta)
}

// ReserveStock moves quantity from available to reserved and records a reservation.
func (r *PostgresRepository) ReserveStock(ctx context.Context, sku string, quantity int32, orderID string) (*Reservation, error) {
	return reserveStock(ctx, r.db, sku, quantity, orderID)
}

// ReleaseStock cancels a reservation and returns its quantity to available.
func (r *PostgresRepository) ReleaseStock(ctx context.Context, reservationID string) (*Stock, error) {
	return releaseStock(ctx, r.db, reservationID)
}

// SeedSKU inserts a SKU with initial stock if it does not already exist.
func (r *PostgresRepository) SeedSKU(ctx context.Context, sku string, available int32) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stock (sku, available, reserved) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING`,
		sku, available,
	)
	return err
}

// HasAnyStock reports whether any stock records exist.
func (r *PostgresRepository) HasAnyStock(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock`).Scan(&count)
	return count > 0, err
}

// pgTxRepository implements Repository using a *sql.Tx.
type pgTxRepository struct {
	tx *sql.Tx
}

// WithTx returns a Repository bound to the given tx, replacing the current one.
func (r *pgTxRepository) WithTx(tx *sql.Tx) Repository {
	return &pgTxRepository{tx: tx}
}

// GetStock returns the current stock record within the transaction.
func (r *pgTxRepository) GetStock(ctx context.Context, sku string) (*Stock, error) {
	return queryStock(ctx, r.tx, sku)
}

// AdjustStock applies delta to available stock within the transaction.
func (r *pgTxRepository) AdjustStock(ctx context.Context, sku string, delta int32) (*Stock, error) {
	return adjustStock(ctx, r.tx, sku, delta)
}

// ReserveStock moves quantity to reserved within the transaction.
func (r *pgTxRepository) ReserveStock(ctx context.Context, sku string, quantity int32, orderID string) (*Reservation, error) {
	return reserveStock(ctx, r.tx, sku, quantity, orderID)
}

// ReleaseStock cancels a reservation within the transaction.
func (r *pgTxRepository) ReleaseStock(ctx context.Context, reservationID string) (*Stock, error) {
	return releaseStock(ctx, r.tx, reservationID)
}

// SeedSKU inserts a SKU within the transaction.
func (r *pgTxRepository) SeedSKU(ctx context.Context, sku string, available int32) error {
	_, err := r.tx.ExecContext(ctx,
		`INSERT INTO stock (sku, available, reserved) VALUES ($1, $2, 0) ON CONFLICT DO NOTHING`,
		sku, available,
	)
	return err
}

// HasAnyStock reports whether any stock records exist within the transaction.
func (r *pgTxRepository) HasAnyStock(ctx context.Context) (bool, error) {
	var count int
	err := r.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock`).Scan(&count)
	return count > 0, err
}

// querier is the common subset of *sql.DB and *sql.Tx used by the shared helpers.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func queryStock(ctx context.Context, q querier, sku string) (*Stock, error) {
	var s Stock
	err := q.QueryRowContext(ctx,
		`SELECT sku, available, reserved FROM stock WHERE sku = $1`, sku,
	).Scan(&s.SKU, &s.Available, &s.Reserved)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: sku=%s", ErrNotFound, sku)
	}
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}
	return &s, nil
}

func adjustStock(ctx context.Context, q querier, sku string, delta int32) (*Stock, error) {
	s, err := queryStock(ctx, q, sku)
	if err != nil {
		return nil, err
	}
	newAvailable := s.Available + delta
	if newAvailable < 0 {
		return nil, fmt.Errorf("%w: sku=%s delta=%d available=%d", ErrInsufficientStock, sku, delta, s.Available)
	}
	if _, err = q.ExecContext(ctx,
		`UPDATE stock SET available = $1 WHERE sku = $2`, newAvailable, sku,
	); err != nil {
		return nil, fmt.Errorf("update stock: %w", err)
	}
	s.Available = newAvailable
	return s, nil
}

func reserveStock(ctx context.Context, q querier, sku string, quantity int32, orderID string) (*Reservation, error) {
	s, err := queryStock(ctx, q, sku)
	if err != nil {
		return nil, err
	}
	if s.Available < quantity {
		return nil, fmt.Errorf("%w: sku=%s requested=%d available=%d", ErrInsufficientStock, sku, quantity, s.Available)
	}
	id := newID()
	if _, err = q.ExecContext(ctx,
		`INSERT INTO reservations (id, sku, quantity, order_id) VALUES ($1, $2, $3, $4)`,
		id, sku, quantity, orderID,
	); err != nil {
		return nil, fmt.Errorf("insert reservation: %w", err)
	}
	if _, err = q.ExecContext(ctx,
		`UPDATE stock SET available = available - $1, reserved = reserved + $2 WHERE sku = $3`,
		quantity, quantity, sku,
	); err != nil {
		return nil, fmt.Errorf("update stock for reserve: %w", err)
	}
	return &Reservation{ID: id, SKU: sku, Quantity: quantity, OrderID: orderID}, nil
}

func releaseStock(ctx context.Context, q querier, reservationID string) (*Stock, error) {
	var res Reservation
	err := q.QueryRowContext(ctx,
		`SELECT id, sku, quantity, order_id FROM reservations WHERE id = $1`, reservationID,
	).Scan(&res.ID, &res.SKU, &res.Quantity, &res.OrderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: reservation_id=%s", ErrNotFound, reservationID)
	}
	if err != nil {
		return nil, fmt.Errorf("get reservation: %w", err)
	}
	if _, err = q.ExecContext(ctx,
		`UPDATE stock SET available = available + $1, reserved = reserved - $2 WHERE sku = $3`,
		res.Quantity, res.Quantity, res.SKU,
	); err != nil {
		return nil, fmt.Errorf("update stock for release: %w", err)
	}
	if _, err = q.ExecContext(ctx,
		`DELETE FROM reservations WHERE id = $1`, reservationID,
	); err != nil {
		return nil, fmt.Errorf("delete reservation: %w", err)
	}
	return queryStock(ctx, q, res.SKU)
}

// newID generates a random UUID v4.
func newID() string {
	var b [16]byte
	rand.Read(b[:]) //nolint:errcheck
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
