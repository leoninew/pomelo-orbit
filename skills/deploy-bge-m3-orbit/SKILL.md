---
name: deploy-bge-m3-orbit
description: Deploy, rebuild, validate, or export the BAAI/bge-m3 Text Embeddings Inference CPU service through Pomelo Orbit MCP. Use for ghcr.io/huggingface/text-embeddings-inference, data/deployment/bge-m3/default/docker-compose.yml, data/deployment/tei-bge-m3/default/docker-compose.yml, local port 8081, and preparing or reusing the local Git LFS BGE-M3 model cache for offline startup.
---

# Deploy BGE-M3 With Orbit

Deploy BGE-M3 as one standard Orbit Application, one Version with a `tei` Component, and one `default` Service. Use `pomelo_orbit` MCP for every deployment lifecycle write; do not use Docker Compose lifecycle commands directly.

## Inspect First

1. Read the relevant historical Compose inputs. `bge-m3` records the original remote-model layout; `tei-bge-m3` records the preferred local-model layout.
2. Before any MCP write, inspect the current `pomelo_orbit` tool schemas, then list Orbit projects and Applications. Reuse an existing `bge-m3` Application only after inspecting its Version, Service, and deployment state.
3. Run `runtime_doctor(network_name="traefik")`. Standard applications join this external network even when they expose only a local endpoint. When it is missing or unhealthy, provision the managed Gateway through `orbit_provision_gateway`; never create the network with Docker CLI.

## Prepare The Local Model

Use the local Git LFS checkout instead of passing `BAAI/bge-m3` to TEI. The expected checkout is `<repo>/data/deployment/tei-bge-m3/default/tei/cache/bge-m3`; mount its parent `cache` directory at `/data`, then pass `--model-id /data/bge-m3`.

Before deployment, verify the checkout has actual model content, not merely a directory or LFS pointer files. At minimum, require `config.json`, `pytorch_model.bin`, tokenizer files, and the `onnx/` directory. The existing full checkpoint is several GiB, so a quick deployment task should not redownload it.

When the checkout is missing or incomplete, prepare it before the Orbit deployment on an empty target directory:

```powershell
git lfs install
git lfs clone https://huggingface.co/BAAI/bge-m3 <cache-root>/bge-m3
```

When `git lfs clone` is unavailable, use `git clone` followed by `git -C <cache-root>/bge-m3 lfs pull`. Do not delete, reset, or overwrite an existing cache to repair it. Resolve a non-empty incomplete cache with the user. Do not use `BAAI/bge-m3` as the TEI model ID once the local checkout is available, because that re-enters Hugging Face download behavior.

## Create Or Update The Topology

1. Create the standard Application as `BGE-M3 Embeddings`, code `bge-m3`, `image_pull_policy: missing`, and use the initial draft Version. Label it `bge-m3-cpu`. Do not create a second empty Version.
2. Create one Component named `tei` using the image reference from the selected Compose input, `pull_policy: missing`, and `restart_policy: unless-stopped`.
3. Set the command to `--model-id /data/bge-m3 --json-output`.
4. Set one mount with `source_type: directory`, `source: <absolute cache-root>`, `target: /data`, `source_is_host_path: true`, and `read_only: false`. The API requires an absolute source path; do not submit a relative Compose path.
5. Set the health check to `CMD-SHELL` with `curl -fsS http://127.0.0.1:80/health`, interval `30s`, timeout `5s`, retries `10`, and start period `5m`.
6. Create exactly one `default` Service with empty `runtime_config` and one local HTTP Expose: `{component_name: tei, protocol: http, container_port: 80, access: local, listen_port: 8081}`. Do not use component ports or version-level Traefik labels for this mapping.

## Deploy And Verify

1. Use `orbit_preview_service` before deployment.
2. Call `orbit_deploy(service_id)` without `force_recreate`, then use `orbit_wait_deployment` for its terminal status.
3. Check `runtime_compose_ps`. A temporary `starting` health state is normal while the CPU model loads; inspect scoped logs only when it does not advance or the container restarts.
4. Confirm `runtime_http_probe` reaches component `tei`, port `80`, path `/health`, then run `verify_deployment`.
5. Report the health result and `http://localhost:8081/health`. Do not print deployment internals or runtime configuration.

## Diagnostics

- If the host source path is rejected, resolve it to an absolute path and preserve `source_is_host_path: true`.
- If TEI attempts model download, inspect the saved command and mount: it must use `/data/bge-m3` and mount the parent cache directory at `/data`.
- If the health check remains `starting`, wait through the configured start period before treating it as a failure. Use `runtime_compose_logs` scoped to `tei` for evidence; do not replace the model cache or force-recreate the Service automatically.
- If local port `8081` conflicts, report the owner/conflict and request a port decision rather than silently selecting another port.
