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
        "orbit_create_application",
        "orbit_bootstrap_application",
        "orbit_get_application",
        "orbit_delete_application",
        "orbit_list_versions",
        "orbit_get_version",
        "orbit_create_version",
        "orbit_update_version",
        "orbit_publish_version",
        "orbit_delete_version",
        "orbit_preview_version",
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
        "verify_deployment",
    }
