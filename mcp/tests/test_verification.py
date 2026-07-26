from __future__ import annotations

import pytest

from pomelo_orbit_mcp.verification import observe_stability

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
