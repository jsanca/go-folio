package main

import (
	"context"
	"os"

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
		logger.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	catalogRT := runtime.NewCatalogRuntime(db)

	composite := runtime.NewComposite(catalogRT)
	defer func() {
	    if err := composite.Close(); err != nil {
	        logger.Warn("composite close", "err", err)
	    }
	}()

	seed.Run(context.Background(), catalogRT.CatalogSvc, logger)

	srv := server.New(catalogRT, db, logger)
	logger.Info("server listening", "addr", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}
