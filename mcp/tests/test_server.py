from __future__ import annotations

import pytest

from pomelo_orbit_mcp.orbit_client import OrbitClient
from pomelo_orbit_mcp.server import create_server

from .conftest import make_settings


@pytest.mark.asyncio
async def test_server_registers_the_accepted_tool_surface(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        names = {tool.name for tool in await server.list_tools()}
    finally:
        await client.aclose()
    assert names == {
        "orbit_list_projects",
        "orbit_list_applications",
        "orbit_list_application_services",
        "orbit_list_gateways",
        "orbit_create_application",
        "orbit_create_gateway",
        "orbit_update_gateway",
        "orbit_get_application",
        "orbit_get_gateway",
        "orbit_delete_application",
        "orbit_list_versions",
        "orbit_get_version",
        "orbit_create_version",
        "orbit_update_version",
        "orbit_update_version_component",
        "orbit_publish_version",
        "orbit_delete_version",
        "orbit_preview_version",
        "orbit_create_service",
        "orbit_get_service_runtime_config",
        "orbit_update_service_runtime_config",
        "orbit_deploy",
        "orbit_stop",
        "orbit_restart",
        "orbit_deployment_status",
        "orbit_deployment_logs",
        "orbit_wait_deployment",
        "runtime_doctor",
        "runtime_compose_config",
        "runtime_compose_ps",
        "runtime_compose_logs",
        "runtime_container_inspect",
        "runtime_network_inspect",
        "runtime_http_probe",
        "verify_deployment",
    }


@pytest.mark.asyncio
async def test_create_gateway_exposes_local_traefik_defaults(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    schema = tools["orbit_create_gateway"].inputSchema
    assert schema["required"] == ["project_id"]
    assert schema["properties"]["code"]["default"] == "traefik"
    assert schema["properties"]["name"]["default"] == "Traefik"
    assert schema["properties"]["rest_api_url"]["default"] == "http://localhost:8080"
    assert schema["properties"]["base_domain"]["default"] == "lvh.me"
    assert schema["properties"]["image"]["default"] == "traefik:3.6"
