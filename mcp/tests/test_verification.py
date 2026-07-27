from __future__ import annotations

import pytest

from pomelo_orbit_mcp.docker_runtime import DockerRuntimeError
from pomelo_orbit_mcp.verification import observe_stability, verify_deployment

from .conftest import make_settings
from .test_docker_runtime import target


class StabilityRuntime:
    def __init__(self, states):
        self.states = iter(states)
        self.current = None

    async def compose_ps(self, _target):
        self.current = next(self.states)
        return {
            "containers": [
                {"ID": "container-1", "State": self.current["state"], "Status": self.current.get("health", "")}
            ]
        }

    async def container_inspect(self, _target, _container_id):
        return {
            "inspect": [
                {
                    "State": {"Status": self.current["state"], "Health": {"Status": self.current.get("health", "")}},
                    "RestartCount": self.current.get("restarts", 0),
                }
            ]
        }


async def no_sleep(_seconds: float) -> None:
    return None


@pytest.mark.asyncio
async def test_stability_marks_unhealthy_container_failed(tmp_path) -> None:
    result = await observe_stability(
        StabilityRuntime([{"state": "running", "health": "unhealthy"}]),
        target(tmp_path),
        make_settings(tmp_path),
    )
    assert result["state"] == "failed"
    assert "not healthy" in result["issues"][0]


@pytest.mark.asyncio
async def test_stability_detects_restart_count_increase_without_waiting(tmp_path) -> None:
    clock_values = iter([0.0, 0.0, 0.0])
    result = await observe_stability(
        StabilityRuntime([{"state": "running", "restarts": 0}, {"state": "running", "restarts": 1}]),
        target(tmp_path),
        make_settings(tmp_path, stability_window_seconds=10),
        sleep=no_sleep,
        monotonic=lambda: next(clock_values),
    )
    assert result["state"] == "failed"
    assert "restart count increased" in result["issues"][0]


class FailingRuntime:
    async def compose_ps(self, _target):
        raise DockerRuntimeError("docker stderr secret=not-for-output")


@pytest.mark.asyncio
async def test_stability_propagates_runtime_errors(tmp_path) -> None:
    with pytest.raises(DockerRuntimeError, match="secret=not-for-output"):
        await observe_stability(FailingRuntime(), target(tmp_path), make_settings(tmp_path))


class VerificationClient:
    async def get_deployment(self, _deployment_id):
        return {
            "application_id": "application-1",
            "status": "ran_to_completion",
            "operation_type": "deploy",
            "service_id": "service-1",
            "version_id": "version-1",
        }

    async def get_application(self, _application_id):
        return {"id": "application-1", "kind": "standard", "code": "demo"}

    async def get_version(self, _version_id):
        return {"id": "version-1"}

    async def get_service(self, _service_id):
        return {"id": "service-1", "application_id": "application-1", "instance_key": "default"}

    async def preview_version(self, _version_id, _instance_key):
        return {"compose_yaml": "services: {}"}


class VerificationRuntime:
    async def compose_config(self, _target):
        raise DockerRuntimeError("docker stderr secret=not-for-output")


@pytest.mark.asyncio
async def test_verification_propagates_runtime_errors_instead_of_returning_inconclusive(tmp_path) -> None:
    with pytest.raises(DockerRuntimeError, match="secret=not-for-output"):
        await verify_deployment(
            VerificationClient(),
            VerificationRuntime(),
            make_settings(tmp_path),
            "application-1",
            "deployment-1",
        )
