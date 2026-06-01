# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

The repo is split into `server/` (Go workspace) and `client/` (two React SPAs). The Go workspace (`go.work`) has four modules under `server/`: `catalog-service`, `inventory-service`, `gateway-service`, and `gen`.

```bash
# Run tests for a single service
cd server/catalog-service && go test ./...
cd server/inventory-service && go test ./...
cd server/gateway-service && go test ./...

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

# Start all services with Docker (includes store on :3000 and admin on :3001 with hot reload)
docker compose up --build

# Run a client app locally (outside Docker)
cd client/store && npm install && npm run dev   # http://localhost:3000
cd client/admin && npm install && npm run dev   # http://localhost:3001
```

## Architecture

### Repo layout

```
server/   → Go workspace: catalog-service, inventory-service, gateway-service, gen
client/
├── store/  → Customer storefront (Vite + React, port 3000)
└── admin/  → Catalog & inventory management (Vite + React, port 3001)
proto/    → Protobuf definitions
go.work   → Workspace root
```

### Three services, one workspace

- **catalog-service** — REST HTTP API (chi router, port 8080) backed by PostgreSQL (`folio_catalog` DB). Manages the leather goods product catalog for a fictional store. Serves product, variant, and image data plus cursor-paginated sync endpoints for dotCMS integration.
- **inventory-service** — gRPC server (port 9090) backed by PostgreSQL (`folio_inventory` DB). Manages stock levels, reservations, and adjustments. Proto definitions live in `proto/inventory/`, generated stubs in `server/gen/`.
- **gateway-service** — REST HTTP API (chi router, port 8090). Aggregates catalog and inventory into a single response. HTTP client to catalog-service, gRPC client to inventory-service. Derives `StockStatus` from inventory data using `LOW_STOCK_THRESHOLD` env var (default 5).

### Catalog-service layers (inside `internal/`)

```
domain/       → Pure types and validation (Product, ProductVariant, Money, SyncCursor, …)
repository/   → Interfaces (CatalogProductRepository, ProductVariantRepository, …) + PostgreSQL impl
service/      → Interfaces (CatalogService, ProductService) + default implementations; cursor encoding lives here
handler/      → HTTP handlers; thin layer that parses request, calls service, writes JSON
runtime/      → Composition root — wires repos → services into CatalogRuntime
server/       → Chi router setup, middleware registration (PanicRecovery, RequestLogger, Prometheus metrics)
observability/→ Structured logging (slog), Prometheus metrics middleware, health/ready handlers
config/       → Reads env vars; Config struct (DATABASE_URL, PORT)
database/     → Opens PostgreSQL via pgx/v5/stdlib, runs embedded migrations in order
seed/         → Idempotent seed data applied at startup
```

`runtime.NewCatalogRuntime` is the single wiring point. `main.go` constructs config → DB → runtime → server, then calls `seed.Run` before starting the HTTP listener.

### CatalogService interface split

`CatalogService` takes **four** separate repository interfaces (`CatalogProductRepository`, `ProductVariantRepository`, `ProductImageRepository`, `CatalogSyncRepository`). The PostgreSQL implementation (`postgres_catalog_repository.go`, struct `PostgresCatalogRepository`) satisfies all four. The split is intentional: interface segregation keeps tests minimal and service dependencies explicit.

### Sync / cursor pagination

`GET /catalog/product-projections` and `GET /catalog/variant-inventory` return cursor-paginated responses. The cursor is a base64-encoded JSON struct (`domain.SyncCursor{UpdatedAt, ID}`). Encoding/decoding lives in `service/cursor.go`. Callers pass `?cursor=<token>&pageSize=<n>&updatedSince=<ISO-8601>`.

**Known limitation:** `ListProductProjections` filters by `catalog_products.updated_at` only — variant-only or image-only changes are invisible unless the product row is also touched.

### Inventory-service layers (inside `internal/`)

```
domain/       → Pure types: Stock, Reservation
repository/   → Repository interface + PostgreSQL impl; WithTx pattern for transaction scoping
service/      → Implements generated gRPC interface; owns transaction lifecycle via withTx helper
config/       → Env var config (DATABASE_URL, PORT)
database/     → PostgreSQL connection via pgx/v5/stdlib + migrations
observability/→ Structured logging (slog)
runtime/      → Composition root — wires repo + service into InventoryRuntime; CompositeRuntime for lifecycle
seed/         → Startup seed data
server/       → gRPC server wiring (grpc.go)
```

`runtime.NewInventoryRuntime` is the single wiring point, mirroring the catalog-service pattern.

The service layer owns transactions (not the repository). `Repository.WithTx(*sql.Tx)` returns a transaction-scoped copy; a private `withTx` helper on the service handles `BeginTx`, commit, and rollback uniformly. Only inventory-service uses this pattern — catalog operations are currently single-row.

### Gateway-service layers (inside `internal/`)

```
config/       → Env var config (PORT, CATALOG_URL, INVENTORY_ADDR, LOW_STOCK_THRESHOLD, KEYCLOAK_URL, KEYCLOAK_REALM)
observability/→ Structured logging (slog)
middleware/   → auth.go — OIDC token verifier (RequireAuth, RequireRole); nil inner verifier = permissive mode
clients/      → catalog.go (HTTP) and inventory.go (gRPC); both implement io.Closer
runtime/      → Composition root — wires clients + auth verifier into GatewayRuntime; CompositeRuntime for lifecycle
server/       → server.go (chi setup) + routes.go + handlers per concern + response.go (writeJSON/writeError)
```

`runtime.NewGatewayRuntime` is the wiring point. It returns an error because `grpc.NewClient` can fail on invalid options. The gRPC connection itself is lazy — no network traffic until the first RPC call.

**Auth routes:**

| Route | Auth |
|---|---|
| `GET /products` | Public |
| `GET /products/{id}` | Public |
| `GET /admin/products` | `admin` role |
| `POST /admin/products` | `admin` role |
| `PATCH /admin/products/{id}` | `admin` role |
| `DELETE /admin/products/{id}` | `admin` role |
| `GET /admin/inventory` | `admin` role |
| `GET /admin/inventory/{sku}` | `admin` role |
| `PUT /admin/inventory/{sku}` | `admin` role |

## Go Code Quality

### Naming
- Receivers: `svc` for `*Service`, `r` for `*PostgresCatalogRepository`/`*PostgresRepository`, `tx` for transaction repos
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

## Keycloak

`docker compose up` is fully self-contained — no manual realm setup required.

- The `folio` realm is auto-imported from `scripts/keycloak/folio-realm.json` via `--import-realm`.
- Keycloak only imports the file when the realm does not already exist (idempotent on existing volumes).
- Public client: `gateway` (renamed from `folio-public`). Redirect URIs: `localhost:3000/*`, `localhost:3001/*`, `localhost:8090/*`.
- Realm roles: `admin`, `customer`.
- Test users: `admin@folio.dev / admin123` (admin role), `customer@folio.dev / customer123` (customer role).
- `KC_HOSTNAME_URL=http://keycloak:8080` pins the JWT issuer to the Docker-internal URL; gateway uses `SkipIssuerCheck: true` for dev.
- **Permissive mode:** when `KEYCLOAK_URL` is unset the gateway's `middleware.Verifier` holds a nil inner verifier — all requests pass without authentication. Useful for running locally without Keycloak.
- **Port note:** `docker compose` exposes Keycloak on host port `8080`. The admin SPA defaults to `http://localhost:8180` when `VITE_KEYCLOAK_URL` is not set. When running the client outside Docker, set `VITE_KEYCLOAK_URL=http://localhost:8080` to match the compose port.

**To re-export the realm** (after changes in the Admin UI):
```bash
docker compose exec keycloak /opt/keycloak/bin/kc.sh export \
  --dir /tmp/realm-export --realm folio --users realm_file
docker compose cp keycloak:/tmp/realm-export/folio-realm.json scripts/keycloak/folio-realm.json
```

## Engineering Failure Reports

When you detect a structural or architectural failure — a duplicate domain
stack, conflicting sources of truth, a hidden coupling that caused significant
rework — create an EFR in `docs/efr/` using the format described in
`docs/efr/README.md`. Number sequentially (`EFR-0002.md`, …). The learning
section of each EFR is the canonical output; absorb it into working practices
or this file as appropriate.

## Runtime Pattern

All services follow the same startup pattern established in `server/catalog-service`.
`main.go` must stay thin: load config, wire dependencies, hand off to a runtime
or server abstraction that owns lifecycle (start, graceful shutdown, signal
handling). Initialization logic, database setup, and seeding belong in dedicated
packages — not in `main`. When adding a new service, use `catalog-service` as
the canonical reference.
