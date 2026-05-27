package clients

import (
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

// Close releases the underlying gRPC connection.
// Implements io.Closer.
func (c *InventoryClient) Close() error {
	return c.conn.Close()
}
