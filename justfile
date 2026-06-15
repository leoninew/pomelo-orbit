set shell := ["bash", "-cu"]

install:
    cd frontend && yarn install
    cd backend-go && go mod download
    cd backend-go && go mod tidy

dev:
    uv run python scripts/dev.py

dev-backend:
    cd backend-go && POMELO_ORBIT_BACKEND__SERVER__HOST=127.0.0.1 POMELO_ORBIT_BACKEND__SERVER__PORT=9001 go run ./cmd/backend-go serve

dev-worker:
    cd backend-go && go run ./cmd/backend-go worker

dev-frontend:
    cd frontend && yarn dev

check:
    cd frontend && yarn lint:fix
    cd frontend && yarn format
    cd frontend && yarn typecheck
    cd backend-go && go fmt ./...
    cd backend-go && go vet ./...
    cd backend-go && go test ./...

test:
    cd frontend && yarn test
    cd backend-go && go test ./...

build tag="" cn="":
    #!/usr/bin/env bash
    set -euo pipefail
    TAG="{{tag}}"
    if [[ -z "$TAG" ]]; then TAG="latest"; fi
    DOCKERFILE="Dockerfile"
    if [[ "{{cn}}" == "1" ]]; then DOCKERFILE="Dockerfile.cn"; fi
    docker build -f "$DOCKERFILE" -t "pomelo-orbit:$TAG" .
    echo "image built: pomelo-orbit:$TAG"

clean:
    find . -type d -name "__pycache__" -not -path "./.git/*" -exec rm -rf {} +
    find . -type d -name ".mypy_cache" -not -path "./.git/*" -exec rm -rf {} +
    find . -type d -name ".ruff_cache" -not -path "./.git/*" -exec rm -rf {} +
    find . -type d -name ".pytest_cache" -not -path "./.git/*" -exec rm -rf {} +
    find . -type d -name "htmlcov" -not -path "./.git/*" -exec rm -rf {} +
    find . -type f -name "*.pyc" -not -path "./.git/*" -exec rm -f {} +
    echo "clean complete"
