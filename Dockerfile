# Build stage
FROM golang:1.23.2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/bot cmd/bot/main.go
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/worker cmd/worker/main.go

# Final stage
FROM alpine:latest

# Set default environment variables
ENV RUN_TYPE=bot \
    WORKER_TYPE=ai \
    WORKER_SUBTYPE=friend \
    WORKERS_COUNT=1

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /app/config/credentials /app/logs/bot_logs /app/logs/worker_logs

# Copy binaries from builder
COPY --from=builder /app/bin/bot /app/bin/bot
COPY --from=builder /app/bin/worker /app/bin/worker

# Set working directory
WORKDIR /app

# Copy entrypoint script
COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

ENTRYPOINT ["/app/docker-entrypoint.sh"]
