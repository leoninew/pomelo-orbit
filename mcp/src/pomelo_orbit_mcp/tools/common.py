"""Shared, credential-free MCP response helpers."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from ..workspace import RuntimeTarget


def compact(values: Mapping[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in values.items() if value is not None}


def write_result(
    operation: str,
    resource_ids: Mapping[str, str],
    method: str,
    path: str,
    *,
    request_body: Mapping[str, Any] | None = None,
    steps: list[str] | None = None,
    data: Mapping[str, Any] | None = None,
) -> dict[str, Any]:
    result: dict[str, Any] = {
        "operation": operation,
        "resource_ids": dict(resource_ids),
        "steps": steps or [f"Orbit HTTP {method} {path} completed"],
        "request_summary": compact(
            {
                "method": method,
                "path": path,
                "body": dict(request_body) if request_body is not None else None,
            }
        ),
    }
    if data:
        result.update(data)
    return result


def runtime_result(target: RuntimeTarget, data: Mapping[str, Any]) -> dict[str, Any]:
    return {
        "target": target.as_dict(),
        "working_directory": str(target.working_directory),
        **dict(data),
    }
