# go-folio
go-folio is a production-ready product catalog service written in Go, designed to manage and serve structured product and category data for a leather goods store.

Built with a focus on clean architecture, it separates concerns across distinct layers — configuration, persistence, business logic, and transport — keeping each boundary explicit and testable. The service exposes a REST API, connects to a relational database, and includes structured logging and observability from the start.

Designed as a portfolio project, go-folio demonstrates idiomatic Go patterns: dependency injection without frameworks, layered runtime composition, database-backed seeding, and a minimal but production-conscious main.

# high level architecture

React SPA
    │
    ▼ REST/JSON (HTTPS)
API Gateway (Go) ----> Auth/Key Clock
    ├──→ Catalog Service  (gRPC interno)
    └──→ Inventory Service (gRPC interno)
