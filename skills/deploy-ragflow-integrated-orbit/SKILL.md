---
name: deploy-ragflow-integrated-orbit
description: "Initialize, repair, preview, or deploy only the integrated RAGFlow topology through pomelo_delivery MCP: one ragflow-integrated Application with CPU and GPU six-component Versions. Use for the default all-in-one RAGFlow deployment; do not initialize split or backing Applications."
---

# Deploy Integrated RAGFlow Through Orbit

Use this skill for the all-in-one topology in [the integrated primitive](references/ragflow-integrated-components.md). The primitive is generated from this directory's `deployment-contract.json` and is authoritative for component configuration, runtime-key ownership, aliases, endpoints, model-cache checks, and Provider initialization.

## Preconditions

1. Confirm the request is for the integrated topology. This is the default when the user requests RAGFlow without explicitly requesting split deployment.
2. Check only the integrated generated artifacts:

   ```powershell
   python skills/deploy-ragflow-integrated-orbit/render_contract.py --check
   ```

   Drift blocks this workflow until the integrated contract is corrected and regenerated.
3. Discover NVIDIA hardware with a read-only check. Select GPU only when NVIDIA is available. Run `python skills/_ragflow/prepare_ragflow_tei.py check --profile <selected-profile> --model-dir <primitive-model-cache-path>` against every path listed in the integrated primitive. A selected cache and image are required before startup, not before inventory initialization. Before `orbit_deploy` or any start/restart action, rerun the preflight. A detected GPU whose GPU preflight fails blocks deployment; do not fall back to CPU. Ask for authorization before download, image pull, staging, restore, or overwrite of a non-empty cache.
4. List Projects, Applications, Versions, Services, and active Deployments. Inspect `ragflow-integrated` and `ragflow-split/default` only as needed to enforce RAGFlow single-active routing. Run `runtime_doctor(network_name="traefik")`; use `orbit_provision_gateway` only when the managed Gateway or network is unhealthy.
5. Before every `pomelo_delivery` write, inspect the live tool schema. Use only `pomelo_delivery` for lifecycle writes; do not use Docker Compose lifecycle commands.

## Initialize Integrated Inventory

1. Create or repair only `ragflow-integrated`, both complete six-component Versions `ragflow-integrated-cpu` and `ragflow-integrated-gpu`, and its `default` Service. New Services remain stopped after initialization.
2. Apply the primitive's component declarations exactly, including the TEI CPU/GPU image difference, GPU device request, mounts, aliases, environment, health gates, and endpoints.
3. For a new and empty integrated data boundary, create one in-memory `IntegratedRuntimeConfig` and apply it only to `ragflow-integrated/default`. For non-empty data, retain matching authorized values or stop for an explicit reset decision. Never print, export, persist, infer, or rotate runtime secrets.
4. Do not create, repair, deploy, or initialize `ragflow-split` or any split backing Application in this workflow.

## Deploy Integrated Topology

1. Before startup, stop `ragflow-split/default` through Orbit with `remove_volumes=false` when it is running. Do not alter its Versions, backing Services, data, or cache.
2. Select `ragflow-integrated-gpu` only after GPU detection and preflight succeed; otherwise select `ragflow-integrated-cpu`.
3. Preview and deploy only `ragflow-integrated/default`. Wait until every primitive-defined Component is healthy, run the primitive HTTP probe, then run `verify_deployment`.

## Completion And Safety

- Keep Components within the primitive-defined endpoint contract; do not add local, host, or public routes.
- Do not use integrated storage as a migration target for split data.
- Sanitize tool results before reporting them; never render raw runtime configuration, preview, deployment records, or logs.
- After health is confirmed, configure the primitive-defined HuggingFace embedding provider manually in the RAGFlow UI.
