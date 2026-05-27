// Package database handles SQLite connection setup and migration application.
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jsanca/go-folio/internal/config"
	"github.com/jsanca/go-folio/migrations"
)

// Connect opens a SQLite database using cfg.DBPath, applies all migrations in
// order, and returns the ready-to-use connection. The caller is responsible for
// closing the returned *sql.DB.
func Connect(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.DBPath+"?_loc=UTC")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(migrations.SQL002); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migration: %w", err)
	}
	return db, nil
}
