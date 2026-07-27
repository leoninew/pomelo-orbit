from __future__ import annotations

import json

import httpx
import pytest

from pomelo_orbit_mcp.orbit_client import OrbitClient

from .conftest import make_settings


@pytest.mark.asyncio
async def test_service_runtime_config_client_uses_service_aggregate(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        assert request.headers["Authorization"] == f"Bearer {token}"
        assert request.url.path == "/api/service"
        payload = json.loads(request.content)
        assert payload == {
            "application_id": "application-1",
            "version_id": "version-1",
            "instance_key": "default",
            "runtime_config": {"MYSQL_PASSWORD": "runtime-test-value"},
        }
        return httpx.Response(
            201,
            json={"id": "service-1", "application_id": "application-1", "version_id": "version-1", "status": "stopped"},
        )

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        created = await OrbitClient(settings, http_client).create_service(
            "application-1", "version-1", "default", {"MYSQL_PASSWORD": "runtime-test-value"}
        )
    assert created["id"] == "service-1"
    assert len(requests) == 1
