package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/jsanca/go-folio/inventory-service/internal/config"
	"github.com/jsanca/go-folio/inventory-service/internal/database"
	"github.com/jsanca/go-folio/inventory-service/internal/inventory"
	"github.com/jsanca/go-folio/inventory-service/internal/seed"
	"github.com/jsanca/go-folio/inventory-service/internal/server"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	repo := inventory.NewSQLiteRepository(db)
	svc := inventory.NewService(db, repo, logger)

	seed.Run(context.Background(), repo, logger)

	srv := server.New(svc, logger)
	logger.Info("grpc server listening", "addr", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		log.Fatalf("grpc server: %v", err)
	}
}
