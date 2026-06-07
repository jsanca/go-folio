// Package runtime wires all downstream clients together.
// It is the composition root for the gateway service.
package runtime

import (
	"context"
	"errors"
	"strings"

	"github.com/jsanca/go-folio/gateway-service/internal/clients"
	"github.com/jsanca/go-folio/gateway-service/internal/config"
	"github.com/jsanca/go-folio/gateway-service/internal/middleware"
	"github.com/jsanca/go-folio/gateway-service/internal/sse"
)

// GatewayRuntime holds all downstream clients for the gateway domain.
type GatewayRuntime struct {
	Catalog           *clients.CatalogClient
	Inventory         clients.InventoryClient
	Auth              *middleware.Verifier
	Events            *sse.Broker
	LowStockThreshold int
	CORSOrigins       []string
}

// NewGatewayRuntime creates and connects all downstream clients.
// ctx is used for the OIDC provider discovery request to Keycloak.
// The gRPC connection to inventory-service is lazy — no network traffic until the first RPC.
func NewGatewayRuntime(ctx context.Context, cfg config.Config) (*GatewayRuntime, error) {
	catalog := clients.NewCatalogClient(cfg.CatalogURL)

	inventory, err := clients.NewInventoryClient(cfg.InventoryAddr)
	if err != nil {
		return nil, err
	}

	auth, err := middleware.NewVerifier(ctx, cfg.KeycloakURL, cfg.KeycloakRealm)
	if err != nil {
		return nil, err
	}

	return &GatewayRuntime{
		Catalog:           catalog,
		Inventory:         inventory,
		Auth:              auth,
		Events:            sse.NewBroker(),
		LowStockThreshold: cfg.LowStockThreshold,
		CORSOrigins:       strings.Split(cfg.CORSAllowedOrigins, ","),
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
