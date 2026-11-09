# Pomelo Orbit - Multi-stage Dockerfile

# Stage 1: Build web
FROM node:22.23.4-alpine AS web-builder

WORKDIR /app/web
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN npm install --global corepack@0.34.6

# Copy web package files
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml web/.npmrc ./

# Install dependencies
RUN --mount=type=cache,id=pnpm10-store,target=/pnpm/store,sharing=locked \
    corepack pnpm install --store-dir=/pnpm/store --frozen-lockfile

# Copy web source
COPY web/ ./

# Build web
RUN corepack pnpm build

# Stage 2: Build backend
FROM golang:1.25-bookworm AS backend-builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source
COPY cmd ./cmd
COPY internal ./internal
COPY sql ./sql

# Build backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pomelo-orbit ./cmd/server

# Stage 3: Final image
FROM debian:trixie-slim

WORKDIR /app

# Install system dependencies and docker CLI
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    docker.io \
    docker-cli \
    docker-compose \
    && rm -rf /var/lib/apt/lists/* \
    && docker compose version

# Copy backend binary and configuration
COPY --from=backend-builder /out/pomelo-orbit /usr/local/bin/pomelo-orbit
COPY configs ./configs

# Copy web build from stage 1
COPY --from=web-builder /app/web/dist ./static

# Create runtime directories
RUN mkdir -p /app/data/db /app/logs

# Environment variables
ENV POMELO_ORBIT_SERVER__HOST=0.0.0.0
ENV POMELO_ORBIT_SERVER__PORT=80

# Expose port
EXPOSE 80

# Run the API server and background worker in one process.
ENTRYPOINT ["pomelo-orbit"]
CMD ["serve"]
