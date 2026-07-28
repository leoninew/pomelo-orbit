"""Orbit HTTP control-plane MCP tools."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..orbit_client import OrbitClient
from ..version_specs import VersionComponent, VersionExpose, version_component_payload, version_expose_payload
from .common import compact, write_result


def register_orbit_tools(mcp: FastMCP, client: OrbitClient) -> None:
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
        """Read a Version with its Components and Exposes."""
        return {"version": await client.get_version(version_id)}

    @mcp.tool(name="orbit_create_version")
    async def orbit_create_version(
        application_id: str,
        label: str,
        components: list[VersionComponent],
        exposes: list[VersionExpose],
        env_json: str | None = None,
        note: str | None = None,
    ) -> dict[str, Any]:
        """Create a Version using complete Component and Expose collections."""
        body = compact(
            {
                "label": label,
                "components": [version_component_payload(component) for component in components],
                "exposes": [version_expose_payload(expose) for expose in exposes],
                "env_json": env_json,
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
        env_json: str | None = None,
        note: str | None = None,
        exposes: list[VersionExpose] | None = None,
    ) -> dict[str, Any]:
        """Update Version metadata or replace its exposes; components have dedicated tools."""
        body = compact(
            {
                "label": label,
                "env_json": env_json,
                "note": note,
                "exposes": [version_expose_payload(expose) for expose in exposes] if exposes is not None else None,
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

    @mcp.tool(name="orbit_update_version_component")
    async def orbit_update_version_component(
        version_id: str, component_id: str, component: VersionComponent
    ) -> dict[str, Any]:
        """Replace one Version component through Orbit's component route."""
        body = version_component_payload(component)
        updated = await client.update_version_component(version_id, component_id, body)
        return write_result(
            "update_version_component",
            {"version_id": version_id, "component_id": component_id},
            "PUT",
            f"/api/version/{version_id}/component/{component_id}",
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

    @mcp.tool(name="orbit_preview_version")
    async def orbit_preview_version(version_id: str, instance_key: str = "default") -> dict[str, Any]:
        """Render a Version's expected Compose document through Orbit without deploying it."""
        preview = await client.preview_version(version_id, instance_key)
        return {
            "version_id": version_id,
            "instance_key": instance_key,
            "preview": preview,
        }

    @mcp.tool(name="orbit_create_service")
    async def orbit_create_service(
        application_id: str,
        version_id: str,
        instance_key: str,
        runtime_config: dict[str, str],
    ) -> dict[str, Any]:
        """Create a stopped Service with its plain runtime K/V configuration."""
        service = await client.create_service(application_id, version_id, instance_key, runtime_config)
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
            },
            data={"service": service},
        )

    @mcp.tool(name="orbit_get_service_runtime_config")
    async def orbit_get_service_runtime_config(service_id: str) -> dict[str, Any]:
        """Read the saved Service runtime configuration, not container process state."""
        return {"runtime_config": await client.get_service_runtime_config(service_id)}

    @mcp.tool(name="orbit_update_service_runtime_config")
    async def orbit_update_service_runtime_config(service_id: str, runtime_config: dict[str, str]) -> dict[str, Any]:
        """Replace a Service runtime configuration; it takes effect on a later deploy or restart."""
        result = await client.update_service_runtime_config(service_id, runtime_config)
        return write_result(
            "update_service_runtime_config",
            {"service_id": service_id},
            "PUT",
            f"/api/service/{service_id}/runtime-config",
            request_body={"runtime_config": runtime_config},
            data={"runtime_config": result},
        )

    @mcp.tool(name="orbit_deploy")
    async def orbit_deploy(
        application_id: str,
        version_id: str,
        instance_key: str = "default",
        force_recreate: bool = False,
    ) -> dict[str, Any]:
        """Create an Orbit deployment and immediately return its persisted command summary."""
        body = {
            "version_id": version_id,
            "instance_key": instance_key,
            "force_recreate": force_recreate,
        }
        action = await client.deploy_application(application_id, body)
        deployment_id = str(action["deployment_id"])
        deployment = await client.get_deployment(deployment_id)
        return write_result(
            "deploy_application",
            {
                "application_id": application_id,
                "version_id": version_id,
                "instance_key": instance_key,
                "deployment_id": deployment_id,
            },
            "POST",
            f"/api/application/{application_id}/deploy",
            request_body=body,
            steps=["Created Deployment through Orbit", "Read persisted Deployment command summary"],
            data={
                "deployment_id": deployment_id,
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
