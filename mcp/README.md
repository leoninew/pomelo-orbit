# Pomelo MCP Projects

`mcp/src/` contains independently packaged local MCP projects:

- `delivery-mcp/`: the runnable `pomelo-delivery-mcp` delivery-control server. See [its README](src/delivery-mcp/README.md).
- `pipeline-mcp/`: the runnable, read-only `pomelo-pipeline-mcp` CI-query server. It has no Codex registration and no CI or delivery write tools; see [its README](src/pipeline-mcp/README.md).

Do not add compatibility entry points at `mcp/`. Register and run each MCP from its own project directory.
