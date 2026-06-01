package clients

import (
	"context"
	"fmt"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InventoryClient wraps the gRPC connection to inventory-service.
type InventoryClient struct {
	conn *grpc.ClientConn
	// Svc is the generated gRPC client; handlers call it directly.
	Svc invpb.InventoryServiceClient
}

// NewInventoryClient dials inventory-service at addr and returns a ready client.
// The connection is lazy; no network traffic occurs until the first RPC.
func NewInventoryClient(addr string) (*InventoryClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial inventory: %w", err)
	}
	return &InventoryClient{
		conn: conn,
		Svc:  invpb.NewInventoryServiceClient(conn),
	}, nil
}

// SeedSKU registers a SKU in inventory-service with zero stock by applying a
// delta of 0. This is the saga step that follows variant creation in catalog.
func (c *InventoryClient) SeedSKU(ctx context.Context, sku string) error {
	_, err := c.Svc.AdjustStock(ctx, &invpb.AdjustStockRequest{
		Sku:    sku,
		Delta:  0,
		Reason: "saga: variant created",
	})
	return err
}

// Close releases the underlying gRPC connection.
// Implements io.Closer.
func (c *InventoryClient) Close() error {
	return c.conn.Close()
}
