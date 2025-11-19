# Build stage
FROM golang:1.21-alpine AS builder

# Install git (required for some go modules)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o smite-bot ./cmd/lan_smite_teams_bot

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/smite-bot .

# Copy example config
COPY config.yaml.example config.yaml.example

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose any ports if needed (not required for Discord bots)
# EXPOSE 8080

# Set environment variables
ENV CONFIG_PATH=/app/config.yaml

# Health check (optional)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD pgrep smite-bot || exit 1

# Run the application
CMD ["./smite-bot"]