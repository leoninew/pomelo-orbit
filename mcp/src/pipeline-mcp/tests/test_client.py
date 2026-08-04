from __future__ import annotations

import httpx
import pytest
import ulid

from pomelo_pipeline_mcp.client import PipelineAPIError, PipelineClient

from .conftest import make_settings


@pytest.mark.asyncio
async def test_list_runs_uses_only_authenticated_get_with_fixed_query_fields() -> None:
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        return httpx.Response(200, json={"items": [], "total": 0, "page": 1, "per_page": 20, "pages": 0})

    http_client = httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://pipeline.test")
    client = PipelineClient(make_settings(), http_client)
    try:
        response = await client.list_runs("project-1", 1, 20, "repository-1", "template-1")
    finally:
        await http_client.aclose()

    assert response.data["items"] == []
    assert response.request_id
    assert ulid.parse(response.request_id)
    assert len(requests) == 1
    request = requests[0]
    assert request.method == "GET"
    assert request.url.path == "/api/pipeline-run"
    assert dict(request.url.params) == {
        "project_id": "project-1",
        "page": "1",
        "per_page": "20",
        "repository_id": "repository-1",
        "template_id": "template-1",
    }
    assert request.headers["Authorization"] == "Bearer not-for-output"
    assert request.headers["X-Request-ID"] == response.request_id


@pytest.mark.asyncio
async def test_client_does_not_include_error_body_in_the_failure_summary() -> None:
    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            403,
            json={"code": "forbidden", "error": "password=secret-value", "requestId": "server-request-id"},
        )

    http_client = httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://pipeline.test")
    client = PipelineClient(make_settings(), http_client)
    try:
        with pytest.raises(PipelineAPIError) as raised:
            await client.get_run("run-1")
    finally:
        await http_client.aclose()

    error = raised.value
    assert error.status_code == 403
    assert error.code == "forbidden"
    assert error.request_id == "server-request-id"
    assert "secret-value" not in str(error)
