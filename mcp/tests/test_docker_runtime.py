from __future__ import annotations

import json

import pytest

from pomelo_orbit_mcp.docker_runtime import CommandResult, DockerRuntime, DockerRuntimeError
from pomelo_orbit_mcp.workspace import RuntimeTarget

from .conftest import make_settings


class FakeRunner:
    def __init__(self) -> None:
        self.calls: list[tuple[list[str], object]] = []

    async def __call__(self, args, cwd):
        values = list(args)
        self.calls.append((values, cwd))
        if values[-2:] == ["--format", "json"]:
            return CommandResult(
                tuple(values), json.dumps([{"ID": "container-1", "Service": "web", "State": "running"}]), "", 0
            )
        if values[1:2] == ["inspect"]:
            return CommandResult(
                tuple(values),
                json.dumps(
                    [
                        {
                            "Config": {"Labels": {"com.docker.compose.service": "web"}},
                            "NetworkSettings": {"Networks": {"demo_default": {}}},
                        }
                    ]
                ),
                "",
                0,
            )
        return CommandResult(tuple(values), "logs", "", 0)


def target(tmp_path) -> RuntimeTarget:
    return RuntimeTarget(
        application_id="app-1",
        environment_id="env-1",
        service_id="service-1",
        instance_key="default",
        application_code="demo",
        environment_code="local",
        working_directory=tmp_path,
        compose_project="demo-local-default",
    )


@pytest.mark.asyncio
async def test_runtime_builds_only_read_only_compose_commands(tmp_path) -> None:
    runner = FakeRunner()
    runtime = DockerRuntime(make_settings(tmp_path), runner)
    resolved = target(tmp_path)
    logs = await runtime.compose_logs(resolved, tail=25, services=["web"])
    assert logs["services"] == ["web"]
    assert runner.calls[-1][0] == [
        "docker",
        "compose",
        "-p",
        "demo-local-default",
        "-f",
        "docker-compose.yml",
        "logs",
        "--tail",
        "25",
        "web",
    ]
    forbidden = {"up", "down", "restart", "rm"}
    assert not any(forbidden.intersection(command) for command, _ in runner.calls)


@pytest.mark.asyncio
async def test_runtime_rejects_container_not_derived_from_ps(tmp_path) -> None:
    runtime = DockerRuntime(make_settings(tmp_path), FakeRunner())
    with pytest.raises(DockerRuntimeError, match="was not returned"):
        await runtime.container_inspect(target(tmp_path), "outside-container")
