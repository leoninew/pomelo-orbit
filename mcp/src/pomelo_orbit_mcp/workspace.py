"""Resolve a Docker Compose workspace from Orbit-managed domain data."""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from .settings import Settings


class RuntimeTargetError(ValueError):
    """Raised when an untrusted runtime target is outside Orbit management."""


_SEGMENT_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]*$")


def _safe_segment(value: object, label: str) -> str:
    text = str(value or "").strip()
    if not _SEGMENT_PATTERN.fullmatch(text) or text in {".", ".."}:
        raise RuntimeTargetError(f"invalid {label}")
    return text


def compose_project_name(app_code: str, environment_code: str, instance_key: str) -> str:
    raw = f"{app_code}-{environment_code}-{instance_key}".strip().lower()
    normalized = "".join(char if ("a" <= char <= "z") or ("0" <= char <= "9") or char in "-_" else "-" for char in raw)
    normalized = normalized.strip("-_")
    return normalized or "app"


@dataclass(frozen=True)
class RuntimeTarget:
    application_id: str
    environment_id: str
    service_id: str
    instance_key: str
    application_code: str
    environment_code: str
    working_directory: Path
    compose_project: str

    @property
    def compose_file(self) -> Path:
        return self.working_directory / "docker-compose.yml"

    def as_dict(self) -> dict[str, str]:
        return {
            "application_id": self.application_id,
            "environment_id": self.environment_id,
            "service_id": self.service_id,
            "instance_key": self.instance_key,
            "application_code": self.application_code,
            "environment_code": self.environment_code,
            "working_directory": str(self.working_directory),
            "compose_project": self.compose_project,
        }


def build_runtime_target(
    settings: Settings,
    application: dict[str, Any],
    environment: dict[str, Any],
    service: dict[str, Any],
    instance_key: str,
) -> RuntimeTarget:
    if application.get("kind") != "standard":
        raise RuntimeTargetError("runtime tools only support kind=standard applications")
    application_id = _safe_segment(application.get("id"), "application id")
    environment_id = _safe_segment(environment.get("id"), "environment id")
    service_id = _safe_segment(service.get("id"), "service id")
    application_code = _safe_segment(application.get("code"), "application code")
    environment_code = _safe_segment(environment.get("code"), "environment code")
    instance_key = _safe_segment(instance_key, "instance key")
    if service.get("application_id") != application_id:
        raise RuntimeTargetError("service does not belong to application")
    if service.get("environment_id") != environment_id:
        raise RuntimeTargetError("service does not belong to environment")
    if service.get("instance_key") != instance_key:
        raise RuntimeTargetError("service does not match instance key")
    if environment.get("project_id") != application.get("project_id"):
        raise RuntimeTargetError("environment does not belong to application project")

    root = settings.data_root.expanduser().resolve()
    workspace = (root / "deployment" / application_code / environment_code / instance_key).resolve()
    try:
        workspace.relative_to(root)
    except ValueError as error:
        raise RuntimeTargetError("workspace is outside configured data root") from error
    return RuntimeTarget(
        application_id=application_id,
        environment_id=environment_id,
        service_id=service_id,
        instance_key=instance_key,
        application_code=application_code,
        environment_code=environment_code,
        working_directory=workspace,
        compose_project=compose_project_name(application_code, environment_code, instance_key),
    )
