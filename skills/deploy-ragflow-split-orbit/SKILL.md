---
name: deploy-ragflow-split-orbit
description: "Create, repair, preview, or deploy the split RAGFlow topology through pomelo_delivery MCP: one RAGFlow Application containing TEI plus independently reusable or provisioned MySQL, Valkey, MinIO, and Elasticsearch Applications. Use only when the split topology is explicitly requested; never use for the bundled RAGFlow deployment."
---

# Deploy Split RAGFlow Through Orbit

Deploy the split RAGFlow topology defined by the generated deployment primitive. Reuse compatible Orbit-managed backing resources when they already exist; otherwise provision the missing resources as independent Applications. Do not modify, convert, or deploy the integrated topology except to stop its selected RAGFlow Service before starting split RAGFlow.

Read [shared initialization inventory](../ragflow-orbit-inventory.md) and [references/ragflow-split-primitives.md](references/ragflow-split-primitives.md) before any `pomelo_delivery` write. The primitive is generated from `scripts/ragflow-split/deployment-contract.json` and is the source for topology identifiers, component configuration, model cache paths, runtime-key ownership, network aliases, endpoints, and manual Provider initialization.

## Preconditions

1. Confirm the user explicitly requests split deployment. Use the generated primitive as the authority for the split Application inventory; never create a standalone TEI Application.
2. Check generated-artifact consistency before making a deployment decision:

   ```powershell
   python scripts/render_ragflow_split_contract.py --check
   ```

   A drift result blocks this workflow until the contract is corrected and its generated artifacts are rewritten.
3. Discover the runtime profile with a read-only NVIDIA check. Select GPU only when an NVIDIA GPU is present; otherwise select CPU. Run the selected profile's preflight against every model-cache path in the generated primitive:

   ```powershell
   nvidia-smi --query-gpu=name,driver_version --format=csv,noheader
   ```

   For each listed path, run `python scripts/prepare_ragflow_tei.py check --profile <selected-profile> --model-dir <primitive-model-cache-path>`. A present NVIDIA GPU selects GPU; if its GPU preflight fails, stop rather than silently falling back to CPU. A missing cache or unavailable selected image blocks deployment or startup, but not inventory initialization of Applications, Versions, Components, and stopped Services. Before `orbit_deploy` or any start/restart action, rerun the selected-profile preflight and stop if it fails. Ask for permission before a model download or image pull. Restore or stage a verified archive only into an empty target.
4. List Projects and Applications. Inspect both RAGFlow topologies, the primitive-defined backing Applications, Versions, Services, and active Deployments before changing anything. Run `runtime_doctor(network_name="traefik")`; use `orbit_provision_gateway` only when the managed Gateway or network is unhealthy.
5. Before every `pomelo_delivery` write, inspect the current MCP tool schema. Use only `pomelo_delivery` for lifecycle writes. Do not use Docker Compose lifecycle commands.

## Resource Discovery

Evaluate every backing resource against the generated primitive:

- Reuse only an Orbit-managed Application whose Component, image family, health check, external-network hostname, and credential contract are compatible.
- Treat absent, incomplete, incompatible, or running-unhealthy candidates as unresolved. A stopped Service created during initialization is expected; deploy it rather than creating a duplicate. Preserve an already running compatible, healthy backing Service. Do not silently substitute a different database, cache, bucket store, or Elasticsearch cluster.
- The MCP intentionally does not expose runtime secret values. If a compatible existing resource's matching credentials are not already available through the authorized secure context, stop and request them. Never guess, rotate, export, or print existing credentials.
- When a required resource is absent and its data directory is new and empty, generate one in-memory `RagflowRuntimeConfig` and apply the relevant keys to every new backing Service and to `ragflow-split`. Do not persist it in SQL, files, logs, or chat output.
- If any existing MySQL or Elasticsearch data directory is non-empty, retain its matching bootstrap credentials. A reset requires an explicit user decision.

## Provisioning Workflow

1. Initialize every Application, Version, and stopped Service in the shared inventory. For every absent backing resource, create the primitive-defined Application, complete Version, and stopped Service. Stop the selected integrated RAGFlow Service through Orbit with `remove_volumes=false` when it is running.
2. Preview and deploy every split backing Service through Orbit. Wait for its Component health before deploying split RAGFlow.
3. Create or repair the split RAGFlow Versions and Components from the primitive. Apply its CPU/GPU difference, mount source types, aliases, component environment, and endpoint definitions exactly. Add the note `GPU runtime unverified` until inference is tested on an NVIDIA host.
4. Use the discovered or newly created hostnames only when they satisfy the primitive contract. RAGFlow may depend on TEI locally, but cross-Application backing resources have no Compose `depends_on`; deploy and verify those resources first. Initialize and verify the managed Gateway according to its current configured endpoint contract.
5. Select the primitive-defined GPU Version only when GPU was detected and passed preflight; otherwise select its CPU Version. Preview and deploy only that Version. Wait until its Components are healthy, run the primitive-defined HTTP probe, then run `verify_deployment`.

## Completion And Safety

- Keep the final topology within the primitive-defined split Application and backing-Application inventory. Do not create model, temporary, duplicate, or bundled applications.
- Do not expose Components beyond the primitive-defined endpoint contract. Do not add public routes.
- Do not render or echo raw deployment records, previews, logs, or status payloads that may contain runtime configuration. Report only sanitized state and health summaries.
- After the selected Version is healthy, configure the primitive-defined HuggingFace embedding provider manually in the UI. Do not automate provider registration or API-key handling.
- Do not migrate, mount, or reuse `ragflow-integrated` data directories. The two RAGFlow Applications intentionally isolate persistent stores.
