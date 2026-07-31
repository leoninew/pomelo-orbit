"""Managed, read-only Docker runtime MCP tools."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..docker_runtime import DockerRuntime, compose_container_summaries
from ..orbit_client import OrbitClient
from ..settings import Settings
from ..verification import resolve_runtime_target
from ..workspace import RuntimeTarget
from .common import runtime_result

TRAEFIK_NETWORK_NAME = "traefik"


async def _target(
    client: OrbitClient,
    settings: Settings,
    application_id: str,
    instance_key: str,
) -> RuntimeTarget:
    return await resolve_runtime_target(client, settings, application_id, instance_key)


async def _gateway_target(
    client: OrbitClient,
    settings: Settings,
    application_id: str,
    instance_key: str,
) -> RuntimeTarget:
    application = await client.get_application(application_id)
    if application.get("kind") != "gateway":
        raise ValueError("gateway_application_id must identify an Orbit Gateway Application")
    return await resolve_runtime_target(
        client,
        settings,
        application_id,
        instance_key,
        allowed_application_kinds=("gateway",),
    )


def register_runtime_tools(mcp: FastMCP, client: OrbitClient, runtime: DockerRuntime, settings: Settings) -> None:
    @mcp.tool(name="runtime_doctor")
    async def runtime_doctor(
        application_id: str | None = None,
        instance_key: str | None = None,
        gateway_application_id: str | None = None,
        gateway_instance_key: str | None = None,
        network_name: str | None = None,
    ) -> dict[str, Any]:
        """Check Docker prerequisites, a managed target, or the fixed external traefik network."""
        supplied = [application_id, instance_key]
        if any(value is not None for value in supplied) and not all(supplied):
            raise ValueError("application_id and instance_key must be supplied together")
        gateway_supplied = [gateway_application_id, gateway_instance_key]
        if any(value is not None for value in gateway_supplied) and not all(gateway_supplied):
            raise ValueError("gateway_application_id and gateway_instance_key must be supplied together")
        if all(supplied) and all(gateway_supplied):
            raise ValueError("application target and gateway target cannot be requested together")
        normalized_network_name = network_name.strip() if network_name is not None else None
        if network_name is not None and not normalized_network_name:
            raise ValueError("network_name is required when supplied")
        if normalized_network_name and normalized_network_name != TRAEFIK_NETWORK_NAME:
            raise ValueError("network_name must be traefik")
        if normalized_network_name and (all(supplied) or all(gateway_supplied)):
            raise ValueError("network_name cannot be combined with an application or gateway target")
        target = None
        expected_external_networks: tuple[str, ...] = ()
        if application_id and instance_key:
            target = await _target(client, settings, application_id, instance_key)
        if gateway_application_id and gateway_instance_key:
            target = await _gateway_target(client, settings, gateway_application_id, gateway_instance_key)
            expected_external_networks = (TRAEFIK_NETWORK_NAME,)
        result = await runtime.doctor(
            target,
            expected_external_networks=expected_external_networks,
            external_network_name=normalized_network_name,
        )
        return runtime_result(target, result) if target else result

    @mcp.tool(name="runtime_compose_config")
    async def runtime_compose_config(application_id: str, instance_key: str = "default") -> dict[str, Any]:
        """Read rendered Docker Compose configuration for one managed runtime target."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.compose_config(target))

    @mcp.tool(name="runtime_compose_ps")
    async def runtime_compose_ps(
        application_id: str, instance_key: str = "default", detail: bool = False
    ) -> dict[str, Any]:
        """Read concise Compose container state; set detail=true for raw Compose JSON."""
        target = await _target(client, settings, application_id, instance_key)
        result = await runtime.compose_ps(target)
        if detail:
            return runtime_result(target, result)
        return runtime_result(target, {"containers": compose_container_summaries(result["containers"])})

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

    @mcp.tool(name="runtime_http_probe")
    async def runtime_http_probe(
        application_id: str,
        component_name: str,
        port: int,
        path: str = "/",
        instance_key: str = "default",
    ) -> dict[str, Any]:
        """Run a fixed local HTTP curl probe in one managed Compose component."""
        target = await _target(client, settings, application_id, instance_key)
        return runtime_result(target, await runtime.http_probe(target, component_name, port, path))
