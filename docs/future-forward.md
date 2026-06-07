

# Go-folio Future-Forward Engineering Notes

## Purpose

This document captures future-forward improvements for **go-folio**, a learning-oriented full-stack retail platform built with Go microservices, React frontends, PostgreSQL, gRPC, REST, SSE, Keycloak/OIDC, MinIO, and a migration pipeline fed by a Silver product export.

The current project successfully explores several production-style patterns:

* Explicit composition roots.
* Service boundaries.
* REST gateway.
* gRPC internal service communication.
* Transaction ownership in the service layer.
* Repository abstractions.
* Saga-style compensation.
* SSE for real-time stock updates.
* Keycloak/OIDC integration.
* Medallion-style product migration from Silver exports.
* Admin and public frontend separation.

The following items are not criticisms of the project. They are documented as a forward-looking improvement backlog to harden the system, improve idiomatic Go design, and prepare the architecture for more production-like scenarios.

---

# 1. Inventory Service Improvements

## 1.1 Make inventory mutations concurrency-safe

Current stock operations are easy to understand, but some flows read stock state and then update it in a separate step. Under concurrent requests, this can cause race conditions.

Example risk:

```text
Request A reads available = 5
Request B reads available = 5
Both reserve 5
Both succeed unless the database operation is atomic
```

Future improvement:

Use row-level locking:

```sql
SELECT sku, available, reserved
FROM stock
WHERE sku = $1
FOR UPDATE;
```

Or preferably use atomic conditional updates:

```sql
UPDATE stock
SET available = available - $1,
    reserved = reserved + $1
WHERE sku = $2
  AND available >= $1
RETURNING sku, available, reserved;
```

This keeps validation and mutation in the same database operation.

---

## 1.2 Add explicit input validation at the gRPC boundary

The inventory gRPC service should validate basic request shape before reaching the repository layer.

Examples:

```text
- SKU must not be empty.
- Quantity must be greater than zero.
- Order ID must not be empty for reservations.
- Delta must follow the business rules for stock adjustment.
```

Invalid input should return:

```go
codes.InvalidArgument
```

This keeps invalid data out of the persistence layer and makes the API contract clearer.

---

## 1.3 Add an explicit `RegisterSKU` / `SeedSKU` RPC

The gateway currently needs to initialize inventory when a catalog variant becomes sellable. Conceptually, this is not a stock adjustment; it is inventory registration.

The repository already has a `SeedSKU`-style operation. This intent should be exposed explicitly through the protobuf contract.

Possible RPC:

```proto
rpc RegisterSKU(RegisterSKURequest) returns (RegisterSKUResponse);
```

This would make the saga clearer:

```text
1. Create variant in catalog.
2. Register SKU in inventory with zero stock.
3. If inventory registration fails, compensate by deleting the catalog variant.
```

This is more expressive than using `AdjustStock(delta = 0)`.

---

## 1.4 Return richer stock state from mutation operations

Some stock-changing operations currently return partial inventory state. For SSE and admin UI updates, it would be useful for inventory responses to include full stock state:

```text
- SKU
- Available quantity
- Reserved quantity
- Derived status
- Last updated timestamp, if available
```

This would let the gateway publish more accurate SSE events without guessing or setting placeholder values.

---

## 1.5 Separate gRPC adapter from application service if inventory grows

The current inventory service implements the generated gRPC interface and owns use-case transaction logic in the same struct.

This is acceptable while the service is small. If it grows, split it into:

```text
grpc adapter
  - protobuf request/response mapping
  - gRPC status code mapping

application service
  - transaction boundary
  - business use cases
  - domain-level decisions

repository
  - SQL persistence
```

This would keep transport concerns separate from business orchestration.

---

## 1.6 Strengthen transaction tests

Inventory should have service-level tests that verify transaction behavior around:

```text
- Successful stock adjustment commits.
- Failed reservation rolls back.
- Insufficient stock does not mutate state.
- Release reservation updates reservation and stock consistently.
- Transaction commit failures are surfaced correctly.
```

These tests should focus on use-case consistency, not only SQL correctness.

---

# 2. Repository and Persistence Improvements

## 2.1 Keep transaction ownership in the service layer

The current direction is correct: repositories should not own transaction boundaries. The service layer owns the unit of work because it knows the business use case.

The repository should focus on persistence operations:

```text
Repository owns SQL.
Service owns use-case consistency.
```

This principle should remain documented and protected.

---

## 2.2 Keep using `WithTx`, but document the pattern

The `WithTx(*sql.Tx)` pattern is a good explicit way to bind repository operations to a transaction.

Document the intended usage:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

repo := baseRepo.WithTx(tx)

if err := repo.OperationA(ctx); err != nil {
    return err
}

if err := repo.OperationB(ctx); err != nil {
    return err
}

return tx.Commit()
```

This makes transaction flow explicit and avoids hiding transaction boundaries inside repository methods.

---

## 2.3 Rename legacy SQLite constructors

Some constructors still contain SQLite naming even though the implementation is PostgreSQL-backed.

Example:

```go
NewSQLiteRepository(...)
```

Future direction:

```go
NewPostgresRepository(...)
```

or a database-neutral name:

```go
NewRepository(...)
```

The current naming can confuse future readers and interview reviewers.

---

## 2.4 Check `crypto/rand` errors in ID generation

If UUID-like IDs are generated using `crypto/rand`, the error should not be ignored.

Preferred direction:

```go
if _, err := rand.Read(b[:]); err != nil {
    return "", fmt.Errorf("generate id: %w", err)
}
```

For production-like code, either handle the error or use a well-known UUID library.

---

## 2.5 Keep repository errors semantic and wrapped

The repository already uses semantic errors such as “not found” or “insufficient stock”. Continue wrapping them with `%w` so upper layers can use:

```go
errors.Is(err, repository.ErrNotFound)
```

or preferably service/domain-level equivalents.

This is important for clean error mapping across layers.

---

# 3. Gateway Service Improvements

## 3.1 Move saga orchestration out of the HTTP handler if it grows

The gateway currently coordinates a small saga when creating a catalog variant:

```text
1. Create variant in catalog-service.
2. Register/seed SKU in inventory-service.
3. If inventory fails, compensate by deleting the catalog variant.
```

This is acceptable while the flow is small.

If the flow grows, move orchestration into a gateway application service:

```text
handler/
  admin_products_handler.go

service/
  variant_saga_service.go
```

The handler should remain an HTTP adapter. The saga service should own the use case.

---

## 3.2 Persist failed compensation attempts

Currently, if compensation fails, the gateway logs the failure. For production-like reliability, failed compensations should be persisted.

Possible approaches:

```text
- Store failed saga records in a database table.
- Retry failed compensations asynchronously.
- Emit alerts for manual reconciliation.
- Use an outbox pattern for reliable workflow tracking.
```

This would make cross-service consistency failures recoverable instead of only visible in logs.

---

## 3.3 Hide generated gRPC stubs behind gateway client façade methods

The inventory client currently exposes generated protobuf stubs directly in some flows. This is practical, but it couples gateway handlers to protobuf details.

Future direction:

```go
Inventory.AdjustStock(ctx, sku, delta, reason)
Inventory.GetStock(ctx, sku)
Inventory.RegisterSKU(ctx, sku)
Inventory.ReserveStock(ctx, sku, quantity, orderID)
```

Generated protobuf types should stay closer to the infrastructure boundary. Handlers should deal with gateway-level concepts.

---

## 3.4 Add downstream client timeouts and deadlines

The gateway should use explicit timeouts for downstream HTTP and gRPC calls.

For HTTP:

```go
http.Client{
    Timeout: 5 * time.Second,
}
```

For gRPC, apply request-scoped deadlines:

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

Timeout values should ideally come from configuration.

---

## 3.5 Improve downstream error propagation

Some catalog client methods return generic errors for unexpected HTTP statuses.

Current style:

```go
return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
```

Improved style:

```go
return nil, &CatalogError{
    Status: resp.StatusCode,
    Body: body,
}
```

This allows the gateway to preserve catalog-service error shape where appropriate.

---

## 3.6 Clarify SSE ownership

Inventory owns stock state and stock mutations. The gateway owns browser-facing SSE delivery.

Document this clearly:

```text
Inventory-service owns inventory consistency.
Gateway-service exposes inventory-related changes to browser clients through SSE.
Browser clients should not connect directly to inventory-service.
```

This avoids confusion between domain ownership and delivery mechanism.

---

## 3.7 Avoid handler-to-handler dependencies

The admin products handler currently reuses behavior through another handler. This is acceptable for a prototype, but as the gateway grows, shared behavior should move to a helper or application service.

Preferred direction:

```text
PublicProductsHandler
AdminProductsHandler
        ↓
ProductAggregationService / GatewayCatalogService
```

Handlers should adapt HTTP. Shared business/application behavior should live outside handlers.

---

## 3.8 Document permissive auth as local-development only

When Keycloak is not configured, the gateway can run in permissive mode. This is useful for local development but dangerous if accidentally enabled in production.

Document and enforce:

```text
- Permissive auth is local-dev only.
- Production startup should fail if auth is not configured.
- Logs should clearly warn when permissive mode is active.
```

Optional hardening:

```text
If ENV=production and KEYCLOAK_URL is empty, fail startup.
```

---

# 4. Catalog Service Improvements

## 4.1 Avoid repository errors leaking into HTTP handlers

Some catalog handlers currently import repository-level errors and map them directly to HTTP responses.

Cleaner boundary:

```text
repository error
  → service/domain semantic error
  → HTTP handler maps semantic error to HTTP response
```

Handlers should ideally know about service/domain errors, not repository implementation details.

Future direction:

```go
errors.Is(err, service.ErrProductNotFound)
```

instead of:

```go
errors.Is(err, repository.ErrProductNotFound)
```

This keeps the HTTP layer independent from persistence details.

---

## 4.2 Normalize catalog route structure

Catalog routes currently mix different prefixes, such as:

```text
/products
/catalog/products/{id}/variants
/catalog/product-projections
/catalog/variants/{sku}
```

Define a consistent API route convention.

Possible direction:

```text
/products
/products/{id}
/products/{id}/variants
/products/slug/{slug}

/sync/product-projections
/sync/variant-inventory
```

Or keep everything consistently under `/catalog`.

---

## 4.3 Clarify product listing vs product projection endpoints

Some endpoints that look like normal product listing endpoints may return product projections intended for synchronization.

Future direction:

```text
/products
  - public/admin product listing

/catalog/product-projections
  - synchronization projection endpoint
```

This makes API intent clearer.

---

## 4.4 Clarify variant lookup response shape

The variant lookup endpoint may return a product projection with one variant and empty images.

Future direction:

```text
- Return a dedicated VariantLookupResponse, or
- Return a complete product projection including images, if projection semantics are intended.
```

This avoids surprising API consumers.

---

## 4.5 Remove unused helper parameters

Some helper functions contain unused parameters, for example a page-size parser receiving a default value that is not used.

Future direction:

```go
func parsePageSizeQP(w http.ResponseWriter, r *http.Request, param string) (int, bool)
```

Keep helpers small and noise-free.

---

## 4.6 Preserve the PATCH DTO pattern

The catalog handler uses pointer fields for PATCH requests. This is a good pattern because it distinguishes omitted fields from zero values.

Example:

```go
type updateProductRequest struct {
    Title  *string `json:"title"`
    Active *bool   `json:"active"`
}
```

This should be preserved and documented as intentional.

---

# 5. Runtime, Composition, and Lifecycle Improvements

## 5.1 Preserve thin `main.go` files

The current structure is strong:

```text
main.go
  - load config
  - create logger
  - connect resources
  - build runtime
  - run seed if needed
  - start server
```

The real dependency wiring lives in runtime packages. This prevents large procedural `main.go` files and keeps startup readable.

Continue preserving:

```text
main.go stays thin.
runtime package is the composition root.
```

---

## 5.2 Keep resource ownership explicit

Document who owns and closes resources.

Example:

```text
If main creates *sql.DB, main closes it.
If runtime creates clients, runtime closes them.
CompositeRuntime closes registered resources in reverse order.
```

This matters because Go `defer` runs in LIFO order.

Example:

```go
defer db.Close()
defer runtime.Close()
```

At shutdown:

```text
runtime.Close()
db.Close()
```

This is useful when runtime resources depend on the database.

---

## 5.3 Rename `CompositeRuntime` if it becomes generic

`CompositeRuntime` works as a lifecycle helper, but if it is used generically across services, a clearer name may be:

```go
CompositeCloser
```

or:

```go
CloserGroup
```

This is not urgent, but it may better express the type’s responsibility.

---

## 5.4 Add nil-safety to composite closer

Current composite closer assumes all closers are non-nil. A defensive improvement:

```go
for i := len(c.closers) - 1; i >= 0; i-- {
    if c.closers[i] == nil {
        continue
    }
    if err := c.closers[i].Close(); err != nil {
        errs = append(errs, err)
    }
}
```

This avoids shutdown panics if a nil closer is accidentally registered.

---

## 5.5 Use structured logging consistently during startup

Some startup errors currently use the standard library logger:

```go
log.Fatalf(...)
```

while the rest of the service uses structured `slog`.

Future direction:

```go
logger.Error("connect database", "err", err)
os.Exit(1)
```

This keeps logs uniform across startup and runtime.

---

## 5.6 Add graceful shutdown

Services should support graceful shutdown, especially for Docker/Kubernetes-style environments.

For HTTP:

```go
http.Server.Shutdown(ctx)
```

For gRPC:

```go
grpcServer.GracefulStop()
```

The goal is:

```text
1. Stop accepting new requests.
2. Allow in-flight requests to complete.
3. Close resources cleanly.
```

---

# 6. Observability Improvements

## 6.1 Add gRPC interceptors

The inventory gRPC server currently registers the service directly. Future hardening should add unary interceptors for:

```text
- Structured logging
- Panic recovery
- Metrics
- Request/correlation IDs
- Authentication or service-to-service authorization if needed
```

This would align gRPC observability with the HTTP gateway.

---

## 6.2 Add consistent request IDs / correlation IDs

Requests crossing gateway → catalog → inventory should carry a correlation ID.

Future direction:

```text
- Gateway receives or creates X-Request-ID.
- Gateway sends it to catalog over HTTP.
- Gateway sends it to inventory via gRPC metadata.
- Logs include the request ID across services.
```

This would make debugging distributed flows easier.

---

## 6.3 Add tracing later

Once service boundaries stabilize, distributed tracing would be useful.

Possible future technology:

```text
OpenTelemetry
```

Useful traces:

```text
gateway request
  → catalog HTTP call
  → inventory gRPC call
  → SSE publish
```

This is not required immediately, but it fits the architecture.

---

# 7. Migration and Seed Script Improvements

## 7.1 Document the data lineage

The seed script consumes the Silver output of the product extraction pipeline.

Data lineage:

```text
myIR / SYJ extraction
  → Bronze discovered pages/cards/products
  → Silver normalized canonical products
  → images grouped by product slug
  → seed-catalog script
  → gateway admin API
  → catalog-service
  → inventory-service via saga
  → MinIO for product images
```

This should be documented because it connects the migration script to the larger extraction/medallion architecture.

---

## 7.2 Keep migration going through the gateway

The current direction is correct: the migration script calls the gateway/admin API instead of writing directly into service databases.

This preserves service boundaries:

```text
Migration script behaves like an external admin client.
Gateway coordinates catalog and inventory.
Catalog owns product data.
Inventory owns stock data.
```

This also reuses the normal variant creation saga.

---

## 7.3 Make the seed script idempotent beyond product conflicts

Currently, if product creation returns conflict, the script may skip the entire product.

Future direction:

```text
If product already exists:
  - fetch it by slug or product code
  - reconcile variants
  - reconcile images
  - update primary image if needed
```

This makes repeated migrations safer.

---

## 7.4 Use Silver image metadata as the source of truth

The Silver product structure contains image metadata, but the script may also scan the image directory directly.

Future direction:

```text
Use Silver image metadata to preserve:
- image ordering
- alt text
- source URL
- intended product association
```

Directory scanning can remain a fallback, but Silver should be the authoritative migration source.

---

## 7.5 Persist the full product image gallery

The script uploads images to MinIO and sets the first successful image as `primaryImageUrl`.

Future direction:

```text
- Persist all product images through catalog image endpoints.
- Preserve image order.
- Preserve alt text.
- Keep primary image explicit.
```

This would make the catalog richer and closer to the original product source.

---

## 7.6 Avoid unbounded recursive token retry

The migration client retries on `401 Unauthorized` by clearing the token and trying again. This is useful, but the retry should be bounded.

Future direction:

```text
Retry once after refreshing the token.
If it still fails, return the authentication error.
```

Avoid recursive retry without an explicit limit.

---

## 7.7 Avoid hard-coded default credentials

Defaults are convenient for local development, but migration scripts should prefer environment variables or required flags for sensitive values.

Examples:

```text
- Keycloak username
- Keycloak password
- MinIO username
- MinIO password
```

Future direction:

```text
Local defaults can remain documented in docker-compose, but production-like runs should require explicit credentials.
```

---

## 7.8 Write a migration report

At the end of a migration run, write a JSON report with:

```text
- created products
- skipped products
- failed products
- created variants
- failed variants
- uploaded images
- failed images
- compensation failures
```

This is especially useful for real migration work.

---

# 8. Testing Improvements

## 8.1 Add gateway saga integration tests

The variant creation saga should be tested across success and failure paths:

```text
- Catalog variant created and inventory registration succeeds.
- Catalog variant created but inventory registration fails, compensation succeeds.
- Catalog variant created but inventory registration fails, compensation also fails.
- Catalog validation fails and inventory is not called.
```

This protects the most important cross-service workflow.

---

## 8.2 Add repository concurrency tests

Inventory repository tests should simulate concurrent reservations/adjustments.

The goal is to prove that stock cannot go negative under concurrent access.

Test cases:

```text
- Two concurrent reservations competing for the same available stock.
- Reservation while adjustment is in progress.
- Release reservation while stock is being adjusted.
```

These tests become especially important after introducing atomic updates or row locks.

---

## 8.3 Add handler tests for error mapping

Handlers should be tested for error translation:

```text
repository/service not found → HTTP 404
duplicate product → HTTP 409
invalid request → HTTP 400
inventory FailedPrecondition → HTTP 422
unexpected downstream error → HTTP 502 or 500
```

This keeps API behavior stable.

---

## 8.4 Add migration dry-run tests

The seed script supports dry-run behavior. Add tests around:

```text
- reading Silver JSON files
- skipping malformed products
- detecting images
- avoiding external writes in dry-run mode
```

---

# 9. Go Idiom Improvements

## 9.1 Keep interfaces small and close to consumers

This is an important Go design principle.

Instead of large producer-owned interfaces, define small consumer-owned interfaces.

Java-style tendency:

```go
type Repository interface {
    CreateProduct(...)
    UpdateProduct(...)
    DeleteProduct(...)
    FindProduct(...)
    AddVariant(...)
    DeleteVariant(...)
    AddImage(...)
    ListImages(...)
    SyncProducts(...)
}
```

Go-style direction:

```go
type ProductReader interface {
    FindProduct(...)
    ListProducts(...)
}

type ProductWriter interface {
    CreateProduct(...)
    UpdateProduct(...)
}

type VariantWriter interface {
    AddVariant(...)
    DeleteVariant(...)
}

type SyncReader interface {
    ListUpdatedProducts(...)
}
```

A single concrete repository can satisfy multiple interfaces implicitly.

This explains wiring such as:

```go
catalogRepo := repository.NewPostgresCatalogRepository(db)

service.NewCatalogService(
    catalogRepo, // ProductRepository
    catalogRepo, // VariantRepository
    catalogRepo, // ImageRepository
    catalogRepo, // SyncRepository
)
```

The same concrete object is being passed under different roles.

---

## 9.2 Document intentional interface segregation

The repeated repository argument in service constructors may look strange to readers coming from Java.

Document it explicitly:

```text
The same concrete repository satisfies multiple small interfaces.
Each constructor argument represents a different capability.
The runtime package is the correct place to know that these capabilities share the same implementation.
```

This makes the design easier to understand.

---

## 9.3 Avoid over-abstracting too early

Some duplicated code, such as transaction helpers with different return types, can be acceptable in Go.

Future direction:

```text
Only introduce a generic transaction helper if more transactional return types appear.
Prefer clarity over premature abstraction.
```

Go code should remain simple and explicit.

---

## 9.4 Keep using `context.Context` from entrypoint to downstream calls

The project already passes context through HTTP/gRPC/service/repository boundaries. Preserve that pattern.

Guideline:

```text
HTTP request context
  → handler
  → service
  → repository
  → database query

Gateway request context
  → catalog HTTP request
  → inventory gRPC request
```

This enables cancellation, timeouts, and request-scoped metadata.

---

## 9.5 Avoid using `context.Value` for business dependencies

`context.Context` should not become a dependency container.

Acceptable uses:

```text
- request ID
- trace ID
- authenticated principal
- deadline/cancellation
```

Avoid using it for:

```text
- repositories
- services
- configuration
- domain inputs
```

---

# 10. API and Contract Improvements

## 10.1 Clarify public, admin, and sync APIs

The system has different kinds of API consumers:

```text
- public storefront
- admin UI
- migration/seed tooling
- downstream sync consumers
```

Future route design should make this explicit.

Possible structure:

```text
/public/products
/admin/products
/admin/inventory
/sync/product-projections
/sync/variant-inventory
```

or a similar convention.

---

## 10.2 Version external-facing APIs

If this project continues growing, add versioning for gateway APIs:

```text
/api/v1/products
/api/v1/admin/products
/api/v1/sync/product-projections
```

This is especially useful if the React apps and migration scripts depend on stable contracts.

---

## 10.3 Define ownership of DTOs

Clarify which DTOs belong to:

```text
- service domain
- HTTP request/response
- gRPC protobuf
- migration/Silver import
- frontend UI
```

Avoid leaking one layer’s DTOs into unrelated layers.

---

# 11. Security Improvements

## 11.1 Enforce authentication in production

Permissive auth should not be possible in production.

Future direction:

```text
if ENV=production and KEYCLOAK_URL is empty:
    fail startup
```

---

## 11.2 Add service-to-service security later

Currently, gateway protects browser-facing APIs. If internal services become exposed across a broader network, add service-to-service authentication.

Possible options:

```text
- mTLS
- internal JWT
- network policies
- sidecar/service mesh later if needed
```

Keep it simple until the project needs it.

---

## 11.3 Avoid logging sensitive values

Review logs to ensure they do not include:

```text
- tokens
- passwords
- authorization headers
- sensitive user data
```

---

# 12. Suggested Priority

## P0 — Interview-critical documentation

These items help explain the project clearly:

```text
1. Document service boundaries.
2. Document inventory vs gateway SSE ownership.
3. Document transaction ownership: service owns unit of work, repository owns SQL.
4. Document gateway saga flow and compensation.
5. Document interface segregation and why the same repository may be passed multiple times.
6. Rename or document legacy SQLite naming.
```

## P1 — Backend correctness

These items improve real system correctness:

```text
1. Make inventory mutations concurrency-safe.
2. Add gRPC input validation.
3. Add explicit RegisterSKU RPC.
4. Add service-level transaction tests.
5. Add gateway saga integration tests.
```

## P2 — Production hardening

These items improve operational behavior:

```text
1. Add downstream timeouts/deadlines.
2. Add graceful shutdown.
3. Add gRPC interceptors.
4. Persist failed compensations.
5. Add request/correlation IDs.
6. Add migration reports.
```

## P3 — Cleanup and polish

These items improve maintainability:

```text
1. Hide protobuf stubs behind client façade methods.
2. Move saga orchestration out of handlers if it grows.
3. Avoid handler-to-handler dependencies.
4. Normalize route structure.
5. Improve catalog response DTO consistency.
6. Remove unused helper parameters.
7. Use structured logging consistently.
```

---

# Final Positioning

The most important way to frame go-folio is:

```text
go-folio is not production-perfect, but it is intentionally production-shaped.
```

It explores meaningful backend engineering patterns:

```text
- explicit dependency wiring
- service boundaries
- REST and gRPC integration
- transaction ownership
- repository abstraction
- gateway orchestration
- SSE real-time updates
- OIDC authentication
- migration from normalized Silver data
```

The future-forward backlog shows the next maturity steps:

```text
- concurrency correctness
- stronger validation
- explicit contracts
- reliable saga recovery
- graceful shutdown
- better observability
- clearer API boundaries
```

This makes the project valuable both as a learning platform and as an architectural portfolio piece.
