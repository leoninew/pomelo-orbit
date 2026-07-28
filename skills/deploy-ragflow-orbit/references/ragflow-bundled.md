# RAGFlow Bundled Mapping

Use this reference for `scripts/ragflow-bundled/docker-compose.yml`. All Components belong to the same Version and the same `default` Service.

## Component Mapping

| Component | Image | Logical mount | Required configuration |
| --- | --- | --- | --- |
| `mysql` | `mysql:8.0.39` | `mysql/data` -> `/var/lib/mysql` | Preserve all six MySQL command flags; `MYSQL_DATABASE=rag_flow`; `MYSQL_ROOT_PASSWORD=${MYSQL_PASSWORD}`; `MYSQL_ROOT_HOST=%`; CMD-SHELL `mysqladmin ping`; `unless-stopped` |
| `redis` | `valkey/valkey:8` | `redis/data` -> `/data` | `sh -c exec redis-server --requirepass "$${REDIS_PASSWORD}" --maxmemory 128mb --maxmemory-policy allkeys-lru`; `REDIS_PASSWORD=${REDIS_PASSWORD}`; CMD-SHELL `redis-cli`; `unless-stopped` |
| `minio` | `pgsty/minio:RELEASE.2026-03-25T00-00-00Z` | `minio/data` -> `/data` | `server --console-address :9001 /data`; root user/password use `${MINIO_USER}` and `${MINIO_PASSWORD}`; CMD healthcheck for `/minio/health/live`; `unless-stopped` |
| `es01` | `elasticsearch:8.11.3` | `es01/data` -> `/usr/share/elasticsearch/data` | Preserve security and watermark settings; `${ELASTIC_PASSWORD}`; CMD-SHELL curl healthcheck; memory limit `8g`; tmpfs `/tmp` with `536870912` bytes and mode `1777`; memlock `-1/-1`; `unless-stopped` |
| `ragflow-cpu` | `infiniflow/ragflow:v0.26.4` | `ragflow-cpu/logs` -> `/ragflow/logs` | Command `--enable-adminserver --init-model-provider-tables`; preserve database/cache/object-store/ES environment; depend on all four services with `service_healthy`; CMD curl healthcheck on `/`; `unless-stopped` |

Use `${MYSQL_PASSWORD}`, `${REDIS_PASSWORD}`, `${MINIO_USER}`, `${MINIO_PASSWORD}`, and `${ELASTIC_PASSWORD}` only as complete environment values. Keep `$${...}` in shell commands and healthchecks so Docker Compose passes expansion to the container shell.

## Connectivity

Within the same Version, set RAGFlow hosts to Compose service names:

| Environment key | Value |
| --- | --- |
| `MYSQL_HOST` | `mysql` |
| `REDIS_HOST` | `redis` |
| `MINIO_HOST` | `minio` |
| `ES_HOST` | `es01` |

Set `MYSQL_PORT=3306`, `MYSQL_DBNAME=rag_flow`, `MYSQL_USER=root`, `ES_USER=elastic`, and `MYSQL_MAX_PACKET=1073741824`.

## Rendered Compose Differences

Expect Orbit to add `container_name`, Version labels, a `ragflow-default_default` network, and managed absolute bind sources below `data/deployment/ragflow/default`. It retains the external `traefik` network and aliases. The local Expose renders `127.0.0.1:9380:80`; it replaces the source Compose's configurable `${RAGFLOW_HTTP_PORT:-9380}` binding.

Elasticsearch's `8g` memory limit may render as `8589934592`. Compose may render `required: true` alongside each `service_healthy` dependency. These are equivalent to the intended deployment behavior.

## Verification Checklist

- All five containers are `running` and `healthy`.
- `ragflow-cpu` exposes `127.0.0.1:9380 -> 80/tcp` only.
- No database, cache, object-store, or Elasticsearch port is published to the host.
- `runtime_http_probe` reports `reachable` for `ragflow-cpu:80/`.
- `verify_deployment` concludes `consistent` with no restart-count increase during the stability window.
