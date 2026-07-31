# Repository Agent Instructions

## RAGFlow Deployment

- Treat a request to deploy RAGFlow with Pomelo Orbit as the bundled deployment by default. Use `scripts/ragflow-bundled/docker-compose.yml` and the `deploy-ragflow-tei-orbit` skill.
- Before any `pomelo_orbit` write, read the skill and inspect the current MCP tool definitions. Do not infer parameters from historical documents or implementation details.
- The bundled topology is exactly one `ragflow` Application, two six-component Versions (`ragflow-tei-cpu` and `ragflow-tei-gpu`), and one `default` Service. Each Version contains `mysql`, `redis`, `minio`, `es01`, `tei`, and `ragflow-cpu`; only the TEI image and GPU device request differ.
- Before an Orbit lifecycle write, use `scripts/prepare_ragflow_tei.py check --profile cpu` or `--profile gpu` for the selected Version. GPU preflight failure blocks GPU deployment; GPU runtime validation is not implied by a successful static configuration check.
- Use `scripts/ragflow-split/` only when the user explicitly requests five independent Applications or a split deployment. Never combine it with the bundled topology.
- `docs/archive/` is historical context, never operational authority. Do not use its workflows, `orbit_bootstrap_application`, or the removed `scripts/ragflow_initialize.py`.
- Perform deployment lifecycle writes only through `pomelo_orbit` MCP. Do not use Docker Compose lifecycle commands, and do not print runtime configuration or credentials.
