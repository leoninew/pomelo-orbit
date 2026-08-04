# Pomelo Pipeline MCP

`pomelo-pipeline-mcp` is an independent local stdio MCP server for **read-only** Pomelo CI queries. It is intentionally separate from `pomelo_delivery`: it does not contain delivery lifecycle tools and cannot trigger, cancel, retry, deploy, stop, or modify any resource.

## Current tools

- `pipeline_list_stages`, `pipeline_get_stage`
- `pipeline_list_templates`, `pipeline_get_template`, `pipeline_get_snapshot`
- `pipeline_list_runs`, `pipeline_list_repository_runs`, `pipeline_get_run`
- `pipeline_list_artifacts`, `pipeline_list_run_artifacts`
- `pipeline_get_stage_log`

Stage and template scripts, variable defaults and values, run error text, and physical artifact paths are excluded from responses. Stage logs are redacted for common credential patterns and bounded by `max_bytes`; use the returned `offset` to continue reading.

## Configuration and launch

Copy the variable names from `.env.example` into the process environment. `POMELO_PIPELINE_JWT` must be a short-lived JWT for a non-admin user whose existing project membership grants CI read access. The server never accepts username/password and never stores tokens locally.

```powershell
uv --directory mcp/src/pipeline-mcp run pomelo-pipeline-mcp
```

The server is not registered in `.codex/config.toml` by this change. Register it explicitly in a follow-up only after the configured identity, tool schema, and local operating boundary have been reviewed. Restart the stdio MCP session after source changes.

## Checks

```powershell
uv --directory mcp/src/pipeline-mcp run ruff format --check .
uv --directory mcp/src/pipeline-mcp run ruff check .
uv --directory mcp/src/pipeline-mcp run mypy
uv --directory mcp/src/pipeline-mcp run pytest
```
