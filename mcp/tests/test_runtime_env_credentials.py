from __future__ import annotations

import json

import httpx
import pytest

from pomelo_orbit_mcp.orbit_client import OrbitClient
from pomelo_orbit_mcp.tools.orbit import runtime_env_credential_metadata

from .conftest import make_settings


@pytest.mark.asyncio
async def test_runtime_env_credential_client_sends_values_only_to_orbit(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        assert request.headers["Authorization"] == f"Bearer {token}"
        assert request.url.path == "/api/credential"
        assert dict(request.url.params) == {"project_id": "project-1"}
        payload = json.loads(request.content)
        assert payload["type"] == "runtime_env"
        assert json.loads(payload["data"]) == {"MYSQL_PASSWORD": "runtime-test-secret"}
        return httpx.Response(
            201,
            json={"id": "credential-1", "name": "runtime", "type": "runtime_env", "created_at": "2026-07-26T00:00:00Z"},
        )

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        created = await OrbitClient(settings, http_client).create_runtime_env_credential(
            "project-1", "runtime", {"MYSQL_PASSWORD": "runtime-test-secret"}
        )
    metadata = runtime_env_credential_metadata(created)
    assert metadata == {
        "id": "credential-1",
        "name": "runtime",
        "type": "runtime_env",
        "created_at": "2026-07-26T00:00:00Z",
    }
    assert "runtime-test-secret" not in json.dumps(metadata)
    assert len(requests) == 1
