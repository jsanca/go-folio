// Package runtime wires all downstream clients together.
// It is the composition root for the gateway service.
package runtime

import (
	"errors"

	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/config"
)

// GatewayRuntime holds all downstream clients for the gateway domain.
type GatewayRuntime struct {
	Catalog           *clients.CatalogClient
	Inventory         *clients.InventoryClient
	LowStockThreshold int
}

// NewGatewayRuntime creates and connects all downstream clients.
// The gRPC connection to inventory-service is lazy; no network traffic occurs until the first RPC.
func NewGatewayRuntime(cfg config.Config) (*GatewayRuntime, error) {
	catalog := clients.NewCatalogClient(cfg.CatalogURL)
	inventory, err := clients.NewInventoryClient(cfg.InventoryAddr)
	if err != nil {
		return nil, err
	}
	return &GatewayRuntime{
		Catalog:           catalog,
		Inventory:         inventory,
		LowStockThreshold: cfg.LowStockThreshold,
	}, nil
}

// Close releases all client connections owned by GatewayRuntime.
// Implements io.Closer.
func (rt *GatewayRuntime) Close() error {
	return errors.Join(
		rt.Inventory.Close(),
		rt.Catalog.Close(),
	)
}
