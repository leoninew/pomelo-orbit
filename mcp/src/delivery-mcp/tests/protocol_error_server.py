"""Standalone stdio server used by the MCP error protocol test."""

from __future__ import annotations

from pomelo_delivery_mcp.docker_runtime import DockerRuntimeError
from pomelo_delivery_mcp.orbit_client import OrbitAPIError
from pomelo_delivery_mcp.server import OrbitMCP
from pomelo_delivery_mcp.workspace import RuntimeTargetError


def main() -> None:
    server = OrbitMCP("protocol-error-server")

    @server.tool()
    def requires_integer(value: int) -> dict[str, str]:
        """Return a value after FastMCP validates the input."""
        return {"value": str(value)}

    @server.tool()
    def value_error() -> dict[str, str]:
        """Raise a validation-style tool error."""
        raise ValueError("input=not-for-output")

    @server.tool()
    def orbit_error() -> dict[str, str]:
        """Raise a normalized Orbit HTTP failure."""
        raise OrbitAPIError(
            "GET",
            "/api/application/application-1",
            404,
            "Resource not found.",
            "not_found",
            "request-protocol-test",
        )

    @server.tool()
    def runtime_error() -> dict[str, str]:
        """Raise an unsafe runtime failure that must be redacted."""
        raise DockerRuntimeError("docker stderr secret=not-for-output")

    @server.tool()
    def runtime_target_error() -> dict[str, str]:
        """Raise a rejected runtime target error."""
        raise RuntimeTargetError("workspace is outside configured data root")

    @server.tool()
    def internal_error() -> dict[str, str]:
        """Raise an unexpected failure that must be redacted."""
        raise RuntimeError("token=not-for-output")

    @server.tool()
    def domain_conclusion() -> dict[str, str]:
        """Return a normal deployment conclusion."""
        return {"conclusion": "failed"}

    server.run(transport="stdio")


if __name__ == "__main__":
    main()
