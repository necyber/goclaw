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
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 goclaw

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/goclaw .

# Copy config example (can be overridden)
COPY --from=builder /build/config/config.example.yaml ./config.yaml

# Change ownership to non-root user
RUN chown -R goclaw:goclaw /app

# Switch to non-root user
USER goclaw

# Expose ports
EXPOSE 8080 9090 9091

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./goclaw -help || exit 1

# Run the application
ENTRYPOINT ["./goclaw"]
CMD ["-config", "config.yaml"]
