---
name: deploy-ragflow-split-orbit
description: "Create, repair, preview, or deploy the split RAGFlow topology through pomelo_delivery MCP: one RAGFlow Application containing TEI plus independently reusable or provisioned MySQL, Valkey, MinIO, and Elasticsearch Applications. Use only when the split topology is explicitly requested; never use for the bundled RAGFlow deployment."
---

# Deploy Split RAGFlow Through Orbit

Deploy one `ragflow-split` Application containing `tei` and `ragflow-cpu`. Reuse compatible Orbit-managed MySQL, Valkey, MinIO, and Elasticsearch resources when they already exist; otherwise provision each resource as its own Application. Do not modify, convert, or deploy the integrated `ragflow-integrated` Application.

Use these names only for the split topology: Application code `ragflow-split`, Version labels `ragflow-split-cpu` and `ragflow-split-gpu`, and Service instance key `default`. The backing Application codes are `ragflow-mysql`, `ragflow-redis`, `ragflow-minio`, and `ragflow-elasticsearch`; each has one `default` Service.

Read [shared initialization inventory](../ragflow-orbit-inventory.md), [references/ragflow-split-primitives.md](references/ragflow-split-primitives.md), and [integrated component mapping](../deploy-ragflow-integrated-orbit/references/ragflow-integrated-components.md) before any `pomelo_delivery` write. Together they define the initialization scope, topology, compatibility contract, component mapping, secret ownership, and MCP primitives.

## Preconditions

1. Confirm the user explicitly requests split deployment. This topology contains up to five Applications: `ragflow-split`, `ragflow-mysql`, `ragflow-redis`, `ragflow-minio`, and `ragflow-elasticsearch`. `tei` is never a standalone Application.
2. Discover the runtime profile with a read-only NVIDIA check. Select GPU only when an NVIDIA GPU is present; otherwise select CPU. Run the selected profile's preflight against both RAGFlow model directories:

   ```powershell
   nvidia-smi --query-gpu=name,driver_version --format=csv,noheader
   python scripts/prepare_ragflow_tei.py check --profile cpu --model-dir data/deployment/ragflow-integrated/default/tei/cache/bge-m3
   python scripts/prepare_ragflow_tei.py check --profile cpu --model-dir data/deployment/ragflow-split/default/tei/cache/bge-m3
   ```

   Use the same two commands with `--profile gpu` when NVIDIA was detected. A present NVIDIA GPU selects GPU; if its GPU preflight fails, stop rather than silently falling back to CPU. A missing model cache or unavailable selected image blocks deployment or startup, but not inventory initialization of Applications, Versions, Components, and stopped Services. Before `orbit_deploy` or any start/restart action, rerun the selected-profile preflight and stop if it fails. Ask for permission before a model download or image pull. Restore or stage a verified archive only into an empty target.
3. List Projects and Applications. Inspect both RAGFlow Applications, all four backing Applications, Versions, Services, and active Deployments before changing anything. Run `runtime_doctor(network_name="traefik")`; use `orbit_provision_gateway` only when the managed Gateway or network is unhealthy.
4. Before every `pomelo_delivery` write, inspect the current MCP tool schema. Use only `pomelo_delivery` for lifecycle writes. Do not use Docker Compose lifecycle commands.

## Resource Discovery

Evaluate every backing resource against the contract in the reference:

- Reuse only an Orbit-managed Application whose Component, image family, health check, external-network hostname, and credential contract are compatible.
- Treat absent, incomplete, incompatible, or running-unhealthy candidates as unresolved. A stopped Service created during initialization is expected; deploy it rather than creating a duplicate. Preserve an already running compatible, healthy backing Service. Do not silently substitute a different database, cache, bucket store, or Elasticsearch cluster.
- The MCP intentionally does not expose runtime secret values. If a compatible existing resource's matching credentials are not already available through the authorized secure context, stop and request them. Never guess, rotate, export, or print existing credentials.
- When a required resource is absent and its data directory is new and empty, generate one in-memory `RagflowRuntimeConfig` and apply the relevant keys to every new backing Service and to `ragflow-split`. Do not persist it in SQL, files, logs, or chat output.
- If any existing MySQL or Elasticsearch data directory is non-empty, retain its matching bootstrap credentials. A reset requires an explicit user decision.

## Provisioning Workflow

1. Initialize every Application, Version, and stopped Service in the shared inventory. For every absent backing resource, create the Application, complete Version, and one stopped `default` Service from the reference. Stop `ragflow-integrated/default` through Orbit with `remove_volumes=false` when it is running.
2. Preview and deploy every split backing Service through Orbit. Wait for its Component health before deploying split RAGFlow.
3. Create or repair `ragflow-split` with exactly two Versions: `ragflow-split-cpu` and `ragflow-split-gpu`. Each contains exactly `tei` and `ragflow-cpu`; the GPU Version changes only the fixed CUDA TEI image and `nvidia/all/gpu` device request. Add the note `GPU runtime unverified` until inference is tested on an NVIDIA host. Use the source types and logical source names in the split primitive: the TEI cache is a directory, while backing data and RAGFlow logs are named volumes. Keep TEI internal; the only host expose is `127.0.0.1:9380 -> ragflow-cpu:80`.
4. Use the discovered or newly created network hostnames in `ragflow-cpu` environment values. RAGFlow may depend on `tei` locally, but cross-Application backing resources have no Compose `depends_on`; deploy and verify those resources first.
5. Select `ragflow-split-gpu` for the `default` Service when GPU was detected and passed preflight; otherwise select `ragflow-split-cpu`. Preview and deploy only that Version. Wait until both RAGFlow Components are healthy, probe `ragflow-cpu:80/`, then run `verify_deployment`.

## Completion And Safety

- Keep the final topology limited to one RAGFlow split Application plus zero to four backing Applications. Do not create `bge-m3`, temporary, duplicate, or bundled applications.
- Do not expose TEI, MySQL, Valkey, MinIO, or Elasticsearch. Do not add public routes.
- Do not render or echo raw deployment records, previews, logs, or status payloads that may contain runtime configuration. Report only sanitized state and health summaries.
- After the selected Version is healthy, configure the RAGFlow HuggingFace embedding provider manually in the UI: `tei-bge-m3`, `BAAI/bge-m3`, `http://tei:80`, and `8192`. Do not automate provider registration or API-key handling.
- Do not migrate, mount, or reuse `ragflow-integrated` data directories. The two RAGFlow Applications intentionally isolate persistent stores.
