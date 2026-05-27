FROM golang:1.24-alpine AS builder
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY server/gen/ server/gen/
COPY server/gateway-service/ server/gateway-service/

RUN CGO_ENABLED=0 go build -C server/gateway-service -o /gateway ./cmd/main.go

FROM scratch
COPY --from=builder /gateway /gateway
ENTRYPOINT ["/gateway"]
