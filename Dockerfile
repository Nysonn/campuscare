# Migration stage - for running database migrations
FROM golang:1.25-alpine AS migrate
WORKDIR /app

# Install golang-migrate and netcat for health checks
RUN apk add --no-cache netcat-openbsd && \
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations using shell so $DATABASE_URL is expanded from env_file at runtime
CMD ["sh", "-c", "migrate -path /app/migrations -database \"$DATABASE_URL\" up"]

# Dev stage - for local development with hot reload
FROM golang:1.25-alpine AS dev
WORKDIR /app

# Install air for hot reload
RUN go install github.com/cosmtrek/air@v1.49.0

# Copy go mod files
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Run air for hot reload
CMD ["air", "-c", ".air.toml"]

# Builder stage - for building the binary
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org
RUN go mod download

# Copy source code
COPY . .

# Disable cgo for static binary
ENV CGO_ENABLED=0

# Build the binary with optimizations
RUN go build -ldflags="-s -w" -o /app/cmd-server ./cmd/server

# Production stage - minimal image
FROM alpine:latest AS production
WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/cmd-server /app/cmd-server

# Create non-root user
RUN addgroup -S app && adduser -S app -G app

# Change ownership of app directory
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Set Gin to release mode
ENV GIN_MODE=release

# Run the application
ENTRYPOINT ["/app/cmd-server"]