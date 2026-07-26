from __future__ import annotations

import base64
import json

import httpx
import pytest

from pomelo_orbit_mcp.orbit_client import OrbitAPIError, OrbitClient

from .conftest import make_settings


def make_jwt(exp: int = 4_102_444_800) -> str:
    payload = base64.urlsafe_b64encode(json.dumps({"exp": exp}).encode()).decode().rstrip("=")
    return f"header.{payload}.signature"


@pytest.mark.asyncio
async def test_client_logs_in_then_calls_existing_orbit_route(tmp_path) -> None:
    requests: list[httpx.Request] = []
    token = make_jwt()

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        if request.url.path == "/api/auth/csrf-token":
            return httpx.Response(200, json={"token": "csrf"})
        if request.url.path == "/api/auth/login":
            assert json.loads(request.content) == {
                "username": "tester",
                "password": "not-for-output",
                "csrf_token": "csrf",
            }
            return httpx.Response(200, json={"access_token": token, "token_type": "Bearer"})
        assert request.headers["Authorization"] == f"Bearer {token}"
        return httpx.Response(200, json={"items": [{"id": "project-1"}]})

    settings = make_settings(tmp_path)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        client = OrbitClient(settings, http_client)
        assert await client.list_projects() == [{"id": "project-1"}]
    assert [request.url.path for request in requests] == ["/api/auth/csrf-token", "/api/auth/login", "/api/project"]
    assert settings.jwt_cache_path.read_text(encoding="utf-8").startswith("POMELO_ORBIT_JWT=")


@pytest.mark.asyncio
async def test_client_reauthenticates_once_after_401(tmp_path) -> None:
    stale = make_jwt()
    fresh = make_jwt(4_102_444_801)
    project_calls = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal project_calls
        if request.url.path == "/api/project":
            project_calls += 1
            if project_calls == 1:
                assert request.headers["Authorization"] == f"Bearer {stale}"
                return httpx.Response(401, json={"detail": "expired"})
            assert request.headers["Authorization"] == f"Bearer {fresh}"
            return httpx.Response(200, json={"items": []})
        if request.url.path == "/api/auth/csrf-token":
            return httpx.Response(200, json={"token": "csrf"})
        if request.url.path == "/api/auth/login":
            return httpx.Response(200, json={"access_token": fresh})
        raise AssertionError(request.url.path)

    settings = make_settings(tmp_path, jwt_from_environment=stale)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        client = OrbitClient(settings, http_client)
        assert await client.list_projects() == []
    assert project_calls == 2


@pytest.mark.asyncio
async def test_authentication_error_never_exposes_password(tmp_path) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/api/auth/csrf-token":
            return httpx.Response(200, json={"token": "csrf"})
        return httpx.Response(400, json={"detail": "not-for-output"})

    settings = make_settings(tmp_path)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        with pytest.raises(OrbitAPIError) as raised:
            await OrbitClient(settings, http_client).list_projects()
    assert "not-for-output" not in str(raised.value)


@pytest.mark.asyncio
async def test_list_applications_uses_project_and_kind_filters(tmp_path) -> None:
    token = make_jwt()

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/api/application"
        assert dict(request.url.params) == {"project_id": "project-1", "per_page": "100", "kind": "standard"}
        assert request.headers["Authorization"] == f"Bearer {token}"
        return httpx.Response(200, json={"items": [{"id": "app-1"}]})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        assert await OrbitClient(settings, http_client).list_applications("project-1", "standard") == [{"id": "app-1"}]
