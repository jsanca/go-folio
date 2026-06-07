package clients

import (
	"context"
	"io"

	invpb "github.com/jsanca/go-folio/gen/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StockInfo represents inventory state as seen by the gateway layer.
// All protobuf details are hidden behind this type.
type StockInfo struct {
	SKU       string
	Available int
	Reserved  int
}

// InventoryClient defines the gateway's view of inventory operations.
// Implementations hide all protobuf details from callers.
type InventoryClient interface {
	io.Closer
	GetStock(ctx context.Context, sku string) (*StockInfo, error)
	ListStock(ctx context.Context) ([]StockInfo, error)
	AdjustStock(ctx context.Context, sku string, delta int, reason string) (*StockInfo, error)
	SeedSKU(ctx context.Context, sku string) error
}

// GRPCInventoryClient implements InventoryClient using the generated gRPC stub.
// All invpb.* references are confined to this file.
type GRPCInventoryClient struct {
	conn *grpc.ClientConn
	stub invpb.InventoryServiceClient
}

// NewInventoryClient dials inventory-service at addr and returns a ready client.
// The connection is lazy; no network traffic occurs until the first RPC.
func NewInventoryClient(addr string) (*GRPCInventoryClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCInventoryClient{
		conn: conn,
		stub: invpb.NewInventoryServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection. Implements io.Closer.
func (c *GRPCInventoryClient) Close() error {
	return c.conn.Close()
}

// GetStock fetches current stock levels for the given SKU.
// The raw gRPC error is returned so callers can inspect the status code.
func (c *GRPCInventoryClient) GetStock(ctx context.Context, sku string) (*StockInfo, error) {
	resp, err := c.stub.GetStock(ctx, &invpb.GetStockRequest{Sku: sku})
	if err != nil {
		return nil, err
	}
	return &StockInfo{SKU: resp.Sku, Available: int(resp.Available), Reserved: int(resp.Reserved)}, nil
}

// ListStock returns current stock levels for all registered SKUs.
func (c *GRPCInventoryClient) ListStock(ctx context.Context) ([]StockInfo, error) {
	resp, err := c.stub.ListStock(ctx, &invpb.ListStockRequest{})
	if err != nil {
		return nil, err
	}
	result := make([]StockInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		result = append(result, StockInfo{SKU: item.Sku, Available: int(item.Available), Reserved: int(item.Reserved)})
	}
	return result, nil
}

// AdjustStock applies delta to available stock for the given SKU.
// The raw gRPC error is returned so callers can inspect the status code.
func (c *GRPCInventoryClient) AdjustStock(ctx context.Context, sku string, delta int, reason string) (*StockInfo, error) {
	resp, err := c.stub.AdjustStock(ctx, &invpb.AdjustStockRequest{
		Sku:    sku,
		Delta:  int32(delta),
		Reason: reason,
	})
	if err != nil {
		return nil, err
	}
	return &StockInfo{SKU: resp.Sku, Available: int(resp.Available)}, nil
}

// SeedSKU registers a SKU in inventory-service with zero stock.
// This is the saga step that follows variant creation in catalog.
func (c *GRPCInventoryClient) SeedSKU(ctx context.Context, sku string) error {
	_, err := c.stub.AdjustStock(ctx, &invpb.AdjustStockRequest{
		Sku:    sku,
		Delta:  0,
		Reason: "saga: variant created",
	})
	return err
}
