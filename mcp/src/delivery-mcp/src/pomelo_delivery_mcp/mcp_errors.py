"""Safe MCP tool error results."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any, Literal

from mcp.server.fastmcp.exceptions import ToolError
from mcp.types import CallToolResult, TextContent
from pydantic import ValidationError

from .docker_runtime import DockerRuntimeError
from .orbit_client import OrbitAPIError
from .workspace import RuntimeTargetError

ErrorKind = Literal["validation", "orbit_api", "runtime", "internal"]

_CODE_PATTERN = re.compile(r"[a-z][a-z0-9_]*\Z")


@dataclass(frozen=True)
class MCPToolError:
    kind: ErrorKind
    code: str
    message: str
    status: int | None = None
    request_id: str | None = None

    def as_dict(self) -> dict[str, Any]:
        result: dict[str, Any] = {
            "kind": self.kind,
            "code": self.code,
            "message": self.message,
        }
        if self.status is not None:
            result["status"] = self.status
        if self.request_id is not None:
            result["request_id"] = self.request_id
        return result


def tool_error_result(error: BaseException) -> CallToolResult:
    """Convert a FastMCP tool failure into a safe, structured MCP result."""
    mapped = classify_tool_error(error)
    return CallToolResult(
        content=[TextContent(type="text", text=mapped.message)],
        structuredContent={"error": mapped.as_dict()},
        isError=True,
    )


def classify_tool_error(error: BaseException) -> MCPToolError:
    """Classify only trusted error types; unknown details never reach the client."""
    cause = error.__cause__ if isinstance(error, ToolError) else error
    if isinstance(cause, OrbitAPIError):
        return MCPToolError(
            kind="orbit_api",
            code=_stable_code(cause.code, "orbit_request_failed"),
            message=cause.message,
            status=cause.status_code,
            request_id=_request_id(cause.request_id),
        )
    if isinstance(cause, (RuntimeTargetError, DockerRuntimeError)):
        return MCPToolError(kind="runtime", code="runtime_failed", message="Runtime operation failed.")
    if isinstance(cause, (ValidationError, ValueError)):
        return MCPToolError(kind="validation", code="validation_failed", message="Invalid tool input.")
    if isinstance(error, ToolError) and cause is None:
        return MCPToolError(kind="validation", code="tool_not_found", message="Tool is not available.")
    return MCPToolError(kind="internal", code="internal_error", message="Internal server error.")


def _stable_code(value: str | None, default: str) -> str:
    if value and _CODE_PATTERN.fullmatch(value):
        return value
    return default


def _request_id(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = value.strip()
    return normalized or None
