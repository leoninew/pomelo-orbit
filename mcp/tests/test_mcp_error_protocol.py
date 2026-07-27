from __future__ import annotations

import json
import sys
from typing import Any

import pytest
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


@pytest.mark.asyncio
async def test_stdio_tool_errors_are_structured_safe_and_distinct_from_domain_results() -> None:
    parameters = StdioServerParameters(command=sys.executable, args=["-m", "tests.protocol_error_server"])
    async with stdio_client(parameters) as (read_stream, write_stream):
        async with ClientSession(read_stream, write_stream) as session:
            await session.initialize()

            validation = await session.call_tool("requires_integer", {"value": "not-an-integer"})
            value_error = await session.call_tool("value_error", {})
            orbit = await session.call_tool("orbit_error", {})
            runtime = await session.call_tool("runtime_error", {})
            runtime_target = await session.call_tool("runtime_target_error", {})
            internal = await session.call_tool("internal_error", {})
            unknown = await session.call_tool("missing_tool", {})
            domain = await session.call_tool("domain_conclusion", {})

    assert_error(validation, "validation", "validation_failed", "Invalid tool input.")
    assert_error(value_error, "validation", "validation_failed", "Invalid tool input.")
    assert_error(orbit, "orbit_api", "not_found", "Resource not found.")
    assert orbit.structuredContent == {
        "error": {
            "kind": "orbit_api",
            "code": "not_found",
            "message": "Resource not found.",
            "status": 404,
            "request_id": "request-protocol-test",
        }
    }
    assert_error(runtime, "runtime", "runtime_failed", "Runtime operation failed.")
    assert_error(runtime_target, "runtime", "runtime_failed", "Runtime operation failed.")
    assert_error(internal, "internal", "internal_error", "Internal server error.")
    assert_error(unknown, "validation", "tool_not_found", "Tool is not available.")
    assert "not-for-output" not in json.dumps(runtime.model_dump(mode="json"))
    assert "not-for-output" not in json.dumps(internal.model_dump(mode="json"))
    assert domain.isError is False
    assert domain.structuredContent == {"conclusion": "failed"}


def assert_error(result: Any, kind: str, code: str, message: str) -> None:
    assert result.isError is True
    assert result.structuredContent is not None
    assert result.structuredContent["error"].items() >= {"kind": kind, "code": code, "message": message}.items()
    assert result.content[0].text == message
