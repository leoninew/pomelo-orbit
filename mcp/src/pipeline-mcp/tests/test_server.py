from __future__ import annotations

import pytest

from pomelo_pipeline_mcp.client import PipelineClient
from pomelo_pipeline_mcp.server import create_server, source_schema_fingerprint

from .conftest import make_settings


@pytest.mark.asyncio
async def test_server_registers_only_the_read_only_pipeline_tool_surface() -> None:
    client = PipelineClient(make_settings())
    try:
        server = create_server(settings=make_settings(), client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    assert server.name == "Pomelo Pipeline Queries"
    assert set(tools) == {
        "pipeline_list_stages",
        "pipeline_get_stage",
        "pipeline_list_templates",
        "pipeline_get_template",
        "pipeline_get_snapshot",
        "pipeline_list_runs",
        "pipeline_list_repository_runs",
        "pipeline_get_run",
        "pipeline_list_artifacts",
        "pipeline_list_run_artifacts",
        "pipeline_get_stage_log",
    }
    assert "pipeline_trigger_run" not in tools
    assert "pipeline_cancel_run" not in tools
    assert "pipeline_retry_run" not in tools
    assert "orbit_deploy" not in tools
    assert source_schema_fingerprint() in server.instructions


@pytest.mark.asyncio
async def test_read_tool_schemas_require_resource_scope_and_bound_log_output() -> None:
    client = PipelineClient(make_settings())
    try:
        server = create_server(settings=make_settings(), client=client)
        tools = {tool.name: tool for tool in await server.list_tools()}
    finally:
        await client.aclose()

    stages = tools["pipeline_list_stages"].inputSchema
    assert stages["required"] == ["project_id"]
    assert stages["properties"]["page_number"]["default"] == 1
    assert stages["properties"]["per_page"]["default"] == 20

    log = tools["pipeline_get_stage_log"].inputSchema
    assert log["required"] == ["run_id", "stage_run_id"]
    assert log["properties"]["offset"]["default"] == 0
    assert log["properties"]["max_bytes"]["default"] == 16384
