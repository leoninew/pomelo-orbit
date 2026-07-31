# RAGFlow + TEI Bundled Mapping

Use this mapping for `scripts/ragflow-bundled/docker-compose.yml`. All components belong to one Version and one `default` Service.

## Component Mapping

| Component | CPU Version | GPU Version | Logical mount / required configuration |
| --- | --- | --- | --- |
| `mysql` | `mysql:8.0.39` | same | `mysql/data` -> `/var/lib/mysql`; preserve MySQL flags; use `${MYSQL_PASSWORD}` and initial `MYSQL_ROOT_HOST=%`. |
| `redis` | `valkey/valkey:8` | same | `redis/data` -> `/data`; preserve password command and healthcheck. |
| `minio` | `pgsty/minio:RELEASE.2026-03-25T00-00-00Z` | same | `minio/data` -> `/data`; use `${MINIO_USER}` and `${MINIO_PASSWORD}`. |
| `es01` | `elasticsearch:8.11.3` | same | `es01/data` -> Elasticsearch data directory; preserve `8g` memory limit, tmpfs, ulimit, security settings, and healthcheck. |
| `tei` | `ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07` | `ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a` | `tei/cache` -> `/data`; command `--model-id /data/bge-m3 --json-output`; CMD-SHELL healthcheck to `127.0.0.1:80/health`, 30s interval, 5s timeout, 10 retries, 5m start period. GPU also requests `nvidia/all/gpu`. |
| `ragflow-cpu` | `infiniflow/ragflow:v0.26.4` | same | `ragflow-cpu/logs` -> `/ragflow/logs`; preserve RAGFlow configuration and depend on all five dependencies with `service_healthy`. |

Use `${MYSQL_PASSWORD}`, `${REDIS_PASSWORD}`, `${MINIO_USER}`, `${MINIO_PASSWORD}`, and `${ELASTIC_PASSWORD}` as complete environment values. Keep `$${...}` in shell commands and healthchecks for container-shell expansion.

## Connectivity And Expose

`MYSQL_HOST=mysql`, `REDIS_HOST=redis`, `MINIO_HOST=minio`, and `ES_HOST=es01`. RAGFlow reaches TEI at `http://tei:80`.

The only Service Expose is `{component_name: ragflow-cpu, protocol: http, container_port: 80, access: local, listen_port: 9380}`. Do not expose TEI, storage, cache, or Elasticsearch.

## GPU Scope

The GPU Version is a static deployment specification in this delivery. Its note must contain `GPU runtime unverified`. A successful Compose preview does not validate NVIDIA Driver, Container Toolkit, CUDA compatibility, TEI `/health`, or embeddings. Complete those checks on the target GPU Host before reporting GPU readiness.
