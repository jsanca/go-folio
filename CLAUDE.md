# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

The repo is split into `server/` (Go workspace) and `client/` (React SPA). The Go workspace (`go.work`) has four modules under `server/`: `catalog-service`, `inventory-service`, `gateway-service`, and `gen`.

```bash
# Run tests for a single service
cd server/catalog-service && go test ./...
cd server/inventory-service && go test ./...

# Run a single test
cd server/catalog-service && go test ./internal/service/ -run TestCatalogService_ListProducts

# Build a service
cd server/catalog-service && go build ./cmd/...

# Run a service locally
cd server/catalog-service && go run ./cmd/main.go

# Regenerate protobuf/gRPC stubs (requires buf CLI)
buf generate

# Lint proto files
buf lint

# Start all services with Docker
docker compose up --build

# Client dev server
cd client && npm install && npm run dev
```

## Architecture

### Repo layout

```
server/   → Go workspace: catalog-service, inventory-service, gateway-service, gen
client/   → React SPA (Vite + React)
proto/    → Protobuf definitions
go.work   → Workspace root
```

### Three services, one workspace

- **catalog-service** — REST HTTP API (chi router, port 8080) backed by SQLite. Manages the leather goods product catalog for a fictional store. Serves product, variant, and image data plus cursor-paginated sync endpoints for dotCMS integration.
- **inventory-service** — gRPC server (port 9090) backed by SQLite. Manages stock levels, reservations, and adjustments. Proto definitions live in `proto/inventory/`, generated stubs in `server/gen/`.
- **gateway-service** — REST HTTP API (chi router, port 8090). Aggregates catalog and inventory into a single response. HTTP client to catalog-service, gRPC client to inventory-service. Derives `StockStatus` from inventory data using `LOW_STOCK_THRESHOLD` env var (default 5).

### Catalog-service layers (inside `internal/`)

```
domain/       → Pure types and validation (Product, ProductVariant, Money, SyncCursor, …)
repository/   → Interfaces (CatalogProductRepository, ProductVariantRepository, …) + SQLite impls
service/      → Interfaces (CatalogService, ProductService) + default implementations; cursor encoding lives here
handler/      → HTTP handlers; thin layer that parses request, calls service, writes JSON
runtime/      → Composition root — wires repos → services into CatalogRuntime
server/       → Chi router setup, middleware registration (PanicRecovery, RequestLogger, Prometheus metrics)
observability/→ Structured logging (slog), Prometheus metrics middleware, health/ready handlers
config/       → Reads env vars; Config struct
database/     → Opens SQLite, runs embedded migrations in order
seed/         → Idempotent seed data applied at startup
```

`runtime.NewCatalogRuntime` is the single wiring point. `main.go` constructs config → DB → runtime → server, then calls `seed.Run` before starting the HTTP listener.

### CatalogService interface split

`CatalogService` takes **four** separate repository interfaces (`CatalogProductRepository`, `ProductVariantRepository`, `ProductImageRepository`, `CatalogSyncRepository`). The SQLite implementation (`sqlite_catalog_repository.go`) satisfies all four. The split is intentional: interface segregation keeps tests minimal and service dependencies explicit.

### Sync / cursor pagination

`GET /catalog/product-projections` and `GET /catalog/variant-inventory` return cursor-paginated responses. The cursor is a base64-encoded JSON struct (`domain.SyncCursor{UpdatedAt, ID}`). Encoding/decoding lives in `service/cursor.go`. Callers pass `?cursor=<token>&pageSize=<n>&updatedSince=<ISO-8601>`.

**Known limitation:** `ListProductProjections` filters by `catalog_products.updated_at` only — variant-only or image-only changes are invisible unless the product row is also touched.

### Inventory-service layers (inside `internal/`)

```
config/       → Env var config
database/     → SQLite connection + migrations
inventory/    → Domain types, SQLite repository, service (implements generated gRPC interface)
runtime/      → Composition root — wires repo + service into InventoryRuntime; CompositeRuntime for lifecycle
observability/→ Structured logging (slog)
seed/         → Startup seed data
server/       → gRPC server wiring
```

`runtime.NewInventoryRuntime` is the single wiring point, mirroring the catalog-service pattern.

### Gateway-service layers (inside `internal/`)

```
config/       → Env var config (PORT, CATALOG_URL, INVENTORY_ADDR, LOW_STOCK_THRESHOLD)
observability/→ Structured logging (slog)
clients/      → catalog.go (HTTP) and inventory.go (gRPC); both implement io.Closer
runtime/      → Composition root — wires clients into GatewayRuntime; CompositeRuntime for lifecycle
server/       → server.go (chi setup) + routes.go + products.go (handlers) + response.go (writeJSON/writeError)
```

`runtime.NewGatewayRuntime` is the wiring point. It returns an error because `grpc.NewClient` can fail on invalid options. The gRPC connection itself is lazy — no network traffic until the first RPC call.

## Go Code Quality

### Naming
- Receivers: `svc` for `*Service`, `sqlRepo` for `*SQLiteRepository`, `txRepo` for `*sqliteTxRepository`
- Never single-letter params: `r Repository` → `repo`, `fn` ok when type is self-documenting
- Reservations: `reservation` not `res`
- Transactions: `tx`, never `t`

### Comments
- Every exported type, interface, func and method needs a godoc comment
- Format: `// Name does/returns/is ...`

### Tests
- Table-driven with `t.Run`
- Interface-based fakes in `_test.go`
- Cover happy path + each sentinel error

## Runtime Pattern

All services follow the same startup pattern established in `server/catalog-service`.
`main.go` must stay thin: load config, wire dependencies, hand off to a runtime
or server abstraction that owns lifecycle (start, graceful shutdown, signal
handling). Initialization logic, database setup, and seeding belong in dedicated
packages — not in `main`. When adding a new service, use `catalog-service` as
the canonical reference.
