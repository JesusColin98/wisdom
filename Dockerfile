# Project Wisdom: Unified Cognitive SRE Engine & Neural Atlas

# Stage 1: Build the Frontend (Neural Atlas)
FROM node:20-alpine AS frontend-builder
WORKDIR /app/portal
COPY portal/package*.json ./
RUN npm install
COPY portal/ .
RUN npm run build

# Stage 2: Build the Backend (Wisdom Engine)
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app/wisdom
COPY wisdom/go.mod wisdom/go.sum ./
RUN go mod download
COPY wisdom/ .
RUN go build -o wisdom_engine cmd/wisdom-api/main.go

# Stage 3: Final Production Image
FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=backend-builder /app/wisdom/wisdom_engine .
# Copy schemas
COPY --from=backend-builder /app/wisdom/pkg/cortex/*.sql ./pkg/cortex/

# Copy frontend assets to 'public' directory served by Go
COPY --from=frontend-builder /app/portal/dist ./public

# Expose port
EXPOSE 8080

# Environment variables
ENV PORT=8080

CMD ["./wisdom_engine"]
