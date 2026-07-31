from __future__ import annotations

import pytest

from pomelo_delivery_mcp.orbit_client import OrbitClient
from pomelo_delivery_mcp.server import create_server, source_schema_fingerprint

from .conftest import make_settings


@pytest.mark.asyncio
async def test_server_registers_the_accepted_tool_surface(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        assert server.name == "Pomelo Delivery Control"
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
        "orbit_provision_gateway",
        "orbit_update_gateway",
        "orbit_get_application",
        "orbit_get_gateway",
        "orbit_delete_application",
        "orbit_list_versions",
        "orbit_get_version",
        "orbit_create_version_component",
        "orbit_create_version",
        "orbit_update_version",
        "orbit_update_version_component_basic",
        "orbit_update_version_component_runtime",
        "orbit_update_version_component_ports",
        "orbit_update_version_component_env",
        "orbit_update_version_component_mounts",
        "orbit_update_version_component_dependencies",
        "orbit_update_version_component_devices",
        "orbit_update_version_component_advanced",
        "orbit_update_version_component_resources",
        "orbit_update_version_component_tmpfs",
        "orbit_update_version_component_ulimits",
        "orbit_publish_version",
        "orbit_delete_version",
        "orbit_preview_service",
        "orbit_create_service",
        "orbit_update_service_configuration",
        "orbit_update_service_basic",
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


@pytest.mark.asyncio
async def test_server_exposes_provisioning_and_summary_schema_defaults(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    assert "orbit_update_version_component" not in tools
    assert tools["orbit_provision_gateway"].inputSchema["required"] == ["project_id"]
    assert tools["runtime_doctor"].inputSchema["properties"]["network_name"]["anyOf"][0]["type"] == "string"
    assert tools["runtime_compose_ps"].inputSchema["properties"]["detail"]["default"] is False
    assert tools["verify_deployment"].inputSchema["properties"]["detail"]["default"] is False
    assert source_schema_fingerprint() in server.instructions


@pytest.mark.asyncio
async def test_server_component_group_schemas_match_the_current_json_contract(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    basic = tools["orbit_update_version_component_basic"].inputSchema
    basic_definition = basic["$defs"]["VersionComponentBasicUpdate"]
    assert basic["required"] == ["version_id", "component_id", "basic"]
    assert set(basic_definition["properties"]) == {
        "name",
        "image",
        "command",
        "pull_policy",
        "restart_policy",
    }
    assert basic_definition["properties"]["command"] == {"title": "Command", "type": "string"}
    assert set(basic_definition["required"]) == {"name", "image", "command", "pull_policy"}

    create_component = tools["orbit_create_version_component"].inputSchema
    create_component_definition = create_component["$defs"]["VersionComponentCreate"]
    assert create_component["required"] == ["version_id", "component"]
    assert set(create_component_definition["properties"]) == {
        "name",
        "image",
        "command",
        "pull_policy",
        "restart_policy",
    }
    assert create_component_definition["properties"]["command"] == {
        "default": "",
        "title": "Command",
        "type": "string",
    }
    assert set(create_component_definition["required"]) == {"name", "image", "pull_policy"}

    runtime = tools["orbit_update_version_component_runtime"].inputSchema
    runtime_definition = runtime["$defs"]["VersionComponentRuntimeUpdate"]
    assert runtime["required"] == ["version_id", "component_id", "runtime"]
    assert set(runtime_definition["properties"]) == {"healthcheck"}

    advanced = tools["orbit_update_version_component_advanced"].inputSchema
    advanced_definition = advanced["$defs"]["VersionComponentAdvancedUpdate"]
    assert advanced["required"] == ["version_id", "component_id", "advanced"]
    assert set(advanced_definition["properties"]) == {"resources", "tmpfs", "ulimits"}
    assert advanced_definition["required"] == ["tmpfs", "ulimits"]

    for tool_name, definition_name, field_name in (
        ("orbit_update_version_component_resources", "VersionComponentResourcesUpdate", "resources"),
        ("orbit_update_version_component_tmpfs", "VersionComponentTmpfsUpdate", "tmpfs"),
        ("orbit_update_version_component_ulimits", "VersionComponentUlimitsUpdate", "ulimits"),
        ("orbit_update_version_component_devices", "VersionComponentDevicesUpdate", "devices"),
    ):
        schema = tools[tool_name].inputSchema
        definition = schema["$defs"][definition_name]
        assert schema["required"] == ["version_id", "component_id", field_name]
        assert set(definition["properties"]) == {field_name}
        assert definition["required"] == [field_name]

    create_version = tools["orbit_create_version"].inputSchema
    component_definition = create_version["$defs"]["VersionComponent"]
    assert set(component_definition["properties"]) == {
        "name",
        "image",
        "command",
        "env",
        "ports",
        "mounts",
        "dependencies",
        "healthcheck",
        "resources",
        "pull_policy",
        "restart_policy",
        "tmpfs",
        "ulimits",
        "devices",
    }
    assert set(component_definition["required"]) == {"name", "image", "pull_policy"}
    assert "networks" not in component_definition["properties"]

    for tool_name, definition_name, field_name in (
        ("orbit_update_version_component_ports", "VersionComponentPortsUpdate", "ports"),
        ("orbit_update_version_component_env", "VersionComponentEnvUpdate", "env"),
        ("orbit_update_version_component_mounts", "VersionComponentMountsUpdate", "mounts"),
        ("orbit_update_version_component_dependencies", "VersionComponentDependenciesUpdate", "dependencies"),
    ):
        schema = tools[tool_name].inputSchema
        definition = schema["$defs"][definition_name]
        assert schema["required"] == ["version_id", "component_id", field_name]
        assert set(definition["properties"]) == {field_name}
        assert definition["required"] == [field_name]

    assert "orbit_update_version_component_connectivity" not in tools


@pytest.mark.asyncio
async def test_server_service_schemas_target_saved_service_configuration(tmp_path) -> None:
    settings = make_settings(tmp_path)
    client = OrbitClient(settings)
    try:
        server = create_server(settings=settings, client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    assert "orbit_preview_version" not in tools

    create = tools["orbit_create_service"].inputSchema
    assert create["required"] == ["application_id", "version_id", "instance_key", "runtime_config", "exposes"]
    expose = create["$defs"]["ServiceExpose"]
    assert set(expose["properties"]) == {
        "component_name",
        "protocol",
        "container_port",
        "path_prefix",
        "access",
        "listen_port",
    }
    assert expose["required"] == ["component_name", "protocol", "container_port", "access"]

    configuration = tools["orbit_update_service_configuration"].inputSchema
    assert configuration["required"] == ["service_id", "runtime_config", "exposes"]
    assert "version_id" not in configuration["properties"]

    basic = tools["orbit_update_service_basic"].inputSchema
    assert basic["required"] == ["service_id", "version_id", "instance_key"]

    preview = tools["orbit_preview_service"].inputSchema
    assert preview["required"] == ["service_id"]

    deploy = tools["orbit_deploy"].inputSchema
    assert deploy["required"] == ["service_id"]
    assert deploy["properties"]["force_recreate"]["default"] is False
    assert "application_id" not in deploy["properties"]
    assert "version_id" not in deploy["properties"]
    assert "instance_key" not in deploy["properties"]
