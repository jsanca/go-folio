package main

import (
	"context"
	"log"
	"github.com/jsanca/go-folio/internal/config"
	"github.com/jsanca/go-folio/internal/database"
	"github.com/jsanca/go-folio/internal/observability"
	"github.com/jsanca/go-folio/internal/runtime"
	"github.com/jsanca/go-folio/internal/seed"
	"github.com/jsanca/go-folio/internal/server"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()
	catalogRT := runtime.NewCatalogRuntime(db)
	composite := runtime.NewComposite(catalogRT)
	defer composite.Close() //nolint:errcheck
	seed.Run(context.Background(), catalogRT.ProductSvc, catalogRT.CatalogSvc, logger)
	srv := server.New(catalogRT, db, logger)
	logger.Info("server listening", "addr", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
