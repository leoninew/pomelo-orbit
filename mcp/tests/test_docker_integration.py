from __future__ import annotations

import os
from pathlib import Path

import pytest

from pomelo_orbit_mcp.docker_runtime import DockerRuntime
from pomelo_orbit_mcp.workspace import RuntimeTarget

from .conftest import make_settings


@pytest.mark.docker
@pytest.mark.asyncio
async def test_external_compose_workspace_is_readable_when_explicitly_enabled(tmp_path) -> None:
    if os.environ.get("POMELO_ORBIT_RUN_DOCKER_TESTS") != "1":
        pytest.skip("set POMELO_ORBIT_RUN_DOCKER_TESTS=1 to run local Docker integration tests")
    workspace_value = os.environ.get("POMELO_ORBIT_DOCKER_TEST_WORKSPACE")
    project = os.environ.get("POMELO_ORBIT_DOCKER_TEST_PROJECT")
    if not workspace_value or not project:
        pytest.skip("set POMELO_ORBIT_DOCKER_TEST_WORKSPACE and POMELO_ORBIT_DOCKER_TEST_PROJECT")
    workspace = Path(workspace_value).resolve()
    target = RuntimeTarget(
        application_id="integration-app",
        service_id="integration-service",
        instance_key="integration",
        application_code="integration-app",
        working_directory=workspace,
        compose_project=project,
    )
    runtime = DockerRuntime(make_settings(tmp_path))
    config = await runtime.compose_config(target)
    ps = await runtime.compose_ps(target)
    assert config["compose_yaml"]
    assert isinstance(ps["containers"], list)
