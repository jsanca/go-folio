// Package database handles PostgreSQL connection setup and migration application.
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jsanca/go-folio/inventory-service/internal/config"
	"github.com/jsanca/go-folio/inventory-service/migrations"
)

// Connect opens a PostgreSQL database using cfg.DatabaseURL, applies all
// migrations, and returns the ready-to-use connection. The caller is
// responsible for closing it.
func Connect(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if _, err := db.Exec(migrations.SQL001); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migration: %w", err)
	}
	return db, nil
}
