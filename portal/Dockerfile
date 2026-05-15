# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM node:20-slim AS builder

WORKDIR /app

# Build arguments for environment variables
ARG VITE_ENGINE_URL
ARG VITE_AGENT_URL
ARG VITE_ENGINE_WS_URL
ARG VITE_AGENT_WS_URL
ARG VITE_REDIRECT_URI

# Environment variables for build time (Vite bakes these into the bundle)
ENV VITE_ENGINE_URL=$VITE_ENGINE_URL
ENV VITE_AGENT_URL=$VITE_AGENT_URL
ENV VITE_ENGINE_WS_URL=$VITE_ENGINE_WS_URL
ENV VITE_AGENT_WS_URL=$VITE_AGENT_WS_URL
ENV VITE_REDIRECT_URI=$VITE_REDIRECT_URI

# Install dependencies first (layer cache)
COPY package.json package-lock.json ./
RUN npm ci --prefer-offline

# Copy source and build
# VITE_ env vars are baked at build time via Cloud Build substitutions
COPY . .
RUN npm run build

# ── Stage 2: Serve ────────────────────────────────────────────────────────────
FROM nginx:1.27-alpine AS runtime

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Custom nginx config — SPA routing (all 404s → index.html)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
