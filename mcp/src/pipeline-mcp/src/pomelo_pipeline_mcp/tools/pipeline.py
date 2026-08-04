"""Read-only CI pipeline MCP tools."""

from __future__ import annotations

import re
from typing import Any

from mcp.server.fastmcp import FastMCP

from ..client import PipelineClient
from ..projections import artifact, detail, list_result, page, run, snapshot, stage, stage_log, template

_IDENTIFIER = re.compile(r"[A-Za-z0-9][A-Za-z0-9_-]{0,127}\Z")
_MAX_PAGE_SIZE = 100
_MAX_LOG_BYTES = 32 * 1024


def register_pipeline_tools(mcp: FastMCP, client: PipelineClient) -> None:
    """Register the complete first-phase read-only pipeline tool surface."""

    @mcp.tool(name="pipeline_list_stages")
    async def list_stages(
        project_id: str, page_number: int = 1, per_page: int = 20, search: str | None = None
    ) -> dict[str, Any]:
        """List CI stages visible in one project; scripts are intentionally not returned."""
        response = await client.list_stages(
            _identifier(project_id, "project_id"),
            _page_number(page_number),
            _page_size(per_page),
            _search(search),
        )
        return page(response, stage)

    @mcp.tool(name="pipeline_get_stage")
    async def get_stage(stage_id: str) -> dict[str, Any]:
        """Read one CI stage summary without its executable script."""
        return detail(await client.get_stage(_identifier(stage_id, "stage_id")), stage)

    @mcp.tool(name="pipeline_list_templates")
    async def list_templates(
        project_id: str, page_number: int = 1, per_page: int = 20, search: str | None = None
    ) -> dict[str, Any]:
        """List CI templates visible in one project without script or variable values."""
        response = await client.list_templates(
            _identifier(project_id, "project_id"),
            _page_number(page_number),
            _page_size(per_page),
            _search(search),
        )
        return page(response, template)

    @mcp.tool(name="pipeline_get_template")
    async def get_template(template_id: str) -> dict[str, Any]:
        """Read one CI template topology without scripts or variable values."""
        return detail(await client.get_template(_identifier(template_id, "template_id")), template)

    @mcp.tool(name="pipeline_get_snapshot")
    async def get_snapshot(snapshot_id: str) -> dict[str, Any]:
        """Read immutable snapshot metadata without embedded scripts or variable values."""
        return detail(await client.get_snapshot(_identifier(snapshot_id, "snapshot_id")), snapshot)

    @mcp.tool(name="pipeline_list_runs")
    async def list_runs(
        project_id: str,
        page_number: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
    ) -> dict[str, Any]:
        """List CI runs visible in one project; status and stage summaries only."""
        response = await client.list_runs(
            _identifier(project_id, "project_id"),
            _page_number(page_number),
            _page_size(per_page),
            _optional_identifier(repository_id, "repository_id"),
            _optional_identifier(template_id, "template_id"),
        )
        return page(response, run)

    @mcp.tool(name="pipeline_list_repository_runs")
    async def list_repository_runs(repository_id: str, page_number: int = 1, per_page: int = 20) -> dict[str, Any]:
        """List CI runs visible for one repository; no run variables or error text are returned."""
        response = await client.list_repository_runs(
            _identifier(repository_id, "repository_id"), _page_number(page_number), _page_size(per_page)
        )
        return page(response, run)

    @mcp.tool(name="pipeline_get_run")
    async def get_run(run_id: str) -> dict[str, Any]:
        """Read one CI run status and stage summaries without variable values or error text."""
        return detail(await client.get_run(_identifier(run_id, "run_id")), run)

    @mcp.tool(name="pipeline_list_artifacts")
    async def list_artifacts(
        project_id: str,
        page_number: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
        search: str | None = None,
    ) -> dict[str, Any]:
        """List artifact metadata visible in one project without physical artifact paths."""
        response = await client.list_artifacts(
            _identifier(project_id, "project_id"),
            _page_number(page_number),
            _page_size(per_page),
            _optional_identifier(repository_id, "repository_id"),
            _optional_identifier(template_id, "template_id"),
            _search(search),
        )
        return page(response, artifact)

    @mcp.tool(name="pipeline_list_run_artifacts")
    async def list_run_artifacts(run_id: str) -> dict[str, Any]:
        """List metadata for artifacts of one visible CI run without physical paths."""
        return list_result(await client.list_run_artifacts(_identifier(run_id, "run_id")), artifact)

    @mcp.tool(name="pipeline_get_stage_log")
    async def get_stage_log(
        run_id: str,
        stage_run_id: str,
        offset: int = 0,
        max_bytes: int = 16384,
    ) -> dict[str, Any]:
        """Read a bounded, redacted stage-log window; reuse the returned offset to continue."""
        normalized_offset = _offset(offset)
        response = await client.get_stage_log(
            _identifier(run_id, "run_id"), _identifier(stage_run_id, "stage_run_id"), normalized_offset
        )
        return stage_log(response, normalized_offset, _log_bytes(max_bytes))


def _identifier(value: str, field_name: str) -> str:
    normalized = value.strip()
    if not _IDENTIFIER.fullmatch(normalized):
        raise ValueError(f"{field_name} must be a resource identifier")
    return normalized


def _optional_identifier(value: str | None, field_name: str) -> str | None:
    if value is None or not value.strip():
        return None
    return _identifier(value, field_name)


def _page_number(value: int) -> int:
    if value < 1:
        raise ValueError("page_number must be positive")
    return value


def _page_size(value: int) -> int:
    if value < 1 or value > _MAX_PAGE_SIZE:
        raise ValueError(f"per_page must be between 1 and {_MAX_PAGE_SIZE}")
    return value


def _offset(value: int) -> int:
    if value < 0:
        raise ValueError("offset must not be negative")
    return value


def _log_bytes(value: int) -> int:
    if value < 1 or value > _MAX_LOG_BYTES:
        raise ValueError(f"max_bytes must be between 1 and {_MAX_LOG_BYTES}")
    return value


def _search(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = value.strip()
    if len(normalized) > 200:
        raise ValueError("search must be at most 200 characters")
    return normalized or None
