# Project Wisdom: High-Performance Cognitive SRE Engine
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache build-base sqlite

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o wisdom-api cmd/wisdom-api/main.go

# Final stage
FROM alpine:latest
RUN apk add --no-cache sqlite-libs

WORKDIR /root/

# Copy binary and assets
COPY --from=builder /app/wisdom-api .
COPY --from=builder /app/pkg/cortex/schema.sql ./pkg/cortex/schema.sql

# Expose API port
EXPOSE 8080

# Environment variables
ENV WISDOM_PORT=8080
ENV WISDOM_DB_PATH=wisdom.db

CMD ["./wisdom-api"]
