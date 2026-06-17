# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/clinical-record ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/healthcheck ./cmd/healthcheck

# Final stage
FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache ca-certificates wget && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.24/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
COPY --from=builder /bin/clinical-record /usr/local/bin/
COPY --from=builder /bin/healthcheck /usr/local/bin/

EXPOSE 50051 8080 9090

# Health check usando grpc_health_probe
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD grpc_health_probe -addr=localhost:50051 || exit 1

# Comando principal usando tini para manejo de señales
ENTRYPOINT ["/usr/local/bin/clinical-record"]
