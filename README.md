# go-folio

go-folio is a portfolio e-commerce backend for a leather goods store. It demonstrates a multi-service Go architecture with a React admin SPA and a Next.js customer storefront — all wired together through a single API gateway.

## Architecture

```
                    ┌─────────────────┐   ┌─────────────────┐
                    │  store (Next.js) │   │  admin (Vite)   │
                    │   :3000          │   │   :3001          │
                    └────────┬────────┘   └────────┬────────┘
                             │ REST/JSON            │ REST/JSON
                             ▼                      ▼
                    ┌──────────────────────────────────────────┐
                    │          gateway-service :8090           │
                    │  (auth middleware, aggregation, SSE)     │
                    └────────────┬─────────────┬──────────────┘
                                 │             │
                    REST :8080   │             │  gRPC :9090
                                 ▼             ▼
                    ┌─────────────────┐ ┌──────────────────┐
                    │ catalog-service │ │inventory-service │
                    │ (products,      │ │(stock, reserva-  │
                    │  variants)      │ │ tions)           │
                    └────────┬────────┘ └────────┬─────────┘
                             │                   │
                             └─────────┬─────────┘
                                       ▼
                              ┌─────────────────┐
                              │   PostgreSQL     │
                              │ folio_catalog    │
                              │ folio_inventory  │
                              └─────────────────┘

                    ┌─────────────────┐   ┌─────────────────┐
                    │   Keycloak      │   │     MinIO        │
                    │   :8080         │   │  :9000 / :9001   │
                    │ (JWT issuance)  │   │ (image storage)  │
                    └─────────────────┘   └─────────────────┘
```

## Docker topology

```
docker compose up --build
  ├── postgres           (private network)
  ├── keycloak    :8080  (private network, realm auto-imported)
  ├── catalog-service    (private network — reachable only via gateway)
  ├── inventory-service  (private network — reachable only via gateway)
  ├── minio      :9000 / :9001  (private network + host ports)
  ├── gateway-service :8090     (public + private)
  ├── store       :3000         (public)
  └── admin       :3001         (public)
```

## Quick start

```bash
docker compose up --build
```

The stack is fully self-contained. No manual setup required.

| Service | URL |
|---|---|
| Customer store | http://localhost:3000 |
| Admin UI | http://localhost:3001 |
| Gateway API | http://localhost:8090 |
| Keycloak admin console | http://localhost:8080/admin |
| MinIO console | http://localhost:9001 |

## Authentication

Keycloak starts at **http://localhost:8080**. The `folio` realm is automatically imported from `scripts/keycloak/folio-realm.json` — no manual realm or role configuration needed.

**Test users (created by the realm import):**

| User | Password | Role |
|---|---|---|
| `admin@folio.dev` | `admin123` | `admin` |
| `customer@folio.dev` | `customer123` | `customer` |

**Public client:** `gateway` (redirect URIs cover localhost:3000, 3001, 8090).

### Get a token

```bash
curl -s -X POST \
  "http://localhost:8080/realms/folio/protocol/openid-connect/token" \
  -d "client_id=gateway" \
  -d "grant_type=password" \
  -d "username=admin@folio.dev" \
  -d "password=admin123" \
  | jq -r .access_token
```

Use the token as `Authorization: Bearer <token>` on admin routes.

### Permissive mode

When `KEYCLOAK_URL` is unset the gateway runs in **permissive mode** — all requests pass without authentication. Useful for local development outside Docker.

## API routes

| Method | Route | Auth |
|---|---|---|
| `GET` | `/products` | Public |
| `GET` | `/products/{sku}` | Public |
| `GET` | `/admin/products` | `admin` role |
| `POST` | `/admin/products` | `admin` role |
| `PATCH` | `/admin/products/{id}` | `admin` role |
| `DELETE` | `/admin/products/{id}` | `admin` role |
| `POST` | `/admin/products/{id}/variants` | `admin` role |
| `GET` | `/admin/inventory` | `admin` role |
| `GET` | `/admin/inventory/{sku}` | `admin` role |
| `PUT` | `/admin/inventory/{sku}` | `admin` role |
| `GET` | `/admin/events` | `admin` role (SSE stream) |

## Tech stack

| Layer | Technology |
|---|---|
| Backend services | Go 1.23, chi router, pgx/v5 |
| Inter-service RPC | gRPC (inventory), REST/JSON (catalog) |
| Database | PostgreSQL 16 |
| Auth | Keycloak 24, OIDC / RS256 JWT |
| Image storage | MinIO (S3-compatible) |
| Customer storefront | Next.js 14 (App Router), Tailwind CSS |
| Admin UI | Vite + React 18, Ant Design |
| Observability | slog structured logging, Prometheus metrics |
| Containerisation | Docker Compose |

## Seed script

`scripts/seed-catalog` is a standalone Go CLI (outside the workspace) that seeds products from a Silver export directory:

```bash
cd scripts/seed-catalog
GOWORK=off go run . \
  --silver-dir ./silver \
  --images-dir ./images \
  --dry-run
```

Full flags: `--gateway`, `--keycloak`, `--realm`, `--user`, `--password`, `--minio-url`, `--minio-user`, `--minio-pass`, `--dry-run`.
