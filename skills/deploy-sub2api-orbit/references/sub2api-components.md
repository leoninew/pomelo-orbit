# Sub2API Orbit Component Contract

Use these declarations with `pomelo-orbit-mcp`. This is the source of truth for the Orbit Version, Service runtime configuration, and Route target.

## Topology

| Application | Version components | Service | Public endpoint |
| --- | --- | --- | --- |
| `sub2api` (`standard`) | `postgres`, `redis`, `app` | `default` | Custom HTTP Route -> `app` / `http:8080` |

All components use `pull_policy=always` and `restart_policy=unless-stopped`. Use explicit Service-workspace-relative `directory` mounts; every source must begin with `./` and resolves under the Service workspace.

## Runtime Configuration

Set these K/V pairs with `orbit_update_service_env`. Values are plaintext in Orbit's current Service runtime configuration, so do not use this topology when that storage model is unacceptable.

| Key | Required by | Requirement |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | `postgres`, `app` | Stable strong password |
| `REDIS_PASSWORD` | `redis`, `app` | Stable strong password |
| `ADMIN_PASSWORD` | `app` | Initial administrator password |
| `JWT_SECRET` | `app` | Stable 32-byte-or-longer random secret |
| `TOTP_ENCRYPTION_KEY` | `app` | Stable 64-character hexadecimal key |

Use the exact Orbit placeholders `${KEY}` in component environment values. Orbit does not read a Compose `.env` or `env_file`, and `${KEY:?message}` is not a supported Service runtime-config placeholder.

## Components

### postgres

| Field | Value |
| --- | --- |
| Image | `postgres:18-alpine` |
| Environment | `PGDATA=/var/lib/postgresql/data`, `POSTGRES_USER=sub2api`, `POSTGRES_PASSWORD=${POSTGRES_PASSWORD}`, `POSTGRES_DB=sub2api`, `TZ=Asia/Shanghai` |
| Mount | `directory` source `./postgres` -> `/var/lib/postgresql/data` |
| Health check | `CMD-SHELL` `pg_isready -U sub2api -d sub2api`; interval `10s`, timeout `5s`, retries `5`, start period `10s` |
| Endpoint | None |

### redis

| Field | Value |
| --- | --- |
| Image | `redis:8-alpine` |
| Command | `sh -c 'exec redis-server --appendonly yes --requirepass "$${REDIS_PASSWORD}"'` |
| Environment | `REDIS_PASSWORD=${REDIS_PASSWORD}`, `REDISCLI_AUTH=${REDIS_PASSWORD}`, `TZ=Asia/Shanghai` |
| Mount | `directory` source `./redis` -> `/data` |
| Health check | `CMD-SHELL` `redis-cli ping`; interval `10s`, timeout `5s`, retries `5`, start period `5s` |
| Endpoint | None |

### app

| Field | Value |
| --- | --- |
| Image | `weishaw/sub2api:latest` |
| Environment | `AUTO_SETUP=true`, `SERVER_HOST=0.0.0.0`, `SERVER_PORT=8080`, `DATABASE_HOST=postgres`, `DATABASE_PORT=5432`, `DATABASE_USER=sub2api`, `DATABASE_PASSWORD=${POSTGRES_PASSWORD}`, `DATABASE_DBNAME=sub2api`, `DATABASE_SSLMODE=disable`, `REDIS_HOST=redis`, `REDIS_PORT=6379`, `REDIS_PASSWORD=${REDIS_PASSWORD}`, `ADMIN_EMAIL=admin@sub2api.local`, `ADMIN_PASSWORD=${ADMIN_PASSWORD}`, `JWT_SECRET=${JWT_SECRET}`, `JWT_EXPIRE_HOUR=24`, `TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}`, `TZ=Asia/Shanghai` |
| Mount | `directory` source `./app` -> `/app/data` |
| Dependencies | `postgres: service_healthy`; `redis: service_healthy` |
| Health check | `CMD-SHELL` `wget -q -T 5 -O /dev/null http://localhost:8080/health`; interval `30s`, timeout `10s`, retries `3`, start period `30s` |
| Endpoint | protocol `http`, container port `8080`, mode `internal`; do not set bind address or listen port |

`AUTO_SETUP=true` applies migrations and creates the initial administrator when Sub2API detects an uninitialized data directory. Do not reset the `app`, `postgres`, or `redis` mount to re-run setup without explicit authorization.

## Managed Route

Create one HTTP Route only after the Sub2API Service is running. It must use `service_id`, `component_name=app`, `endpoint_protocol=http`, and `endpoint_container_port=8080`. Do not map a host port and do not use `target_url`.
