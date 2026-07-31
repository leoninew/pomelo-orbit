# RAGFlow Orbit Initialization Inventory

Both RAGFlow deployment skills initialize this complete inventory before they select which topology to deploy. Initialization creates configuration and stopped Services; it does not start the topology owned by the other skill.

## Delivery MCP Contract

Use the Codex MCP server `pomelo_delivery`. Its local stdio command is `uv --directory mcp/src/delivery-mcp run pomelo-delivery-mcp`, and its tools appear as `mcp__pomelo_delivery__orbit_*`; the `orbit_*` tool names remain because they adapt the Pomelo Orbit control plane. Do not use the removed `pomelo_orbit` registration or `pomelo-orbit-mcp` command.

## Application, Version, And Service Names

| Topology or role | Application code | Version labels | Service |
| --- | --- | --- | --- |
| Integrated RAGFlow + TEI | `ragflow` | `ragflow-integrated-cpu`, `ragflow-integrated-gpu` | `default` |
| Split RAGFlow + TEI | `ragflow-split` | `ragflow-split-cpu`, `ragflow-split-gpu` | `default` |
| Split MySQL | `ragflow-mysql` | `mysql-8` | `default` |
| Split Valkey | `ragflow-redis` | `valkey-8` | `default` |
| Split MinIO | `ragflow-minio` | `minio` | `default` |
| Split Elasticsearch | `ragflow-elasticsearch` | `elasticsearch-8` | `default` |

The two RAGFlow Applications and all five `default` Services have distinct managed deployment directories. Never mount one Application's data directory into another Application.

## Required Initialization

1. List Projects, Applications, Versions, Services, and active Deployments before any write.
2. Create or repair `ragflow` with its two six-component Versions and one `default` Service.
3. Create or repair `ragflow-split` with its two two-component Versions and one `default` Service.
4. Create or repair each split backing Application with one complete Version and one stopped `default` Service. Initialize them even when the invoked skill will deploy integrated RAGFlow.
5. Keep all Components attached to the managed external `traefik` network. Run `runtime_doctor(network_name="traefik")`; use `orbit_provision_gateway` only when unhealthy.

## Model Caches And Runtime Values

Initialize and verify both separate TEI caches before lifecycle writes:

```text
data/deployment/ragflow/default/tei/cache/bge-m3
data/deployment/ragflow-split/default/tei/cache/bge-m3
```

Run the selected CPU or GPU preflight against each path. When one verified cache exists and the other target is empty, use `stage-model` to make a separately verified copy. Do not download, restore, stage, or overwrite a model cache without the required authorization or into a non-empty target.

Create independent runtime value maps for the two data boundaries only when their relevant directories are new and empty:

```text
IntegratedRuntimeConfig = { MYSQL_PASSWORD, REDIS_PASSWORD, MINIO_USER, MINIO_PASSWORD, ELASTIC_PASSWORD }
SplitRuntimeConfig      = { MYSQL_PASSWORD, REDIS_PASSWORD, MINIO_USER, MINIO_PASSWORD, ELASTIC_PASSWORD }
```

Apply `IntegratedRuntimeConfig` only to `ragflow/default`. Apply `SplitRuntimeConfig` consistently to `ragflow-split/default` and the four split backing Services. The MCP does not return secret values. For non-empty existing stores, stop for matching authorized values or an explicit reset decision; never infer, print, export, or rotate them.

## Selection And Deployment

Use a read-only NVIDIA check before profile selection. If NVIDIA is unavailable, select the CPU Version. If NVIDIA is available, select GPU and require its GPU preflight to pass; do not silently fall back to CPU.

- `deploy-ragflow-integrated-orbit` deploys only `ragflow/default` with `ragflow-integrated-{cpu,gpu}`. It leaves `ragflow-split/default` stopped; split backing Services may remain running.
- `deploy-ragflow-split-orbit` deploys all four split backing Services, waits for their health, then deploys only `ragflow-split/default` with `ragflow-split-{cpu,gpu}`. It leaves `ragflow/default` untouched.

Both RAGFlow Services expose `127.0.0.1:9380`, so only one may run. Before deployment, stop the other RAGFlow Service through Orbit with `remove_volumes=false`; do not stop or remove its backing data. Split backing Services may remain running while integrated RAGFlow is selected.

Before every `pomelo_delivery` lifecycle write, inspect the live tool schema. Use only the delivery MCP for lifecycle changes and sanitize result summaries before reporting them.
