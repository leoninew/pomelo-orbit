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

# Stage 2: Python backend with uv
FROM python:3.12-slim AS backend-builder

WORKDIR /app

# Install uv
RUN pip install --no-cache-dir uv

# Copy backend files
COPY backend/pyproject.toml backend/uv.lock* ./

# Install dependencies
RUN uv sync

# Copy source code (needed for dependency resolution)
COPY backend/src ./src

# Stage 3: Final image
FROM python:3.12-slim

WORKDIR /app

# Install system dependencies and docker CLI
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    docker-cli \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /usr/local/lib/docker/cli-plugins

RUN curl -SL https://github.com/docker/compose/releases/download/v5.1.0/docker-compose-linux-x86_64 -o /usr/local/lib/docker/cli-plugins/docker-compose \
    && chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Copy Python dependencies from builder
COPY --from=backend-builder /app/.venv /app/.venv

# Copy backend configuration files
COPY backend/pyproject.toml backend/config.defaults.yaml ./

# Copy backend source
COPY backend/src/ ./src/

# Copy database migrations
COPY backend/migrations/ ./migrations/

# Copy frontend build from stage 1
COPY --from=frontend-builder /app/frontend/dist ./static

# Set Python path
ENV PYTHONPATH=/app/src

# Create data directories
RUN mkdir -p /app/data/db /app/logs

# Environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PATH="/app/.venv/bin:$PATH"

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:80/api/health || exit 1

# Run the application
CMD ["python", "-m", "pomelo_orbit.main", "--host", "0.0.0.0", "--port", "80"]
