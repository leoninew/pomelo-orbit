"""Deployment verification MCP tool."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..docker_runtime import DockerRuntime
from ..orbit_client import OrbitClient
from ..settings import Settings
from ..verification import verify_deployment


def register_verification_tools(mcp: FastMCP, client: OrbitClient, runtime: DockerRuntime, settings: Settings) -> None:
    @mcp.tool(name="verify_deployment")
    async def verify_deployment_tool(application_id: str, deployment_id: str, detail: bool = False) -> dict[str, Any]:
        """Verify deployment state with a concise result; set detail=true for raw diagnostic evidence."""
        result = await verify_deployment(client, runtime, settings, application_id, deployment_id)
        return {"application_id": application_id, "deployment_id": deployment_id, **result.as_dict(detail=detail)}
