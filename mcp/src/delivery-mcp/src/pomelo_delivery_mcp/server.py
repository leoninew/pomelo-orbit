"""Local stdio MCP server entrypoint."""

from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Any

from mcp.server.fastmcp import FastMCP
from mcp.server.fastmcp.exceptions import ToolError

from .docker_runtime import DockerRuntime
from .mcp_errors import tool_error_result
from .orbit_client import OrbitClient
from .settings import Settings
from .tools.orbit import register_orbit_tools
from .tools.runtime import register_runtime_tools
from .tools.verification import register_verification_tools


def source_schema_fingerprint() -> str:
    """Identify the registered MCP source without exposing local paths or configuration."""
    source_root = Path(__file__).resolve().parent
    digest = hashlib.sha256()
    for source_path in sorted(source_root.rglob("*.py")):
        digest.update(source_path.relative_to(source_root).as_posix().encode("utf-8"))
        digest.update(b"\0")
        digest.update(source_path.read_bytes())
        digest.update(b"\0")
    return f"sha256:{digest.hexdigest()[:16]}"


class OrbitMCP(FastMCP):
    """FastMCP server with one safe error boundary for every tool call."""

    async def call_tool(self, name: str, arguments: dict[str, Any]) -> Any:
        try:
            return await super().call_tool(name, arguments)
        except ToolError as error:
            return tool_error_result(error)
        except Exception as error:
            return tool_error_result(error)


def create_server(
    settings: Settings | None = None,
    client: OrbitClient | None = None,
    runtime: DockerRuntime | None = None,
) -> OrbitMCP:
    """Build the stdio server with no transport other than local MCP stdio."""
    settings = settings or Settings.load()
    client = client or OrbitClient(settings)
    runtime = runtime or DockerRuntime(settings)
    server = OrbitMCP(
        "Pomelo Delivery Control",
        instructions=(
            "Use Orbit tools for every lifecycle write. Runtime tools are read-only and only work for "
            "Orbit-managed deployment targets. Authentication material is never returned. "
            f"Source/schema fingerprint: {source_schema_fingerprint()}. Restart the stdio MCP session after "
            "source changes before using write tools."
        ),
    )
    register_orbit_tools(server, client, runtime)
    register_runtime_tools(server, client, runtime, settings)
    register_verification_tools(server, client, runtime, settings)
    return server


def main() -> None:
    create_server().run(transport="stdio")


if __name__ == "__main__":
    main()
