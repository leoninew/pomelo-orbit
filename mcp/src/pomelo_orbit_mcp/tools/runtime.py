"""Managed, read-only Docker runtime MCP tools."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..docker_runtime import DockerRuntime
from ..orbit_client import OrbitClient
from ..settings import Settings
from ..verification import resolve_runtime_target
from ..workspace import RuntimeTarget
from .common import runtime_result


async def _target(
    client: OrbitClient,
    settings: Settings,
    application_id: str,
    instance_key: str,
) -> RuntimeTarget:
    return await resolve_runtime_target(client, settings, application_id, instance_key)


def register_runtime_tools(mcp: FastMCP, client: OrbitClient, runtime: DockerRuntime, settings: Settings) -> None:
    @mcp.tool(name="runtime_doctor")
    async def runtime_doctor(
        application_id: str | None = None,
        instance_key: str | None = None,
    ) -> dict[str, Any]:
        """Check Docker context, Compose availability, data root, and optionally a managed workspace."""
        supplied = [application_id, instance_key]
        if any(value is not None for value in supplied) and not all(supplied):
            raise ValueError("application_id and instance_key must be supplied together")
        target = None
        if application_id and instance_key:
            target = await _target(client, settings, application_id, instance_key)
        result = await runtime.doctor(target)
        return runtime_result(target, result) if target else result

    @mcp.tool(name="runtime_compose_config")
    async def runtime_compose_config(application_id: str, instance_key: str = "default") -> dict[str, Any]:
        """Read rendered Docker Compose configuration for one managed runtime target."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.compose_config(target))

    @mcp.tool(name="runtime_compose_ps")
    async def runtime_compose_ps(application_id: str, instance_key: str = "default") -> dict[str, Any]:
        """Read Docker Compose container state for one managed runtime target."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.compose_ps(target))

    @mcp.tool(name="runtime_compose_logs")
    async def runtime_compose_logs(
        application_id: str,
        instance_key: str = "default",
        tail: int = 200,
        since: str | None = None,
        services: list[str] | None = None,
    ) -> dict[str, Any]:
        """Read Compose logs for services derived from one managed runtime target."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.compose_logs(target, tail, since, services))

    @mcp.tool(name="runtime_container_inspect")
    async def runtime_container_inspect(
        application_id: str,
        container_id: str,
        instance_key: str = "default",
    ) -> dict[str, Any]:
        """Inspect a container only when its ID was returned by this target's Compose ps output."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.container_inspect(target, container_id))

    @mcp.tool(name="runtime_network_inspect")
    async def runtime_network_inspect(
        application_id: str,
        network_name: str,
        instance_key: str = "default",
    ) -> dict[str, Any]:
        """Inspect a network only when it is derived from a target-managed container inspect result."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.network_inspect(target, network_name))
