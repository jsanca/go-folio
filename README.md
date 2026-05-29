# go-folio

go-folio is a production-ready product catalog service written in Go, designed to manage and serve structured product and category data for a leather goods store.

Built with a focus on clean architecture, it separates concerns across distinct layers — configuration, persistence, business logic, and transport — keeping each boundary explicit and testable. The service exposes a REST API, connects to a relational database, and includes structured logging and observability from the start.

Designed as a portfolio project, go-folio demonstrates idiomatic Go patterns: dependency injection without frameworks, layered runtime composition, database-backed seeding, and a minimal but production-conscious main.

## High-level architecture

```
React SPA (client/)
    │
    ▼ REST/JSON
API Gateway :8090  ──── Keycloak :8180  (JWT validation)
    ├──→ catalog-service  :8080  (REST)
    └──→ inventory-service :9090  (gRPC)
```

## High-level reverse proxy/gateway
```
Internet
    │
    ▼
Nginx / Traefik (reverse proxy, SSL termination)
    │
    ▼
Gateway :8090
    ├──→ Catalog :8080    (internal Docker network, not open)
    └──→ Inventory :9090  (internal Docker network, not open)
```

## Authentication workflow
```
React SPA (public client "gateway")
    │ ask for a token with usuario/password
    ▼
Keycloak realm folio
    │ emits JWT
    ▼
React SPA save the token
    │ Authorization: Bearer <token>
    ▼
Gateway validates the JWT
```

## Quick start

```bash
docker compose up --build
```

Services:
- Gateway:   http://localhost:8090
- Catalog:   http://localhost:8080
- Keycloak:  http://localhost:8180

## Keycloak setup

After `docker compose up`, Keycloak starts at **http://localhost:8180**.
The admin console is at http://localhost:8180/admin (credentials: `admin` / `admin`).

### 1. Create the `folio` realm

1. Log in to the admin console.
2. Open the realm dropdown (top-left, shows **Keycloak**) → **Create realm**.
3. Set **Realm name** to `folio` and click **Create**.

### 2. Create realm roles

Inside the `folio` realm:

1. Go to **Realm roles** → **Create role**.
2. Create role `admin`, save.
3. Repeat for role `customer`.

### 3. Create a client for the SPA / Postman

1. Go to **Clients** → **Create client**.
2. Set **Client ID** to `folio-public` and click **Next**.
3. Disable **Client authentication** (public client) and enable **Standard flow**.
4. Set **Valid redirect URIs** to `http://localhost:5173/*` and `http://localhost:8090/*`.
5. Save.

### 4. Create a test user

1. Go to **Users** → **Create new user**.
2. Fill in **Username** (e.g. `alice`) and save.
3. Go to the **Credentials** tab → **Set password** (disable Temporary).
4. Go to **Role Mappings** → **Assign role** → filter by realm → assign `admin` or `customer`.

### 5. Get a token (Postman / curl)

```bash
curl -s -X POST \
  "http://localhost:8180/realms/folio/protocol/openid-connect/token" \
  -d "client_id=folio-public" \
  -d "grant_type=password" \
  -d "username=alice" \
  -d "password=<password>" \
  | jq .access_token
```

Use the token as `Authorization: Bearer <token>` when calling `/admin/*` endpoints.

## Auth environment variables

| Variable | Default | Description |
|---|---|---|
| `KEYCLOAK_URL` | *(empty)* | Keycloak base URL. **Leave unset for permissive dev mode.** |
| `KEYCLOAK_REALM` | `folio` | Realm name |

When `KEYCLOAK_URL` is empty the gateway starts in **permissive mode** — all
requests pass through without authentication, including admin routes. This lets
you develop locally without running Keycloak.

## Route access control

| Route | Auth required |
|---|---|
| `GET /products` | Public |
| `GET /products/{sku}` | Public |
| `GET /admin/products` | `admin` role |
| `POST /admin/products` | `admin` role |
| `PATCH /admin/products/{sku}` | `admin` role |
| `PUT /admin/inventory/{sku}` | `admin` role |
