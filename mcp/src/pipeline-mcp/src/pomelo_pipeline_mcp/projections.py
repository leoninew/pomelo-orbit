"""Credential-safe projections for CI API response payloads."""

from __future__ import annotations

import re
from collections.abc import Callable, Mapping
from typing import Any

from .client import PipelineResponse

_SECRET_ASSIGNMENT = re.compile(
    r"(?im)(\b(?:access[_-]?token|api[_-]?key|auth[_-]?token|password|secret|token)\b\s*(?:=|:)\s*)([^\s]+)"
)
_BEARER_TOKEN = re.compile(r"(?i)(\bbearer\s+)[A-Za-z0-9._~+/=-]+")
_GITHUB_TOKEN = re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{20,}\b")
_JWT = re.compile(r"\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b")


def stage(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return a stage summary without its executable script."""
    result = _select(value, "id", "name", "image", "description", "version", "created_at", "updated_at")
    artifacts = value.get("artifacts")
    if isinstance(artifacts, list):
        result["artifacts"] = [artifact_config(item) for item in artifacts if isinstance(item, Mapping)]
    return result


def template(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return template topology without scripts or variable default/value fields."""
    result = _select(value, "id", "name", "description", "orchestration", "version", "created_at", "updated_at")
    stages = value.get("stages")
    if isinstance(stages, list):
        result["stages"] = [stage(item) for item in stages if isinstance(item, Mapping)]
    declarations = value.get("variable_declarations")
    if isinstance(declarations, list):
        result["variable_declarations"] = [
            variable_declaration(item) for item in declarations if isinstance(item, Mapping)
        ]
    return result


def snapshot(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return immutable snapshot metadata without snapshot scripts or variable values."""
    result = _select(value, "id", "template_id", "template_name", "template_version", "version", "created_at")
    stages = value.get("stages_snapshot")
    if isinstance(stages, list):
        result["stages_snapshot"] = [snapshot_stage(item) for item in stages if isinstance(item, Mapping)]
    declarations = value.get("variables_snapshot")
    if isinstance(declarations, list):
        result["variables_snapshot"] = [
            variable_declaration(item) for item in declarations if isinstance(item, Mapping)
        ]
    return result


def run(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return run status and stage summaries without variables or error text."""
    result = _select(
        value,
        "id",
        "project_id",
        "repository_id",
        "repository_name",
        "snapshot_id",
        "template_id",
        "template_name",
        "template_version",
        "trigger",
        "trigger_ref",
        "status",
        "retry_of",
        "started_at",
        "finished_at",
        "created_at",
    )
    declarations = value.get("variables_snapshot")
    if isinstance(declarations, list):
        result["variables_snapshot"] = [
            variable_declaration(item) for item in declarations if isinstance(item, Mapping)
        ]
    stage_runs = value.get("pipeline_stage_runs")
    if isinstance(stage_runs, list):
        result["pipeline_stage_runs"] = [
            _select(
                item,
                "id",
                "pipeline_run_id",
                "stage_id",
                "stage_name",
                "status",
                "started_at",
                "finished_at",
                "exit_code",
            )
            for item in stage_runs
            if isinstance(item, Mapping)
        ]
    return result


def artifact(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return artifact metadata without its local physical path."""
    return _select(
        value,
        "id",
        "pipeline_run_id",
        "repository_id",
        "repository_name",
        "template_id",
        "template_name",
        "stage_name",
        "type",
        "name",
        "created_at",
    )


def artifact_config(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return artifact metadata without the workspace path used to collect it."""
    return _select(value, "type", "name")


def snapshot_stage(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return one snapshot stage without its script or artifact workspace paths."""
    result = _select(value, "id", "name", "image", "version", "depends_on")
    artifacts = value.get("artifacts")
    if isinstance(artifacts, list):
        result["artifacts"] = [artifact_config(item) for item in artifacts if isinstance(item, Mapping)]
    return result


def page(response: PipelineResponse, item_projection: Callable[[Mapping[str, Any]], dict[str, Any]]) -> dict[str, Any]:
    """Project a paginated CI payload while retaining its pagination contract."""
    result = _select(response.data, "total", "page", "per_page", "pages")
    items = response.data.get("items")
    result["items"] = (
        [item_projection(item) for item in items if isinstance(item, Mapping)] if isinstance(items, list) else []
    )
    result["request_id"] = response.request_id
    return result


def list_result(
    response: PipelineResponse, item_projection: Callable[[Mapping[str, Any]], dict[str, Any]]
) -> dict[str, Any]:
    """Project a non-paginated item collection."""
    items = response.data.get("items")
    return {
        "items": [item_projection(item) for item in items if isinstance(item, Mapping)]
        if isinstance(items, list)
        else [],
        "request_id": response.request_id,
    }


def detail(response: PipelineResponse, projection: Callable[[Mapping[str, Any]], dict[str, Any]]) -> dict[str, Any]:
    """Project a single response and attach the locally generated request ID."""
    return {**projection(response.data), "request_id": response.request_id}


def stage_log(response: PipelineResponse, offset: int, max_bytes: int) -> dict[str, Any]:
    """Clip and redact logs while returning an offset that can safely continue the read."""
    raw_logs = response.data.get("logs")
    logs = raw_logs if isinstance(raw_logs, str) else ""
    chunk, truncated = _utf8_prefix(logs, max_bytes)
    returned_offset = offset + len(chunk.encode("utf-8"))
    server_offset = response.data.get("offset")
    is_complete = bool(response.data.get("is_complete")) and not truncated
    result: dict[str, Any] = {
        "logs": redact_logs(chunk),
        "offset": returned_offset,
        "is_complete": is_complete,
        "truncated": truncated,
        "request_id": response.request_id,
    }
    if isinstance(server_offset, int):
        result["server_offset"] = server_offset
    return result


def redact_logs(value: str) -> str:
    """Mask common credential formats before exposing CI stage output."""
    value = _SECRET_ASSIGNMENT.sub(r"\1[REDACTED]", value)
    value = _BEARER_TOKEN.sub(r"\1[REDACTED]", value)
    value = _GITHUB_TOKEN.sub("[REDACTED]", value)
    return _JWT.sub("[REDACTED]", value)


def variable_declaration(value: Mapping[str, Any]) -> dict[str, Any]:
    """Return declaration metadata only; values and defaults can be sensitive."""
    return _select(value, "name", "description", "secret", "source", "editable")


def _select(value: Mapping[str, Any], *keys: str) -> dict[str, Any]:
    return {key: value[key] for key in keys if value.get(key) is not None}


def _utf8_prefix(value: str, max_bytes: int) -> tuple[str, bool]:
    encoded = value.encode("utf-8")
    if len(encoded) <= max_bytes:
        return value, False
    prefix = encoded[:max_bytes].decode("utf-8", errors="ignore")
    return prefix, True
