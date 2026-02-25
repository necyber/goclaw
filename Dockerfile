# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o goclaw ./cmd/goclaw/main.go

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN adduser -D -u 1000 goclaw

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/goclaw .

# Copy config example (can be overridden)
COPY --from=builder /build/config/config.example.yaml ./config.yaml

# Create data directory for persistent storage
RUN mkdir -p /app/data && chown -R goclaw:goclaw /app/data

# Change ownership to non-root user
RUN chown -R goclaw:goclaw /app

# Switch to non-root user
USER goclaw

# Declare volume for persistent data
VOLUME ["/app/data"]

# Expose ports
EXPOSE 8080 9090 9091

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./goclaw"]
CMD ["-config", "config.yaml"]
