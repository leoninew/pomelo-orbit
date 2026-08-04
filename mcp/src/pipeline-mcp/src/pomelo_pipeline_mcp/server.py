"""Local stdio entrypoint for read-only Pomelo CI pipeline queries."""

from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Any

from mcp.server.fastmcp import FastMCP
from mcp.server.fastmcp.exceptions import ToolError

from .client import PipelineClient
from .mcp_errors import tool_error_result
from .settings import Settings
from .tools.pipeline import register_pipeline_tools


def source_schema_fingerprint() -> str:
    """Identify the registered source without exposing local paths or settings."""
    source_root = Path(__file__).resolve().parent
    digest = hashlib.sha256()
    for source_path in sorted(source_root.rglob("*.py")):
        digest.update(source_path.relative_to(source_root).as_posix().encode("utf-8"))
        digest.update(b"\0")
        digest.update(source_path.read_bytes())
        digest.update(b"\0")
    return f"sha256:{digest.hexdigest()[:16]}"


class PipelineMCP(FastMCP):
    """FastMCP server with one safe error boundary for every read-only tool."""

    async def call_tool(self, name: str, arguments: dict[str, Any]) -> Any:
        try:
            return await super().call_tool(name, arguments)
        except ToolError as error:
            return tool_error_result(error)
        except Exception as error:
            return tool_error_result(error)


def create_server(settings: Settings | None = None, client: PipelineClient | None = None) -> PipelineMCP:
    """Build the independent stdio server without lifecycle or write tools."""
    settings = settings or Settings.load()
    client = client or PipelineClient(settings)
    server = PipelineMCP(
        "Pomelo Pipeline Queries",
        instructions=(
            "This server only reads CI pipeline data through fixed authenticated GET endpoints. "
            "It cannot trigger, cancel, retry, deploy, stop, or modify resources. Scripts, variable values, "
            "physical artifact paths, and common credential patterns in logs are omitted or redacted. "
            f"Source/schema fingerprint: {source_schema_fingerprint()}. Restart the stdio MCP session after source changes."
        ),
    )
    register_pipeline_tools(server, client)
    return server


def main() -> None:
    create_server().run(transport="stdio")


if __name__ == "__main__":
    main()
