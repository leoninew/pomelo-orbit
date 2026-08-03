# RAGFlow Split Deployment Primitives

Use this reference before making a split RAGFlow `pomelo_delivery` write. All identifiers are deployment contracts, not suggestions for compatibility adapters.

## Topology

| Role | Application code | Version label | Components | Service | External network hostname |
| --- | --- | --- | --- | --- | --- |
| RAGFlow CPU | `ragflow-split` | `ragflow-split-cpu` | `tei`, `ragflow-cpu` | `default` | `ragflow-split-ragflow-cpu` |
| RAGFlow GPU | `ragflow-split` | `ragflow-split-gpu` | `tei`, `ragflow-cpu` | `default` | `ragflow-split-ragflow-cpu` |
| MySQL | `ragflow-mysql` | `mysql-8` | `mysql` | `default` | `ragflow-mysql-mysql` |
| Valkey | `ragflow-redis` | `valkey-8` | `redis` | `default` | `ragflow-redis-redis` |
| MinIO | `ragflow-minio` | `minio` | `minio` | `default` | `ragflow-minio-minio` |
| Elasticsearch | `ragflow-elasticsearch` | `elasticsearch-8` | `es01` | `default` | `ragflow-elasticsearch-es01` |

The RAGFlow split Application contains exactly the CPU and GPU Version labels. Its `default` Service selects GPU only when the current environment reports an NVIDIA GPU and the GPU preflight passes; it selects CPU when NVIDIA is unavailable. Backing Applications are optional only when an already-compatible, Orbit-managed resource fulfills the same contract. GPU never creates another TEI Application.

If NVIDIA is detected but the GPU preflight fails, stop. Do not silently change the selected profile to CPU.

Every Component joins the external `traefik` network. Hostnames follow `<application-code>-<component-name>`. Keep `default` as the only Service instance key.

## Runtime Config Primitive

```text
RagflowRuntimeConfig = {
  MYSQL_PASSWORD: string,
  REDIS_PASSWORD: string,
  MINIO_USER: string,
  MINIO_PASSWORD: string,
  ELASTIC_PASSWORD: string,
}
```

For a new complete resource set, create one value object in memory and use its keys consistently:

| Service | Required keys |
| --- | --- |
| `ragflow-mysql/default` | `MYSQL_PASSWORD` |
| `ragflow-redis/default` | `REDIS_PASSWORD` |
| `ragflow-minio/default` | `MINIO_USER`, `MINIO_PASSWORD` |
| `ragflow-elasticsearch/default` | `ELASTIC_PASSWORD` |
| `ragflow-split/default` | all five keys |

For a reused resource, retain its existing values. Do not derive secrets from an Application code, query them from MCP, or change them to fit RAGFlow. The caller must have the matching values through an authorized secret source before the RAGFlow Service can be configured.

## Component Primitive

| Component | Image / command | Logical mount | Health and special settings |
| --- | --- | --- | --- |
| `mysql` | `mysql:8.0.39`; preserve MySQL flags | named volume `mysql_data` -> `/var/lib/mysql` | `MYSQL_DATABASE=rag_flow`, `MYSQL_ROOT_HOST=%`; `mysqladmin ping`; no host port |
| `redis` | `valkey/valkey:8`; preserve password and 128 MB LRU command | named volume `redis_data` -> `/data` | `redis-cli` password health check; no host port |
| `minio` | `pgsty/minio:RELEASE.2026-03-25T00-00-00Z`; `server --console-address :9001 /data` | named volume `minio_data` -> `/data` | MinIO live health check; no host port |
| `es01` | `elasticsearch:8.11.3` | named volume `elasticsearch_data` -> `/usr/share/elasticsearch/data` | 8g memory limit, `/tmp` tmpfs 512 MiB mode 1777, unlimited memlock, security and disk watermark settings, authenticated health check |
| `tei` CPU | `ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07`; `--model-id /data/bge-m3 --json-output` | `tei/cache` -> `/data` | `127.0.0.1:80/health`; 30s interval, 5s timeout, 10 retries, 5m start period; no host port |
| `tei` GPU | `ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a`; same command | `tei/cache` -> `/data` | CPU settings plus device `nvidia`, `all`, `gpu` |
| `ragflow-cpu` | `infiniflow/ragflow:v0.26.4`; `--enable-adminserver --init-model-provider-tables` | named volume `ragflow_logs` -> `/ragflow/logs` | local dependency on `tei: service_healthy`; root HTTP health check |

Use the exact component environment values from `scripts/ragflow-split/docker-compose.*.yml`. Preserve `$${...}` in container-shell commands and health checks. Use `scripts/ragflow-split/` only for this explicit split topology and never run its Docker Compose lifecycle commands.

## RAGFlow Connectivity Primitive

```text
MYSQL_HOST=ragflow-mysql-mysql
MYSQL_PORT=3306
MYSQL_DBNAME=rag_flow
MYSQL_USER=root
REDIS_HOST=ragflow-redis-redis
MINIO_HOST=ragflow-minio-minio
ES_HOST=ragflow-elasticsearch-es01
ES_USER=elastic
TEI_BASE_URL=http://tei:80
```

Replace an expected hostname only after verifying the reused resource's external-network hostname and component contract. Never point at a localhost port, a public route, or the bundled RAGFlow Components.

## Service Expose Primitive

```json
{
  "component_name": "ragflow-cpu",
  "protocol": "http",
  "container_port": 80,
  "access": "gateway_http",
  "entrypoint": "web"
}
```

TEI and every backing component must have no `local`, `host`, `gateway_http`, or `gateway_tcp` endpoint. An `internal` TCP `6379` declaration is allowed and must never create a host mapping or public route. Do not add a route for a data service.

## MCP Workflow Primitive

```text
discover:  orbit_list_projects -> orbit_list_applications -> orbit_get_application
           -> orbit_list_versions / orbit_get_version -> orbit_list_application_services
preflight: runtime_doctor(network_name="traefik")
write:     orbit_create_application -> orbit_create_version
           -> orbit_create_service -> orbit_preview_service -> orbit_deploy
verify:    orbit_wait_deployment -> runtime_compose_ps -> runtime_http_probe
           -> verify_deployment
```

Inspect the write tool schema immediately before each write. Use `orbit_update_*` only to repair the matching target resource; do not mutate a merely similar Application. Sanitize all tool results before reporting them.
