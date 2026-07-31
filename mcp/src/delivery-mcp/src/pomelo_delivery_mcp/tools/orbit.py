"""Orbit HTTP control-plane MCP tools."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..docker_runtime import DockerRuntime, DockerRuntimeError
from ..orbit_client import OrbitClient
from ..version_specs import (
    ServiceExpose,
    VersionComponent,
    VersionComponentAdvancedUpdate,
    VersionComponentBasicUpdate,
    VersionComponentCreate,
    VersionComponentDependenciesUpdate,
    VersionComponentDevicesUpdate,
    VersionComponentEnvUpdate,
    VersionComponentMountsUpdate,
    VersionComponentPortsUpdate,
    VersionComponentResourcesUpdate,
    VersionComponentRuntimeUpdate,
    VersionComponentTmpfsUpdate,
    VersionComponentUlimitsUpdate,
    service_expose_payload,
    version_component_create_payload,
    version_component_payload,
)
from .common import compact, write_result

TRAEFIK_GATEWAY_CODE = "traefik"
TRAEFIK_NETWORK_NAME = "traefik"


async def _update_component_advanced_section(
    client: OrbitClient, version_id: str, component_id: str, field: str, value: Any
) -> tuple[dict[str, Any], dict[str, Any]]:
    component = await client.get_version_component(version_id, component_id)
    body: dict[str, Any] = {
        "resources": component.get("resources"),
        "tmpfs": component.get("tmpfs") or [],
        "ulimits": component.get("ulimits") or [],
    }
    body[field] = value
    updated = await client.update_version_component_advanced(version_id, component_id, body)
    return body, updated


def register_orbit_tools(mcp: FastMCP, client: OrbitClient, runtime: DockerRuntime) -> None:
    @mcp.tool(name="orbit_list_projects")
    async def orbit_list_projects() -> dict[str, Any]:
        """List Projects visible to the configured Orbit user."""
        return {"projects": await client.list_projects()}

    @mcp.tool(name="orbit_list_applications")
    async def orbit_list_applications(project_id: str, kind: str | None = "standard") -> dict[str, Any]:
        """List Orbit Applications in a Project, optionally limited to one application kind."""
        return {
            "project_id": project_id,
            "kind": kind,
            "applications": await client.list_applications(project_id, kind),
        }

    @mcp.tool(name="orbit_list_application_services")
    async def orbit_list_application_services(application_id: str) -> dict[str, Any]:
        """List non-sensitive Service summaries for an Orbit Application."""
        services = await client.list_application_services(application_id)
        return application_service_list_result(application_id, services)

    @mcp.tool(name="orbit_list_gateways")
    async def orbit_list_gateways(project_id: str) -> dict[str, Any]:
        """List Gateway metadata in a Project through Orbit."""
        return {"project_id": project_id, "gateways": await client.list_gateways(project_id)}

    @mcp.tool(name="orbit_create_gateway")
    async def orbit_create_gateway(
        project_id: str,
        code: str = "traefik",
        name: str = "Traefik",
        rest_api_url: str = "http://localhost:8080",
        base_domain: str = "lvh.me",
        image: str = "traefik:3.6",
        image_pull_policy: str = "missing",
        default_entrypoint: str | None = None,
        tls_mode: str | None = None,
    ) -> dict[str, Any]:
        """Create an Orbit-managed Gateway; deploy it with the existing deployment tools."""
        body = compact(
            {
                "project_id": project_id,
                "code": code,
                "name": name,
                "rest_api_url": rest_api_url,
                "base_domain": base_domain,
                "image": image,
                "image_pull_policy": image_pull_policy,
                "default_entrypoint": default_entrypoint,
                "tls_mode": tls_mode,
            }
        )
        gateway = await client.create_gateway(project_id, body)
        gateway_id = str(gateway.get("id") or "")
        if not gateway_id:
            raise ValueError("Orbit gateway create response did not contain an id")
        return write_result(
            "create_gateway",
            {"gateway_id": gateway_id, "application_id": gateway_id},
            "POST",
            "/api/gateway",
            request_body=body,
            steps=["Created Gateway Application through Orbit"],
            data={"gateway": gateway},
        )

    @mcp.tool(name="orbit_provision_gateway")
    async def orbit_provision_gateway(
        project_id: str,
        instance_key: str = "default",
        runtime_config: dict[str, str] | None = None,
        force_recreate: bool = False,
        timeout_seconds: int | None = None,
    ) -> dict[str, Any]:
        """Ensure one managed traefik Gateway is deployed and its external network is ready."""
        return await provision_gateway(
            client,
            runtime,
            project_id=project_id,
            instance_key=instance_key,
            runtime_config=runtime_config,
            force_recreate=force_recreate,
            timeout_seconds=timeout_seconds,
        )

    @mcp.tool(name="orbit_get_gateway")
    async def orbit_get_gateway(gateway_id: str) -> dict[str, Any]:
        """Read one Gateway and its Application metadata through Orbit."""
        return {"gateway": await client.get_gateway(gateway_id)}

    @mcp.tool(name="orbit_update_gateway")
    async def orbit_update_gateway(
        gateway_id: str,
        name: str | None = None,
        rest_api_url: str | None = None,
        base_domain: str | None = None,
        image: str | None = None,
        image_pull_policy: str | None = None,
        default_entrypoint: str | None = None,
        tls_mode: str | None = None,
    ) -> dict[str, Any]:
        """Update an Orbit-managed Gateway configuration through Orbit."""
        body = compact(
            {
                "name": name,
                "rest_api_url": rest_api_url,
                "base_domain": base_domain,
                "image": image,
                "image_pull_policy": image_pull_policy,
                "default_entrypoint": default_entrypoint,
                "tls_mode": tls_mode,
            }
        )
        if not body:
            raise ValueError("at least one Gateway field must be supplied")
        gateway = await client.update_gateway(gateway_id, body)
        return write_result(
            "update_gateway",
            {"gateway_id": gateway_id, "application_id": gateway_id},
            "PUT",
            f"/api/gateway/{gateway_id}",
            request_body=body,
            steps=["Updated Gateway Application through Orbit"],
            data={"gateway": gateway},
        )

    @mcp.tool(name="orbit_create_application")
    async def orbit_create_application(
        project_id: str,
        name: str,
        code: str,
        image_pull_policy: str = "missing",
        kind: str = "standard",
    ) -> dict[str, Any]:
        """Create a standard Application through Orbit and return its initial draft Version."""
        if kind != "standard":
            raise ValueError("MCP creation only supports kind=standard")
        body = {"name": name, "code": code, "image_pull_policy": image_pull_policy, "kind": kind}
        application = await client.create_application(project_id, body)
        versions = await client.list_versions(str(application["id"]))
        initial = next((item for item in versions if item.get("status") == "unpublished"), None)
        return write_result(
            "create_application",
            compact(
                {
                    "application_id": str(application["id"]),
                    "initial_version_id": str(initial["id"]) if initial else None,
                }
            ),
            "POST",
            "/api/application",
            request_body=body,
            steps=["Created Application through Orbit", "Read initial draft Version through Orbit"],
            data={"application": application, "initial_version": initial},
        )

    @mcp.tool(name="orbit_get_application")
    async def orbit_get_application(application_id: str) -> dict[str, Any]:
        """Read one Orbit Application, including its current service summary."""
        return {"application": await client.get_application(application_id)}

    @mcp.tool(name="orbit_delete_application")
    async def orbit_delete_application(application_id: str, remove_dir: bool = False) -> dict[str, Any]:
        """Delete an Orbit Application and optionally remove its managed deployment directory."""
        await client.delete_application(application_id, remove_dir)
        return write_result(
            "delete_application",
            {"application_id": application_id},
            "DELETE",
            f"/api/application/{application_id}",
            request_body={"remove_dir": remove_dir},
        )

    @mcp.tool(name="orbit_list_versions")
    async def orbit_list_versions(application_id: str) -> dict[str, Any]:
        """List Versions for an Orbit Application."""
        return {"application_id": application_id, "versions": await client.list_versions(application_id)}

    @mcp.tool(name="orbit_get_version")
    async def orbit_get_version(version_id: str) -> dict[str, Any]:
        """Read a Version with its Components."""
        return {"version": await client.get_version(version_id)}

    @mcp.tool(name="orbit_create_version_component")
    async def orbit_create_version_component(version_id: str, component: VersionComponentCreate) -> dict[str, Any]:
        """Add a Component with basic configuration to an unpublished Version."""
        body = version_component_create_payload(component)
        created = await client.create_version_component(version_id, body)
        return write_result(
            "create_version_component",
            {"version_id": version_id, "component_id": str(created["id"])},
            "POST",
            f"/api/version/{version_id}/component",
            request_body=body,
            data={"component": created},
        )

    @mcp.tool(name="orbit_create_version")
    async def orbit_create_version(
        application_id: str,
        label: str,
        components: list[VersionComponent],
        note: str | None = None,
    ) -> dict[str, Any]:
        """Create a Version using a complete Component collection."""
        body = compact(
            {
                "label": label,
                "components": [version_component_payload(component) for component in components],
                "note": note,
            }
        )
        version = await client.create_version(application_id, body)
        return write_result(
            "create_version",
            {"application_id": application_id, "version_id": str(version["id"])},
            "POST",
            f"/api/application/{application_id}/version",
            request_body=body,
            data={"version": version},
        )

    @mcp.tool(name="orbit_update_version")
    async def orbit_update_version(
        version_id: str,
        label: str | None = None,
        note: str | None = None,
    ) -> dict[str, Any]:
        """Update Version metadata; Components have dedicated tools."""
        body = compact(
            {
                "label": label,
                "note": note,
            }
        )
        if not body:
            raise ValueError("at least one Version field must be supplied")
        version = await client.update_version(version_id, body)
        return write_result(
            "update_version",
            {"version_id": version_id},
            "PUT",
            f"/api/version/{version_id}",
            request_body=body,
            data={"version": version},
        )

    @mcp.tool(name="orbit_update_version_component_basic")
    async def orbit_update_version_component_basic(
        version_id: str, component_id: str, basic: VersionComponentBasicUpdate
    ) -> dict[str, Any]:
        """Replace a Component's name, image, command, pull policy, and restart policy."""
        body = basic.model_dump(exclude_none=True)
        updated = await client.update_version_component_basic(version_id, component_id, body)
        return write_result(
            "update_version_component_basic",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/basic",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_runtime")
    async def orbit_update_version_component_runtime(
        version_id: str, component_id: str, runtime: VersionComponentRuntimeUpdate
    ) -> dict[str, Any]:
        """Replace a Component's health check."""
        body = runtime.model_dump(exclude_none=True)
        updated = await client.update_version_component_runtime(version_id, component_id, body)
        return write_result(
            "update_version_component_runtime",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/runtime",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_ports")
    async def orbit_update_version_component_ports(
        version_id: str, component_id: str, ports: VersionComponentPortsUpdate
    ) -> dict[str, Any]:
        """Replace a Component's port collection."""
        body = ports.model_dump(exclude_none=True)
        updated = await client.update_version_component_ports(version_id, component_id, body)
        return write_result(
            "update_version_component_ports",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/ports",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_env")
    async def orbit_update_version_component_env(
        version_id: str, component_id: str, env: VersionComponentEnvUpdate
    ) -> dict[str, Any]:
        """Replace a Component's environment collection."""
        body = env.model_dump(exclude_none=True)
        updated = await client.update_version_component_env(version_id, component_id, body)
        return write_result(
            "update_version_component_env",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/env",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_mounts")
    async def orbit_update_version_component_mounts(
        version_id: str, component_id: str, mounts: VersionComponentMountsUpdate
    ) -> dict[str, Any]:
        """Replace a Component's mount collection."""
        body = mounts.model_dump(exclude_none=True)
        updated = await client.update_version_component_mounts(version_id, component_id, body)
        return write_result(
            "update_version_component_mounts",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/mounts",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_dependencies")
    async def orbit_update_version_component_dependencies(
        version_id: str, component_id: str, dependencies: VersionComponentDependenciesUpdate
    ) -> dict[str, Any]:
        """Replace a Component's dependency collection."""
        body = dependencies.model_dump(exclude_none=True)
        updated = await client.update_version_component_dependencies(version_id, component_id, body)
        return write_result(
            "update_version_component_dependencies",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/dependencies",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_devices")
    async def orbit_update_version_component_devices(
        version_id: str, component_id: str, devices: VersionComponentDevicesUpdate
    ) -> dict[str, Any]:
        """Replace a Component's device requests without changing resources, tmpfs, or ulimits."""
        body = devices.model_dump(exclude_none=True)
        updated = await client.update_version_component_devices(version_id, component_id, body)
        return write_result(
            "update_version_component_devices",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/devices",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_advanced")
    async def orbit_update_version_component_advanced(
        version_id: str, component_id: str, advanced: VersionComponentAdvancedUpdate
    ) -> dict[str, Any]:
        """Replace a Component's resources, tmpfs, and ulimit settings."""
        body = advanced.model_dump(exclude_none=True)
        updated = await client.update_version_component_advanced(version_id, component_id, body)
        return write_result(
            "update_version_component_advanced",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/advanced",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_resources")
    async def orbit_update_version_component_resources(
        version_id: str, component_id: str, resources: VersionComponentResourcesUpdate
    ) -> dict[str, Any]:
        """Replace a Component's resource constraints and preserve its tmpfs and ulimit settings."""
        value = resources.resources.model_dump(exclude_none=True) if resources.resources else None
        body, updated = await _update_component_advanced_section(client, version_id, component_id, "resources", value)
        return write_result(
            "update_version_component_resources",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/advanced",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_tmpfs")
    async def orbit_update_version_component_tmpfs(
        version_id: str, component_id: str, tmpfs: VersionComponentTmpfsUpdate
    ) -> dict[str, Any]:
        """Replace a Component's tmpfs collection and preserve its resource and ulimit settings."""
        value = [item.model_dump() for item in tmpfs.tmpfs]
        body, updated = await _update_component_advanced_section(client, version_id, component_id, "tmpfs", value)
        return write_result(
            "update_version_component_tmpfs",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/advanced",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_update_version_component_ulimits")
    async def orbit_update_version_component_ulimits(
        version_id: str, component_id: str, ulimits: VersionComponentUlimitsUpdate
    ) -> dict[str, Any]:
        """Replace a Component's ulimit collection and preserve its resource and tmpfs settings."""
        value = [item.model_dump() for item in ulimits.ulimits]
        body, updated = await _update_component_advanced_section(client, version_id, component_id, "ulimits", value)
        return write_result(
            "update_version_component_ulimits",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}/advanced",
            request_body=body,
            data={"component": updated},
        )

    @mcp.tool(name="orbit_publish_version")
    async def orbit_publish_version(version_id: str) -> dict[str, Any]:
        """Mark a Version published through Orbit; publication does not lock later edits or deletion."""
        version = await client.publish_version(version_id)
        return write_result(
            "publish_version",
            {"version_id": version_id},
            "POST",
            f"/api/version/{version_id}/publish",
            request_body={},
            data={"version": version},
        )

    @mcp.tool(name="orbit_delete_version")
    async def orbit_delete_version(version_id: str) -> dict[str, Any]:
        """Delete an unreferenced Version through Orbit regardless of its published marker."""
        await client.delete_version(version_id)
        return write_result("delete_version", {"version_id": version_id}, "DELETE", f"/api/version/{version_id}")

    @mcp.tool(name="orbit_preview_service")
    async def orbit_preview_service(service_id: str) -> dict[str, Any]:
        """Render a saved Service configuration without deploying it."""
        preview = await client.preview_service(service_id)
        return {"service_id": service_id, "preview": preview}

    @mcp.tool(name="orbit_create_service")
    async def orbit_create_service(
        application_id: str,
        version_id: str,
        instance_key: str,
        runtime_config: dict[str, str],
        exposes: list[ServiceExpose],
    ) -> dict[str, Any]:
        """Create a stopped Service with complete runtime and expose configuration."""
        payload_exposes = [service_expose_payload(expose) for expose in exposes]
        service = await client.create_service(application_id, version_id, instance_key, runtime_config, payload_exposes)
        service_id = str(service.get("id") or "")
        if not service_id:
            raise ValueError("Orbit service create response did not contain an id")
        return write_result(
            "create_service",
            {"application_id": application_id, "service_id": service_id},
            "POST",
            "/api/service",
            request_body={
                "application_id": application_id,
                "version_id": version_id,
                "instance_key": instance_key,
                "runtime_config": runtime_config,
                "exposes": payload_exposes,
            },
            data={"service": service},
        )

    @mcp.tool(name="orbit_update_service_configuration")
    async def orbit_update_service_configuration(
        service_id: str,
        runtime_config: dict[str, str],
        exposes: list[ServiceExpose],
    ) -> dict[str, Any]:
        """Replace a Service's runtime configuration and exposes."""
        payload_exposes = [service_expose_payload(expose) for expose in exposes]
        result = await client.update_service_configuration(service_id, runtime_config, payload_exposes)
        return write_result(
            "update_service_configuration",
            {"service_id": service_id},
            "PUT",
            f"/api/service/{service_id}/config",
            request_body={"runtime_config": runtime_config, "exposes": payload_exposes},
            data={"service": result},
        )

    @mcp.tool(name="orbit_update_service_basic")
    async def orbit_update_service_basic(
        service_id: str,
        version_id: str,
        instance_key: str,
    ) -> dict[str, Any]:
        """Replace a Service's selected Version and instance key."""
        body = {"version_id": version_id, "instance_key": instance_key}
        result = await client.update_service_basic(service_id, version_id, instance_key)
        return write_result(
            "update_service_basic",
            {"service_id": service_id},
            "PUT",
            f"/api/service/{service_id}/basic",
            request_body=body,
            data={"service": result},
        )

    @mcp.tool(name="orbit_deploy")
    async def orbit_deploy(
        service_id: str,
        force_recreate: bool = False,
    ) -> dict[str, Any]:
        """Create an Orbit deployment and immediately return its persisted command summary."""
        body = {"force_recreate": force_recreate}
        action = await client.deploy_service(service_id, force_recreate)
        deployment_id = str(action["deployment_id"])
        deployment = await client.get_deployment(deployment_id)
        return write_result(
            "deploy_service",
            {
                "service_id": service_id,
                "deployment_id": deployment_id,
            },
            "POST",
            f"/api/service/{service_id}/deploy",
            request_body=body,
            steps=["Created Deployment through Orbit", "Read persisted Deployment command summary"],
            data={
                "deployment_id": deployment_id,
                "warnings": action.get("warnings") or [],
                "command_text": deployment.get("command_text"),
                "deployment": deployment,
            },
        )

    @mcp.tool(name="orbit_stop")
    async def orbit_stop(
        application_id: str,
        service_id: str,
        remove_volumes: bool = False,
    ) -> dict[str, Any]:
        """Create an Orbit stop deployment, optionally requesting managed volume removal."""
        body = compact(
            {
                "service_id": service_id,
                "remove_volumes": remove_volumes,
            }
        )
        action = await client.stop_application(application_id, body)
        deployment_id = str(action["deployment_id"])
        deployment = await client.get_deployment(deployment_id)
        return write_result(
            "stop_application",
            compact({"application_id": application_id, "service_id": service_id, "deployment_id": deployment_id}),
            "POST",
            f"/api/application/{application_id}/stop",
            request_body=body,
            steps=["Created stop Deployment through Orbit", "Read persisted Deployment command summary"],
            data={
                "deployment_id": deployment_id,
                "command_text": deployment.get("command_text"),
                "deployment": deployment,
            },
        )

    @mcp.tool(name="orbit_restart")
    async def orbit_restart(
        application_id: str,
        service_id: str,
    ) -> dict[str, Any]:
        """Create an Orbit restart deployment and immediately return its command summary."""
        body = {"service_id": service_id}
        action = await client.restart_application(application_id, body)
        deployment_id = str(action["deployment_id"])
        deployment = await client.get_deployment(deployment_id)
        return write_result(
            "restart_application",
            compact({"application_id": application_id, "service_id": service_id, "deployment_id": deployment_id}),
            "POST",
            f"/api/application/{application_id}/restart",
            request_body=body,
            steps=["Created restart Deployment through Orbit", "Read persisted Deployment command summary"],
            data={
                "deployment_id": deployment_id,
                "command_text": deployment.get("command_text"),
                "deployment": deployment,
            },
        )

    @mcp.tool(name="orbit_deployment_status")
    async def orbit_deployment_status(deployment_id: str) -> dict[str, Any]:
        """Read the current Orbit Deployment state and command text."""
        return {"deployment": await client.get_deployment(deployment_id)}

    @mcp.tool(name="orbit_deployment_logs")
    async def orbit_deployment_logs(deployment_id: str, offset: int = 0) -> dict[str, Any]:
        """Read Orbit worker logs for a Deployment from an incremental offset."""
        return {"deployment_id": deployment_id, "logs": await client.get_deployment_logs(deployment_id, offset)}

    @mcp.tool(name="orbit_wait_deployment")
    async def orbit_wait_deployment(deployment_id: str, timeout_seconds: int | None = None) -> dict[str, Any]:
        """Wait only until an Orbit Deployment reaches a terminal state or the configured timeout."""
        result = await client.wait_deployment(deployment_id, timeout_seconds)
        return {"operation": "wait_deployment", "deployment_id": deployment_id, **result}


async def provision_gateway(
    client: OrbitClient,
    runtime: DockerRuntime,
    *,
    project_id: str,
    instance_key: str,
    runtime_config: dict[str, str] | None,
    force_recreate: bool,
    timeout_seconds: int | None,
) -> dict[str, Any]:
    """Run the explicit Gateway provisioning workflow without nesting MCP tool calls."""
    normalized_project_id = project_id.strip()
    normalized_instance_key = instance_key.strip()
    if not normalized_project_id:
        raise ValueError("project_id is required")
    if not normalized_instance_key:
        raise ValueError("instance_key is required")
    supplied_runtime_config = dict(runtime_config or {})

    gateway_matches = [
        gateway
        for gateway in await client.list_gateways(normalized_project_id)
        if str(gateway.get("code") or "") == TRAEFIK_GATEWAY_CODE
    ]
    if len(gateway_matches) > 1:
        raise ValueError("multiple traefik Gateways exist in the Project")

    gateway_created = not gateway_matches
    if gateway_created:
        gateway = await client.create_gateway(normalized_project_id, _default_gateway_payload(normalized_project_id))
    else:
        gateway = gateway_matches[0]
    gateway_id = _resource_id(gateway, "Gateway")
    resource_ids = {"gateway_id": gateway_id, "application_id": gateway_id}
    steps = ["Created Gateway Application through Orbit" if gateway_created else "Reused Gateway Application"]

    versions = await client.list_versions(gateway_id)
    services = await client.list_application_services(gateway_id)
    service_matches = [
        service for service in services if str(service.get("instance_key") or "") == normalized_instance_key
    ]
    if len(service_matches) > 1:
        raise ValueError("multiple Gateway Services use the requested instance_key")
    service = service_matches[0] if service_matches else None
    if service is not None and supplied_runtime_config:
        raise ValueError("runtime_config cannot be supplied when reusing an existing Gateway Service")

    version = _select_gateway_version(versions, service)
    version_id = _resource_id(version, "Gateway Version")
    version_published = str(version.get("status") or "") == "published"
    published_now = False
    if not version_published:
        await client.publish_version(version_id)
        version_published = True
        published_now = True
        steps.append("Published Gateway Version")
    resource_ids["version_id"] = version_id

    service_created = service is None
    if service is None:
        service_result = await client.create_service(
            gateway_id, version_id, normalized_instance_key, supplied_runtime_config, []
        )
        steps.append("Created Gateway Service")
    else:
        service_result = await client.get_service(_resource_id(service, "Gateway Service"))
        if str(service_result.get("version_id") or "") != version_id:
            await client.update_service_basic(
                _resource_id(service_result, "Gateway Service"),
                version_id,
                str(service_result.get("instance_key") or ""),
            )
        steps.append("Reused Gateway Service")
    service_id = _resource_id(service_result, "Gateway Service")
    resource_ids["service_id"] = service_id

    deployment = await client.deploy_service(service_id, force_recreate)
    deployment_id = _resource_id(deployment, "Gateway Deployment", field="deployment_id")
    resource_ids["deployment_id"] = deployment_id
    steps.append("Created Gateway Deployment through Orbit")
    waited = await client.wait_deployment(deployment_id, timeout_seconds)
    deployment_status = str(_mapping(waited.get("deployment")).get("status") or "")
    if waited.get("timed_out") or deployment_status != "ran_to_completion":
        return _provision_gateway_result(
            resource_ids,
            steps + ["Gateway Deployment did not reach a successful terminal state"],
            gateway_created=gateway_created,
            service_created=service_created,
            version_published=published_now,
            ready=False,
            network={"name": TRAEFIK_NETWORK_NAME, "ready": False, "status": "not_checked"},
        )

    steps.append("Gateway Deployment reached a successful terminal state")
    try:
        network_detail = await runtime.external_network_inspect(TRAEFIK_NETWORK_NAME)
    except DockerRuntimeError:
        return _provision_gateway_result(
            resource_ids,
            steps + ["External traefik network is unavailable"],
            gateway_created=gateway_created,
            service_created=service_created,
            version_published=published_now,
            ready=False,
            network={"name": TRAEFIK_NETWORK_NAME, "ready": False, "status": "unavailable"},
        )
    network_ready = network_detail["driver"] == "bridge"
    network = {**network_detail, "ready": network_ready, "status": "ready" if network_ready else "unsupported_driver"}
    final_steps = steps + (
        ["Confirmed external traefik bridge network"]
        if network_ready
        else ["External traefik network is not a bridge network"]
    )
    return _provision_gateway_result(
        resource_ids,
        final_steps,
        gateway_created=gateway_created,
        service_created=service_created,
        version_published=published_now,
        ready=network_ready,
        network=network,
    )


def _default_gateway_payload(project_id: str) -> dict[str, str]:
    return {
        "project_id": project_id,
        "code": TRAEFIK_GATEWAY_CODE,
        "name": "Traefik",
        "rest_api_url": "http://localhost:8080",
        "base_domain": "lvh.me",
        "image": "traefik:3.6",
        "image_pull_policy": "missing",
    }


def _select_gateway_version(versions: list[dict[str, Any]], service: dict[str, Any] | None) -> dict[str, Any]:
    unpublished = [version for version in versions if str(version.get("status") or "") == "unpublished"]
    if len(unpublished) == 1:
        return unpublished[0]
    if len(unpublished) > 1:
        raise ValueError("multiple unpublished Gateway Versions exist")
    if service is not None:
        service_version_id = str(service.get("version_id") or "")
        matches = [version for version in versions if str(version.get("id") or "") == service_version_id]
        if len(matches) == 1:
            return matches[0]
        raise ValueError("existing Gateway Service does not reference a listed Version")
    if len(versions) == 1:
        return versions[0]
    raise ValueError("unable to select a unique Gateway Version")


def _resource_id(resource: dict[str, Any], resource_name: str, *, field: str = "id") -> str:
    resource_id = str(resource.get(field) or "").strip()
    if not resource_id:
        raise ValueError(f"Orbit {resource_name} response did not contain {field}")
    return resource_id


def _provision_gateway_result(
    resource_ids: dict[str, str],
    steps: list[str],
    *,
    gateway_created: bool,
    service_created: bool,
    version_published: bool,
    ready: bool,
    network: dict[str, Any],
) -> dict[str, Any]:
    return {
        "operation": "provision_gateway",
        "ready": ready,
        "resource_ids": resource_ids,
        "created": {
            "gateway": gateway_created,
            "service": service_created,
            "version_published": version_published,
        },
        "network": network,
        "steps": steps,
    }


def _mapping(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def application_service_list_result(application_id: str, services: list[dict[str, Any]]) -> dict[str, Any]:
    """Expose only Service fields needed for lifecycle actions."""
    return {
        "application_id": application_id,
        "services": [service_summary(service) for service in services],
    }


def service_summary(service: dict[str, Any]) -> dict[str, Any]:
    allowed = (
        "id",
        "application_id",
        "instance_key",
        "status",
        "version_id",
        "created_at",
        "updated_at",
    )
    return {field: service[field] for field in allowed if service.get(field) is not None}
