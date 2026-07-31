# Integrated RAGFlow Component Mapping

Use this mapping only for `ragflow-integrated-cpu` and `ragflow-integrated-gpu` in Application `ragflow`.

| Component | CPU / GPU mapping |
| --- | --- |
| `mysql` | `mysql:8.0.39`; `mysql/data` -> `/var/lib/mysql`; preserve MySQL flags, `MYSQL_ROOT_HOST=%`, and health check. |
| `redis` | `valkey/valkey:8`; `redis/data` -> `/data`; preserve password command and health check. |
| `minio` | `pgsty/minio:RELEASE.2026-03-25T00-00-00Z`; `minio/data` -> `/data`; preserve credentials and live health check. |
| `es01` | `elasticsearch:8.11.3`; `es01/data` -> Elasticsearch data; preserve 8g limit, `/tmp` tmpfs, memlock ulimit, security settings, and authenticated health check. |
| `tei` | CPU image `ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07`; GPU image `ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a` plus `nvidia/all/gpu`; `tei/cache` -> `/data`; command `--model-id /data/bge-m3 --json-output`; 5m health start period. |
| `ragflow-cpu` | `infiniflow/ragflow:v0.26.4`; `ragflow-cpu/logs` -> `/ragflow/logs`; depends on all five preceding Components with `service_healthy`. |

Use `${MYSQL_PASSWORD}`, `${REDIS_PASSWORD}`, `${MINIO_USER}`, `${MINIO_PASSWORD}`, and `${ELASTIC_PASSWORD}` as whole environment values. Preserve `$${...}` in container-shell commands and health checks. The internal RAGFlow hosts are `mysql`, `redis`, `minio`, and `es01`; TEI is `http://tei:80`. Read `scripts/ragflow-bundled/docker-compose.yml` for the exact environment and health-check collections before a write.
