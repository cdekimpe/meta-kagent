# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /kmeta-agent-server \
    ./cmd/mcp-server

# Runtime stage
FROM alpine:3.19

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 kagent && \
    adduser -u 1000 -G kagent -s /bin/sh -D kagent

WORKDIR /app

# Copy binary from builder
COPY --from=builder /kmeta-agent-server /kmeta-agent-server

# Use non-root user
USER kagent

# Set default environment
ENV KAGENT_NAMESPACE=kagent
ENV LOG_LEVEL=info

# The binary will be executed via stdio transport
ENTRYPOINT ["/kmeta-agent-server"]
