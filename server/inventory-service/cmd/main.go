package main

import (
	"context"
	"log"

	"github.com/jsanca/go-folio/inventory-service/internal/config"
	"github.com/jsanca/go-folio/inventory-service/internal/database"
	"github.com/jsanca/go-folio/inventory-service/internal/observability"
	"github.com/jsanca/go-folio/inventory-service/internal/runtime"
	"github.com/jsanca/go-folio/inventory-service/internal/seed"
	"github.com/jsanca/go-folio/inventory-service/internal/server"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	inventoryRT := runtime.NewInventoryRuntime(db, logger)

	composite := runtime.NewComposite(inventoryRT)
	defer func() {
		if err := composite.Close(); err != nil {
			logger.Warn("composite close", "err", err)
		}
	}()

	seed.Run(context.Background(), inventoryRT.Repo, logger)

	srv := server.New(inventoryRT.Svc, logger)
	logger.Info("grpc server listening", "addr", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
