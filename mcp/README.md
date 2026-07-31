# Pomelo MCP Projects

`mcp/src/` contains independently packaged local MCP projects:

- `delivery-mcp/`: the runnable `pomelo-delivery-mcp` delivery-control server. See [its README](src/delivery-mcp/README.md).
- `pipeline-mcp/`: the reserved `pomelo-pipeline-mcp` scaffold. It has no server implementation, executable entry point, or Codex registration.

Do not add compatibility entry points at `mcp/`. Register and run each MCP from its own project directory.
