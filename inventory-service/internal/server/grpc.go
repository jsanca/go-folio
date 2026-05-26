// Package server builds and starts the gRPC server.
package server

import (
	"fmt"
	"log/slog"
	"net"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"google.golang.org/grpc"
)

// Server wraps a gRPC server instance.
type Server struct {
	grpc *grpc.Server
}

// New creates a Server with the InventoryService registered.
func New(svc invpb.InventoryServiceServer, logger *slog.Logger) *Server {
	_ = logger // reserved for future interceptors
	grpcSrv := grpc.NewServer()
	invpb.RegisterInventoryServiceServer(grpcSrv, svc)
	return &Server{grpc: grpcSrv}
}

// Start begins listening on addr and blocks until the server returns an error.
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	return s.grpc.Serve(lis)
}
