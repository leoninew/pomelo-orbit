from __future__ import annotations

from pathlib import Path

from pomelo_delivery_mcp.settings import Settings


def make_settings(
    tmp_path: Path, *, jwt_from_environment: str | None = None, stability_window_seconds: int = 0
) -> Settings:
    return Settings(
        orbit_url="http://orbit.test",
        username="tester",
        password="not-for-output",
        data_root=tmp_path,
        docker_context=None,
        wait_timeout_seconds=300,
        stability_window_seconds=stability_window_seconds,
        stability_poll_seconds=1,
        config_dir=tmp_path,
        jwt_from_environment=jwt_from_environment,
    )
