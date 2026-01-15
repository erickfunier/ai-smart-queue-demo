# Stage 1: Build binaries
FROM golang:1.24.4 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source
COPY . .

# Build each service binary
RUN go build -o /bin/queue-core ./cmd/queue-core
RUN go build -o /bin/worker-runtime ./cmd/worker-runtime
RUN go build -o /bin/ai-insights-service ./cmd/ai-insights-service

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install curl for health checks
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /bin/queue-core /bin/queue-core
COPY --from=builder /bin/worker-runtime /bin/worker-runtime
COPY --from=builder /bin/ai-insights-service /bin/ai-insights-service

# Copy config files
COPY configs /app/configs

# Default command (can be overridden)
CMD ["/bin/queue-core"]