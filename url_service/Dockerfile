# Multi-stage build for minimal image size

# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o url-shortener .

# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/url-shortener .

# Copy config files
COPY --from=builder /app/configs ./configs

# Copy migration files
COPY --from=builder /app/migration ./migration

# Expose gRPC port
EXPOSE 8080

# Run the application
CMD ["./url-shortener"]
