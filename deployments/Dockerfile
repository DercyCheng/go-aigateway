FROM golang:1.21-alpine AS builder

# Install build dependencies and security tools
RUN apk --no-cache add git ca-certificates tzdata

# Create non-root user for building
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY ../go.mod ../go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY .. .

# Build the application with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o main .

# Final stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy timezone data and certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/main /app/main

# Copy environment template
COPY --from=builder /app/.env.example /app/.env.example

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Health check (distroless doesn't have wget, so we'll rely on k8s probes)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD ["/app/main", "--health-check"] || exit 1

# Run the application
ENTRYPOINT ["/app/main"]
