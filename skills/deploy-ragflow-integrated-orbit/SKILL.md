---
name: deploy-ragflow-integrated-orbit
description: "Initialize the RAGFlow Orbit inventory and deploy its integrated six-component CPU or GPU Version through pomelo_delivery MCP. Use for all-in-one RAGFlow, MySQL, Valkey, MinIO, Elasticsearch, and TEI deployment; do not use for split runtime deployment."
---

# Deploy Integrated RAGFlow Through Orbit

Use this skill for the all-in-one topology defined by the generated integrated deployment primitive. Initialize split resources only as required by the shared inventory; do not deploy the split topology from this skill.

Read [shared initialization inventory](../ragflow-orbit-inventory.md) and [integrated component primitive](references/ragflow-integrated-components.md) before any `pomelo_delivery` write. The primitive is generated from `scripts/ragflow-integrated/deployment-contract.json` and is the source for topology identifiers, component configuration, model-cache paths, runtime-key ownership, network aliases, endpoints, and manual Provider initialization.

## Preflight And Initialization

1. Check generated-artifact consistency before making a deployment decision:

   ```powershell
   python scripts/render_ragflow_deployment_contract.py --check
   ```

   A drift result blocks this workflow until the matching contract is corrected and its generated artifacts are rewritten.
2. Discover NVIDIA hardware with a read-only `nvidia-smi` check, select CPU or GPU, and run `python scripts/prepare_ragflow_tei.py check --profile <selected> --model-dir <primitive-model-cache-path>` against every model-cache path in the generated primitive.
3. A verified selected cache and image are required before starting RAGFlow, not before inventory initialization. Initialization may create or repair Application, Version, Component, and stopped Service records while model preparation is in progress. Before `orbit_deploy` or any start/restart action, rerun the selected-profile preflight and stop if it fails. Ask for authorization before `prepare-model`, `prepare-image`, or any download. Do not make TEI download at startup or overwrite a non-empty cache.
4. List the Project, both RAGFlow Applications, split backing Applications, Versions, Services, and active Deployments. Run `runtime_doctor(network_name="traefik")`; provision the managed Gateway only when unhealthy.
5. Initialize every Application, Version, and stopped Service in the shared inventory. Generate or retain separate integrated and split runtime values inside their respective data-directory boundaries. Read current MCP schemas immediately before every write.

## Deploy Integrated Topology

1. Create or repair the integrated Versions and Components from the primitive. Apply its CPU/GPU difference, mount source types, aliases, component environment, dependency health gates, and endpoint definitions exactly.
2. Stop the selected split RAGFlow Service through Orbit with `remove_volumes=false` when it is running, then select the detected primitive-defined profile on the integrated Service. Do not deploy split backing Services in this workflow.
3. Preview and deploy only the selected integrated Version. Wait for all primitive-defined Components to become healthy, run the primitive-defined HTTP probe, then run `verify_deployment`.
4. Do not use this Application as a migration target for split RAGFlow data. The two RAGFlow topologies have intentionally separate deployment directories and backing stores.

## Completion

Keep Components within the primitive-defined endpoint contract; do not add a local, host, or public route. After health is confirmed, create the primitive-defined HuggingFace embedding provider manually in the RAGFlow UI.
