"""Safe error results for the read-only pipeline MCP."""

from __future__ import annotations

import re
from typing import Any

from mcp.server.fastmcp.exceptions import ToolError
from mcp.types import CallToolResult, TextContent
from pydantic import ValidationError

from .client import PipelineAPIError
from .settings import SettingsError

_CODE_PATTERN = re.compile(r"[a-z][a-z0-9_]*\Z")


def tool_error_result(error: BaseException) -> CallToolResult:
    """Map known failures to safe structured MCP errors."""
    cause = error.__cause__ if isinstance(error, ToolError) else error
    if isinstance(cause, PipelineAPIError):
        return _result(
            "pipeline_api",
            _stable_code(cause.code, "pipeline_request_failed"),
            "Pipeline query failed.",
            status=cause.status_code,
            request_id=cause.request_id,
        )
    if isinstance(cause, (SettingsError, ValidationError, ValueError)):
        return _result("validation", "validation_failed", "Invalid tool input or configuration.")
    return _result("internal", "internal_error", "Internal server error.")


def _result(
    kind: str,
    code: str,
    message: str,
    *,
    status: int | None = None,
    request_id: str | None = None,
) -> CallToolResult:
    payload: dict[str, Any] = {"kind": kind, "code": code, "message": message}
    if status is not None:
        payload["status"] = status
    if request_id:
        payload["request_id"] = request_id
    return CallToolResult(
        content=[TextContent(type="text", text=message)], structuredContent={"error": payload}, isError=True
    )


def _stable_code(value: str | None, default: str) -> str:
    return value if value and _CODE_PATTERN.fullmatch(value) else default
