"""Resolve a Docker Compose workspace from Orbit-managed domain data."""

from __future__ import annotations

import re
from collections.abc import Collection
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


def compose_project_name(app_code: str, instance_key: str) -> str:
    return f"{app_code}-{instance_key}"


@dataclass(frozen=True)
class RuntimeTarget:
    application_id: str
    service_id: str
    instance_key: str
    application_code: str
    working_directory: Path
    compose_project: str

    @property
    def compose_file(self) -> Path:
        return self.working_directory / "docker-compose.yml"

    def as_dict(self) -> dict[str, str]:
        return {
            "application_id": self.application_id,
            "service_id": self.service_id,
            "instance_key": self.instance_key,
            "application_code": self.application_code,
            "working_directory": str(self.working_directory),
            "compose_project": self.compose_project,
        }


def build_runtime_target(
    settings: Settings,
    application: dict[str, Any],
    service: dict[str, Any],
    instance_key: str,
    *,
    allowed_application_kinds: Collection[str] = ("standard",),
) -> RuntimeTarget:
    application_kind = application.get("kind")
    if application_kind not in allowed_application_kinds:
        supported_kinds = ", ".join(sorted(allowed_application_kinds))
        raise RuntimeTargetError(f"runtime tools only support application kinds: {supported_kinds}")
    application_id = _safe_segment(application.get("id"), "application id")
    service_id = _safe_segment(service.get("id"), "service id")
    application_code = _safe_segment(application.get("code"), "application code")
    instance_key = _safe_segment(instance_key, "instance key")
    if service.get("application_id") != application_id:
        raise RuntimeTargetError("service does not belong to application")
    if service.get("instance_key") != instance_key:
        raise RuntimeTargetError("service does not match instance key")

    root = settings.data_root.expanduser().resolve()
    workspace = (root / "deployment" / application_code / instance_key).resolve()
    try:
        workspace.relative_to(root)
    except ValueError as error:
        raise RuntimeTargetError("workspace is outside configured data root") from error
    return RuntimeTarget(
        application_id=application_id,
        service_id=service_id,
        instance_key=instance_key,
        application_code=application_code,
        working_directory=workspace,
        compose_project=compose_project_name(application_code, instance_key),
    )
