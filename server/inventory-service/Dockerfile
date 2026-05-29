FROM golang:1.26-alpine AS builder
WORKDIR /workspace
COPY go.work go.work.sum ./
COPY server/gen/ server/gen/
COPY server/inventory-service/ server/inventory-service/
RUN GOWORK=off CGO_ENABLED=0 go build -C server/inventory-service -o /inventory ./cmd/main.go

FROM scratch
COPY --from=builder /inventory /inventory
ENTRYPOINT ["/inventory"]
