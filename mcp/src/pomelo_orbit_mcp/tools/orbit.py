"""Orbit HTTP control-plane MCP tools."""

from __future__ import annotations

from typing import Any

from mcp.server.fastmcp import FastMCP

from ..orbit_client import OrbitClient
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

    @mcp.tool(name="orbit_bootstrap_application")
    async def orbit_bootstrap_application(
        project_id: str,
        name: str,
        code: str,
        version_label: str,
        components: list[dict[str, Any]],
        exposes: list[dict[str, Any]],
        image_pull_policy: str = "missing",
        kind: str = "standard",
        version_env_json: str | None = None,
        version_note: str | None = None,
    ) -> dict[str, Any]:
        """Create an Application and its first complete Version in one Orbit import request."""
        if kind != "standard":
            raise ValueError("MCP creation only supports kind=standard")
        body = compact(
            {
                "name": name,
                "code": code,
                "kind": kind,
                "image_pull_policy": image_pull_policy,
                "version_label": version_label,
                "version_env_json": version_env_json,
                "version_note": version_note,
                "components": components,
                "exposes": exposes,
            }
        )
        application = await client.import_application(project_id, body)
        versions = await client.list_versions(str(application["id"]))
        initial = next((item for item in versions if item.get("label") == version_label), None)
        return write_result(
            "bootstrap_application",
            compact({"application_id": str(application["id"]), "version_id": str(initial["id"]) if initial else None}),
            "POST",
            "/api/application/import",
            request_body=body,
            steps=["Created Application and Version specification through Orbit", "Read created Version through Orbit"],
            data={"application": application, "version": initial},
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
        components: list[dict[str, Any]],
        exposes: list[dict[str, Any]],
        env_json: str | None = None,
        note: str | None = None,
    ) -> dict[str, Any]:
        """Create a Version using complete Component and Expose collections."""
        body = compact(
            {"label": label, "components": components, "exposes": exposes, "env_json": env_json, "note": note}
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
        components: list[dict[str, Any]] | None = None,
        exposes: list[dict[str, Any]] | None = None,
    ) -> dict[str, Any]:
        """Update a Version; explicit empty Components or Exposes replace that collection with empty."""
        body = compact(
            {"label": label, "env_json": env_json, "note": note, "components": components, "exposes": exposes}
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

    @mcp.tool(name="orbit_deploy")
    async def orbit_deploy(
        application_id: str,
        version_id: str,
        instance_key: str = "default",
        force_recreate: bool = False,
        runtime_config: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        """Create an Orbit deployment and immediately return its persisted command summary."""
        body = {
            "version_id": version_id,
            "instance_key": instance_key,
            "force_recreate": force_recreate,
            "runtime_config": runtime_config or {},
        }
        action = await client.deploy_application(application_id, body)
        deployment_id = str(action["deployment_id"])
        deployment = await client.get_deployment(deployment_id)
        return write_result(
            "deploy_application",
            {"application_id": application_id, "version_id": version_id, "deployment_id": deployment_id},
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
        instance_key: str | None = None,
        service_id: str | None = None,
        remove_volumes: bool = False,
    ) -> dict[str, Any]:
        """Create an Orbit stop deployment, optionally requesting managed volume removal."""
        body = compact(
            {
                "instance_key": instance_key,
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
        instance_key: str | None = None,
        service_id: str | None = None,
    ) -> dict[str, Any]:
        """Create an Orbit restart deployment and immediately return its command summary."""
        body = compact({"instance_key": instance_key, "service_id": service_id})
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
