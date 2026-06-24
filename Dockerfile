# Pomelo Orbit - Multi-stage Dockerfile

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package.json frontend/yarn.lock ./

# Install dependencies
RUN yarn install --frozen-lockfile

# Copy frontend source
COPY frontend/ ./

# Build frontend
RUN yarn build

# Stage 2: Build backend-go
FROM golang:1.25-bookworm AS backend-builder

WORKDIR /app/backend-go

# Copy backend-go module files
COPY backend-go/go.mod backend-go/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend-go source
COPY backend-go/cmd ./cmd
COPY backend-go/internal ./internal

# Build backend-go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend-go ./cmd/backend-go

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

# Copy backend-go binary and configuration
COPY --from=backend-builder /out/backend-go /usr/local/bin/backend-go
COPY backend-go/config.defaults.yaml ./config.defaults.yaml

# Copy frontend build from stage 1
COPY --from=frontend-builder /app/frontend/dist ./static

# Create runtime directories
RUN mkdir -p /app/data/db /app/logs

# Environment variables
ENV POMELO_ORBIT_SERVER__HOST=0.0.0.0
ENV POMELO_ORBIT_SERVER__PORT=80

# Expose port
EXPOSE 80

# Run the application. Override CMD with "worker" to run the background worker.
ENTRYPOINT ["backend-go"]
CMD ["serve"]
