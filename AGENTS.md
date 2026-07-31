# Repository Agent Instructions

## RAGFlow Deployment

- Use the `pomelo_delivery` Codex MCP server. Its local stdio command is `uv --directory mcp/src/delivery-mcp run pomelo-delivery-mcp`; the exposed control-plane tools retain their `orbit_*` names.
- Treat a request to deploy RAGFlow with Pomelo Orbit as the integrated deployment by default. Use `scripts/ragflow-bundled/docker-compose.yml` and the `deploy-ragflow-integrated-orbit` skill.
- Before any `pomelo_delivery` write, read the skill and inspect the current MCP tool definitions. Do not infer parameters from historical documents or implementation details.
- The integrated topology is exactly one `ragflow` Application, two six-component Versions (`ragflow-integrated-cpu` and `ragflow-integrated-gpu`), and one `default` Service. Each Version contains `mysql`, `redis`, `minio`, `es01`, `tei`, and `ragflow-cpu`; only the TEI image and GPU device request differ.
- Use `scripts/ragflow-split/` and `deploy-ragflow-split-orbit` only when the user explicitly requests split deployment. It owns a separate `ragflow-split` Application with two Components in each Version (`ragflow-split-cpu` and `ragflow-split-gpu`), plus independent `ragflow-mysql`, `ragflow-redis`, `ragflow-minio`, and `ragflow-elasticsearch` Applications. Never share or mount data directories between the two RAGFlow Applications.
- Before either topology is deployed, initialize both RAGFlow Applications and all four split backing Applications with their Versions and stopped Services. The invoked skill deploys only its own topology: integrated starts `ragflow/default`; split starts the backing Services and `ragflow-split/default`.
- Both RAGFlow Services use local port `9380`; only one may run. Before starting the selected topology, stop the other RAGFlow Service through Orbit without removing volumes.
- Before an Orbit lifecycle write, detect NVIDIA hardware and run `scripts/prepare_ragflow_tei.py check --profile cpu` or `--profile gpu` for the selected topology and profile. Select GPU only when NVIDIA is detected and the GPU preflight passes; otherwise select CPU. A detected GPU whose preflight fails blocks deployment rather than silently falling back to CPU.
- `docs/archive/` is historical context, never operational authority. Do not use its workflows, `orbit_bootstrap_application`, or the removed `scripts/ragflow_initialize.py`.
- Perform deployment lifecycle writes only through `pomelo_delivery` MCP. Do not use Docker Compose lifecycle commands, and do not print runtime configuration or credentials.
