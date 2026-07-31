---
name: deploy-ragflow-tei-orbit
description: Idempotently deploy, rebuild, validate, prepare, create, or repair the bundled RAGFlow application with its in-process BAAI/bge-m3 Text Embeddings Inference component through Pomelo Orbit. Use for scripts/ragflow-bundled/docker-compose.yml, RAGFlow CPU/GPU Version selection, TEI model cache preparation, local port 9380, and RAGFlow HuggingFace provider initialization.
---

# Deploy RAGFlow With TEI Through Orbit

Deploy one `ragflow` Application with two complete Versions: `ragflow-tei-cpu` and `ragflow-tei-gpu`. Each Version has `mysql`, `redis`, `minio`, `es01`, `tei`, and `ragflow-cpu`; the TEI image and GPU device request are the only intended differences. Keep one `default` Service and select CPU initially.

Read [references/ragflow-tei-bundled.md](references/ragflow-tei-bundled.md) before creating or changing a Version.

## Prepare First

1. Run the selected profile's read-only preflight:

   ```powershell
   python scripts/prepare_ragflow_tei.py check --profile cpu
   python scripts/prepare_ragflow_tei.py check --profile gpu
   ```

2. If the cache is missing, incomplete, or the selected image is unavailable, stop before Orbit writes and identify the precise next command. Only run the write-capable preparation commands after the user authorizes the download or image pull:

   ```powershell
   python scripts/prepare_ragflow_tei.py prepare-model --source git-lfs
   python scripts/prepare_ragflow_tei.py prepare-image --profile cpu
   ```

   Use `--source hf-cli` only when Git LFS cannot be used. Do not make TEI download `BAAI/bge-m3` during container startup. Do not overwrite a non-empty cache.

3. A GPU preflight failure blocks GPU deployment. This skill does not claim GPU container or model inference validation without a suitable NVIDIA Host.

## Orbit Workflow

1. Before each `pomelo_orbit` write, inspect the current MCP tool schemas. Do not infer write parameters from this file.
2. List Projects and Applications, then inspect an existing `ragflow` Application, Versions, Services, and active Deployments before changing it.
3. Run `runtime_doctor(network_name="traefik")`. When the managed Gateway or network is missing or unhealthy, use `orbit_provision_gateway`; do not create the network with Docker or Compose.
4. Create or update the RAGFlow Application according to the reference. Use logical directory mounts, not source Compose paths. Do not create or retain a standalone `bge-m3` Application.
5. Ensure the final Version set idempotently:
   - Inspect existing Versions first.
   - If `ragflow-tei-cpu` or `ragflow-tei-gpu` is absent, create it as a complete six-component Version.
   - If either label exists but is incomplete, unpublished, or stale, update that Version to the final specification instead of duplicating it.
   - For GPU, keep all shared components identical to CPU, replace only TEI with the fixed CUDA image, set its device request to `nvidia`, `all`, and `gpu`, and include the `GPU runtime unverified` note.
   - Do not Fork or retain temporary Versions; the final Application contains exactly these two Version labels.
6. Create or retain only the `default` Service, initially selecting `ragflow-tei-cpu`. It has one local RAGFlow HTTP Expose at `127.0.0.1:9380 -> ragflow-cpu:80`. TEI has no host port or public route.
7. Generate runtime values only for a fresh, empty data directory. For existing MySQL or Elasticsearch directories, retain matching bootstrap credentials or obtain an explicit reset decision. Never print, persist, or export runtime configuration values.
8. Preview and deploy only the selected Version through Orbit. For CPU, wait for all six Components to become healthy, probe `ragflow-cpu:80/`, then run `verify_deployment`. GPU deployment requires a future NVIDIA Host validation and is not part of this baseline.

## RAGFlow Initialization

After the CPU Service is healthy, use the RAGFlow UI form to create the HuggingFace Embedding Provider:

- Name: `tei-bge-m3`
- Model: `BAAI/bge-m3`
- Base URL: `http://tei:80`
- Max tokens: `8192`

Do not append `/embed`. Do not use the removed standalone TEI hostname. Do not automate registration, API-key acquisition, or provider setup through the RAGFlow API.

## Safety

- Use `pomelo_orbit` for all deployment lifecycle writes; never use Docker Compose lifecycle commands.
- Do not export Deployment history, a running Service state, runtime configuration, or RAGFlow's internal MySQL provider configuration into the control-plane baseline SQL.
- Preserve original Git LFS checkouts. Use `backup-model` and `restore-model` for repeated cache tests; restoration only targets an empty directory.
