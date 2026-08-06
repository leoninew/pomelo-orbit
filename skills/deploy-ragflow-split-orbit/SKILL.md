---
name: deploy-ragflow-split-orbit
description: "Initialize, repair, preview, or deploy only the split RAGFlow topology through pomelo_delivery MCP: ragflow-split with CPU and GPU Versions plus MySQL, Valkey, MinIO, and Elasticsearch Applications. Use only when split deployment is explicitly requested; do not initialize the integrated Application."
---

# Deploy Split RAGFlow Through Orbit

Use this skill only for the split topology in [the split primitive](references/ragflow-split-primitives.md). The primitive is generated from this directory's `deployment-contract.json` and is authoritative for application inventory, component configuration, runtime-key ownership, aliases, endpoints, model-cache checks, and Provider initialization.

## Preconditions

1. Confirm the user explicitly requests split deployment.
2. Check only the split generated artifacts:

   ```powershell
   python skills/deploy-ragflow-split-orbit/render_contract.py --check
   ```

   Drift blocks this workflow until the split contract is corrected and regenerated.
3. Discover NVIDIA hardware with a read-only check. Select GPU only when NVIDIA is available. Run `python skills/_ragflow/prepare_ragflow_tei.py check --profile <selected-profile> --model-dir <primitive-model-cache-path>` against every path listed in the split primitive. A selected cache and image are required before startup, not before inventory initialization. Before `orbit_deploy` or any start/restart action, rerun the preflight. A detected GPU whose GPU preflight fails blocks deployment; do not fall back to CPU. Ask for authorization before download, image pull, staging, restore, or overwrite of a non-empty cache.
4. List Projects, Applications, Versions, Services, and active Deployments. Inspect the split Applications and `ragflow-integrated/default` only as needed to enforce RAGFlow single-active routing. Run `runtime_doctor(network_name="traefik")`; use `orbit_provision_gateway` only when the managed Gateway or network is unhealthy.
5. Before every `pomelo_delivery` write, inspect the live tool schema. Use only `pomelo_delivery` for lifecycle writes; do not use Docker Compose lifecycle commands.

## Initialize Split Inventory

1. Create or repair only `ragflow-split`, both complete Versions `ragflow-split-cpu` and `ragflow-split-gpu`, and its `default` Service. Create or repair `ragflow-mysql`, `ragflow-redis`, `ragflow-minio`, and `ragflow-elasticsearch`, each with the primitive-defined complete Version and `default` Service. New Services remain stopped after initialization.
2. Apply the primitive's component declarations exactly, including the RAGFlow/TEI CPU/GPU difference, independent backing Components, mounts, aliases, environment, health gates, and endpoints.
3. Reuse a backing Service only when its Orbit-managed Application, image family, health check, external-network hostname, and credential contract are compatible. Preserve a running compatible healthy backing Service; do not silently substitute a different database, cache, bucket store, or Elasticsearch cluster.
4. For a new and empty split resource set, create one in-memory `RagflowRuntimeConfig` and apply its required keys to `ragflow-split/default` and each new backing Service. For non-empty MySQL or Elasticsearch data, retain matching authorized bootstrap values or stop for an explicit reset decision. Never guess, rotate, export, persist, or print runtime secrets.
5. Do not create, repair, deploy, or initialize `ragflow-integrated` in this workflow.

## Deploy Split Topology

1. Before startup, stop `ragflow-integrated/default` through Orbit with `remove_volumes=false` when it is running. Do not alter its Versions, data, or cache.
2. Preview and deploy each split backing Service through Orbit, then wait for its Component health before starting split RAGFlow.
3. Select `ragflow-split-gpu` only after GPU detection and preflight succeed; otherwise select `ragflow-split-cpu`.
4. Preview and deploy only `ragflow-split/default`. Wait until every primitive-defined Component is healthy, run the primitive HTTP probe, then run `verify_deployment`.

## Completion And Safety

- Keep the final topology within the primitive-defined split inventory. Do not create temporary, duplicate, bundled, or model Applications.
- Do not expose Components beyond the primitive-defined endpoint contract or add public routes.
- Do not migrate, mount, or reuse `ragflow-integrated` data directories.
- Sanitize tool results before reporting them; never render raw runtime configuration, preview, deployment records, or logs.
- After health is confirmed, configure the primitive-defined HuggingFace embedding provider manually in the RAGFlow UI.
