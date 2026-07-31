from __future__ import annotations

import pytest

from pomelo_delivery_mcp.workspace import RuntimeTargetError, build_runtime_target, compose_project_name

from .conftest import make_settings


def test_build_runtime_target_uses_orbit_workspace_rule(tmp_path) -> None:
    settings = make_settings(tmp_path)
    target = build_runtime_target(
        settings,
        {"id": "app-1", "project_id": "project-1", "kind": "standard", "code": "demo-app"},
        {"id": "service-1", "application_id": "app-1", "instance_key": "default"},
        "default",
    )
    assert target.working_directory == tmp_path.resolve() / "deployment" / "demo-app" / "default"
    assert target.compose_project == "demo-app-default"
    assert compose_project_name("demo-app", "instance-1") == "demo-app-instance-1"


def test_build_runtime_target_allows_gateway_when_explicitly_requested(tmp_path) -> None:
    settings = make_settings(tmp_path)
    target = build_runtime_target(
        settings,
        {"id": "gateway-1", "kind": "gateway", "code": "traefik"},
        {"id": "service-1", "application_id": "gateway-1", "instance_key": "default"},
        "default",
        allowed_application_kinds=("gateway",),
    )

    assert target.working_directory == tmp_path.resolve() / "deployment" / "traefik" / "default"


def test_runtime_target_rejects_cross_project_or_path_escape(tmp_path) -> None:
    settings = make_settings(tmp_path)
    with pytest.raises(RuntimeTargetError):
        build_runtime_target(
            settings,
            {"id": "app-1", "project_id": "project-1", "kind": "standard", "code": "../escape"},
            {"id": "service-1", "application_id": "app-1", "instance_key": "default"},
            "default",
        )
