"""Authenticated, fixed-route HTTP client for the existing CI read API."""

from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass
from typing import Any

import httpx
import ulid

from .settings import Settings


@dataclass(frozen=True)
class PipelineAPIError(RuntimeError):
    """A safe summary of an HTTP failure that does not retain response bodies."""

    method: str
    path: str
    status_code: int | None
    code: str | None = None
    request_id: str | None = None

    def __str__(self) -> str:
        status = str(self.status_code) if self.status_code is not None else "network"
        return f"Pipeline API {self.method} {self.path} failed ({status})"


@dataclass(frozen=True)
class PipelineResponse:
    """One successful API response and the request identifier sent by this MCP."""

    data: dict[str, Any]
    request_id: str


class PipelineClient:
    """Calls only fixed, read-only CI API routes with a caller-provided JWT."""

    def __init__(self, settings: Settings, http_client: httpx.AsyncClient | None = None) -> None:
        self._settings = settings
        self._client = http_client or httpx.AsyncClient(
            base_url=settings.pipeline_url, timeout=30.0, follow_redirects=False
        )
        self._owns_client = http_client is None

    async def aclose(self) -> None:
        if self._owns_client:
            await self._client.aclose()

    async def list_stages(self, project_id: str, page: int, per_page: int, search: str | None) -> PipelineResponse:
        return await self._get(
            "/api/pipeline/stage",
            _compact({"project_id": project_id, "page": page, "per_page": per_page, "search": search}),
        )

    async def get_stage(self, stage_id: str) -> PipelineResponse:
        return await self._get(f"/api/pipeline/stage/{stage_id}")

    async def list_templates(self, project_id: str, page: int, per_page: int, search: str | None) -> PipelineResponse:
        return await self._get(
            "/api/pipeline/template",
            _compact({"project_id": project_id, "page": page, "per_page": per_page, "search": search}),
        )

    async def get_template(self, template_id: str) -> PipelineResponse:
        return await self._get(f"/api/pipeline/template/{template_id}")

    async def get_snapshot(self, snapshot_id: str) -> PipelineResponse:
        return await self._get(f"/api/pipeline/snapshot/{snapshot_id}")

    async def list_runs(
        self,
        project_id: str,
        page: int,
        per_page: int,
        repository_id: str | None,
        template_id: str | None,
    ) -> PipelineResponse:
        return await self._get(
            "/api/pipeline-run",
            _compact(
                {
                    "project_id": project_id,
                    "page": page,
                    "per_page": per_page,
                    "repository_id": repository_id,
                    "template_id": template_id,
                }
            ),
        )

    async def list_repository_runs(self, repository_id: str, page: int, per_page: int) -> PipelineResponse:
        return await self._get(f"/api/repository/{repository_id}/pipeline-run", {"page": page, "per_page": per_page})

    async def get_run(self, run_id: str) -> PipelineResponse:
        return await self._get(f"/api/pipeline-run/{run_id}")

    async def list_artifacts(
        self,
        project_id: str,
        page: int,
        per_page: int,
        repository_id: str | None,
        template_id: str | None,
        search: str | None,
    ) -> PipelineResponse:
        return await self._get(
            "/api/pipeline-run/artifact",
            _compact(
                {
                    "project_id": project_id,
                    "page": page,
                    "per_page": per_page,
                    "repository_id": repository_id,
                    "template_id": template_id,
                    "search": search,
                }
            ),
        )

    async def list_run_artifacts(self, run_id: str) -> PipelineResponse:
        return await self._get(f"/api/pipeline-run/{run_id}/artifact")

    async def get_stage_log(self, run_id: str, stage_run_id: str, offset: int) -> PipelineResponse:
        return await self._get(f"/api/pipeline-run/{run_id}/stage/{stage_run_id}/log", {"offset": offset})

    async def _get(self, path: str, params: Mapping[str, Any] | None = None) -> PipelineResponse:
        request_id = str(ulid.new())
        try:
            response = await self._client.get(
                path,
                params=params,
                headers={"Authorization": f"Bearer {self._settings.jwt}", "X-Request-ID": request_id},
            )
        except httpx.RequestError as error:
            raise PipelineAPIError("GET", path, None) from error

        if response.is_error:
            code, response_request_id = _error_metadata(response)
            raise PipelineAPIError("GET", path, response.status_code, code, response_request_id or request_id)
        return PipelineResponse(data=_json_object(response, path), request_id=request_id)


def _compact(values: Mapping[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in values.items() if value is not None}


def _json_object(response: httpx.Response, path: str) -> dict[str, Any]:
    try:
        value = response.json()
    except ValueError as error:
        raise PipelineAPIError("GET", path, response.status_code) from error
    if not isinstance(value, dict):
        raise PipelineAPIError("GET", path, response.status_code)
    return value


def _error_metadata(response: httpx.Response) -> tuple[str | None, str | None]:
    try:
        value = response.json()
    except ValueError:
        return None, response.headers.get("X-Request-ID")
    if not isinstance(value, dict):
        return None, response.headers.get("X-Request-ID")
    code = value.get("code")
    request_id = value.get("requestId")
    return (
        code if isinstance(code, str) and code else None,
        request_id if isinstance(request_id, str) and request_id else None,
    )
