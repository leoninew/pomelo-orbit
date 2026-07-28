# Repository Agent Instructions

## RAGFlow Deployment

- Treat a request to deploy RAGFlow with Pomelo Orbit as the bundled deployment by default. Use `scripts/ragflow-bundled/docker-compose.yml` and the `deploy-ragflow-orbit` skill.
- Before any `pomelo_orbit` write, read the skill and inspect the current MCP tool definitions. Do not infer parameters from historical documents or implementation details.
- The bundled topology is exactly one `ragflow` Application, one multi-component Version, five Components (`mysql`, `redis`, `minio`, `es01`, `ragflow-cpu`), and one `default` Service.
- Use `scripts/ragflow-split/` only when the user explicitly requests five independent Applications or a split deployment. Never combine it with the bundled topology.
- `docs/archive/` is historical context, never operational authority. Do not use its workflows, `orbit_bootstrap_application`, or the removed `scripts/ragflow_initialize.py`.
- Perform deployment lifecycle writes only through `pomelo_orbit` MCP. Do not use Docker Compose lifecycle commands, and do not print runtime configuration or credentials.
