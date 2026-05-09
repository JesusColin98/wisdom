# Project Wisdom: Unified Cognitive SRE Engine & Neural Atlas

# Stage 1: Build the Frontend (Neural Atlas)
FROM node:20-alpine AS frontend-builder
WORKDIR /app/portal
COPY portal/package*.json ./
RUN npm install
COPY portal/ .
RUN npm run build

# Stage 2: Build the Backend (Wisdom Engine)
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app/wisdom
RUN apk add --no-cache build-base sqlite
COPY wisdom/go.mod wisdom/go.sum ./
RUN go mod download
COPY wisdom/ .
RUN go build -o wisdom_engine cmd/wisdom/main.go

# Stage 3: Final Production Image
FROM alpine:latest
RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=backend-builder /app/wisdom/wisdom_engine .
# Copy schema
COPY --from=backend-builder /app/wisdom/pkg/cortex/schema.sql ./pkg/cortex/schema.sql
# Copy schemas for dynamic loading
COPY --from=backend-builder /app/wisdom/pkg/cerebellum/schemas ./pkg/cerebellum/schemas

# Copy frontend assets to 'public' directory served by Go
COPY --from=frontend-builder /app/portal/dist ./public

# Expose port
EXPOSE 8080

# Environment variables
ENV WISDOM_PORT=8080
ENV WISDOM_DB_PATH=wisdom.db

CMD ["./wisdom_engine"]
