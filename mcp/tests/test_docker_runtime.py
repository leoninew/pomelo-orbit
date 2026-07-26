from __future__ import annotations

import json

import pytest

from pomelo_orbit_mcp.docker_runtime import CommandResult, DockerRuntime, DockerRuntimeError
from pomelo_orbit_mcp.workspace import RuntimeTarget

from .conftest import make_settings


class FakeRunner:
    def __init__(self, *, network_name: str = "demo_default", probe_return_code: int = 0) -> None:
        self.calls: list[tuple[list[str], object]] = []
        self.network_name = network_name
        self.probe_return_code = probe_return_code

    async def __call__(self, args, cwd):
        values = list(args)
        self.calls.append((values, cwd))
        if values[-2:] == ["--format", "json"]:
            return CommandResult(
                tuple(values), json.dumps([{"ID": "container-1", "Service": "web", "State": "running"}]), "", 0
            )
        if values[:3] == ["docker", "network", "inspect"]:
            return CommandResult(
                tuple(values),
                json.dumps([{"Name": values[-1], "Driver": "bridge", "Scope": "local"}]),
                "",
                0,
            )
        if values[:2] == ["docker", "exec"]:
            return CommandResult(
                tuple(values), "probe-secret=not-for-output", "probe-secret=not-for-output", self.probe_return_code
            )
        if values[1:2] == ["inspect"]:
            return CommandResult(
                tuple(values),
                json.dumps(
                    [
                        {
                            "Config": {"Labels": {"com.docker.compose.service": "web"}},
                            "NetworkSettings": {"Networks": {self.network_name: {}}},
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
        service_id="service-1",
        instance_key="default",
        application_code="demo",
        working_directory=tmp_path,
        compose_project="demo-default",
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
        "demo-default",
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


@pytest.mark.asyncio
async def test_runtime_doctor_reports_only_derived_traefik_network(tmp_path) -> None:
    runner = FakeRunner(network_name="traefik")
    runtime = DockerRuntime(make_settings(tmp_path), runner)
    result = await runtime.doctor(target(tmp_path), expected_external_networks=("traefik",))
    assert result["healthy"] is True
    assert result["external_networks"] == ["traefik"]
    assert result["external_network_details"] == [{"name": "traefik", "driver": "bridge", "scope": "local"}]
    assert ["docker", "network", "inspect", "traefik"] in [call[0] for call in runner.calls]


@pytest.mark.asyncio
async def test_runtime_doctor_reports_missing_derived_traefik_network(tmp_path) -> None:
    runtime = DockerRuntime(make_settings(tmp_path), FakeRunner())
    result = await runtime.doctor(target(tmp_path), expected_external_networks=("traefik",))
    assert result["healthy"] is False
    assert result["external_networks"] == []
    assert result["issues"] == ["external network traefik is unavailable from the managed target"]


@pytest.mark.asyncio
async def test_runtime_http_probe_uses_fixed_derived_container_command(tmp_path) -> None:
    runner = FakeRunner()
    runtime = DockerRuntime(make_settings(tmp_path), runner)
    result = await runtime.http_probe(target(tmp_path), "web", 80, "/health")
    assert result == {"status": "reachable", "component_name": "web", "port": 80, "path": "/health"}
    assert runner.calls[-1][0] == [
        "docker",
        "exec",
        "container-1",
        "curl",
        "-fsS",
        "--max-time",
        "10",
        "--output",
        "/dev/null",
        "http://127.0.0.1:80/health",
    ]


@pytest.mark.asyncio
async def test_runtime_http_probe_hides_container_output_on_failure(tmp_path) -> None:
    runtime = DockerRuntime(make_settings(tmp_path), FakeRunner(probe_return_code=1))
    with pytest.raises(DockerRuntimeError) as raised:
        await runtime.http_probe(target(tmp_path), "web", 80)
    assert str(raised.value) == "HTTP probe failed"
    assert "not-for-output" not in str(raised.value)
