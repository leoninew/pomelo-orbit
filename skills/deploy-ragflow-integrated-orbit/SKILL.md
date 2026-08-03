---
name: deploy-ragflow-integrated-orbit
description: "Initialize the RAGFlow Orbit inventory and deploy its integrated six-component CPU or GPU Version through pomelo_delivery MCP. Use for all-in-one RAGFlow, MySQL, Valkey, MinIO, Elasticsearch, and TEI deployment; do not use for split runtime deployment."
---

# Deploy Integrated RAGFlow Through Orbit

Use this skill for the all-in-one deployment topology. It owns Application `ragflow-integrated`, its six-component Versions `ragflow-integrated-cpu` and `ragflow-integrated-gpu`, and one `default` Service. Initialize Application `ragflow-split` only as required by the shared inventory; do not deploy it from this skill.

Read [shared initialization inventory](../ragflow-orbit-inventory.md), [integrated component mapping](references/ragflow-integrated-components.md), and [split primitives](../deploy-ragflow-split-orbit/references/ragflow-split-primitives.md) before any `pomelo_delivery` write.

## Preflight And Initialization

1. Discover NVIDIA hardware with a read-only `nvidia-smi` check, select CPU or GPU, and run `python scripts/prepare_ragflow_tei.py check --profile <selected>` against both `ragflow-integrated` and `ragflow-split` model-cache paths from the shared inventory.
2. A verified selected cache and image are required before starting RAGFlow, not before inventory initialization. Initialization may create or repair Application, Version, Component, and stopped Service records while model preparation is in progress. Before `orbit_deploy` or any start/restart action, rerun the selected-profile preflight and stop if it fails. Ask for authorization before `prepare-model`, `prepare-image`, or any download. Do not make TEI download `BAAI/bge-m3` at startup or overwrite a non-empty cache.
3. List the Project, both RAGFlow Applications, four split backing Applications, Versions, Services, and active Deployments. Run `runtime_doctor(network_name="traefik")`; provision the managed Gateway only when unhealthy.
4. Initialize every Application, Version, and stopped Service in the shared inventory. Generate or retain separate integrated and split runtime values inside their respective data-directory boundaries. Read current MCP schemas immediately before every write.

## Deploy Integrated Topology

1. Create or repair `ragflow-integrated-cpu` and `ragflow-integrated-gpu` with exactly six Components: `mysql`, `redis`, `minio`, `es01`, `tei`, and `ragflow-cpu`. Preserve the fixed image digests, logical mounts, dependency health gates, and GPU device request from the mapping.
2. Stop `ragflow-split/default` through Orbit with `remove_volumes=false` when it is running, then select the detected profile on `ragflow-integrated/default` with `orbit_update_service_basic`. Do not deploy split backing Services in this workflow.
3. Preview and deploy only the selected integrated Version. Wait for all six Components to become healthy, probe `ragflow-cpu:80/`, then run `verify_deployment`.
4. Do not use this Application as a migration target for split RAGFlow data. `ragflow-integrated` and `ragflow-split` have intentionally separate deployment directories and backing stores.

## Completion

Keep TEI and all backing Components internal. Expose `ragflow-cpu:80` only as `gateway_http` through the Gateway `web` entrypoint; do not add a local or host mapping. After health is confirmed, create the HuggingFace embedding provider manually in the RAGFlow UI with `ragflow-split-tei`, `BAAI/bge-m3`, `http://ragflow-split-tei:80`, and `8192`.
