FROM golang:1.26-alpine AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /catalog ./cmd/main.go

FROM scratch
COPY --from=builder /catalog /catalog
ENTRYPOINT ["/catalog"]
