from __future__ import annotations

import base64
import json

import httpx
import pytest

from pomelo_delivery_mcp.orbit_client import OrbitAPIError, OrbitClient

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
                return httpx.Response(
                    401, json={"code": "unauthorized", "error": "Expired session.", "requestId": "request-1"}
                )
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
async def test_client_preserves_structured_error_contract(tmp_path) -> None:
    token = make_jwt()

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == f"Bearer {token}"
        return httpx.Response(
            409,
            json={
                "code": "version_already_published",
                "error": "Version is already published.",
                "requestId": "request-2",
            },
        )

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        with pytest.raises(OrbitAPIError) as raised:
            await OrbitClient(settings, http_client).list_projects()
    error = raised.value
    assert error.status_code == 409
    assert error.code == "version_already_published"
    assert error.request_id == "request-2"
    assert error.message == "Version is already published."
    assert "request_id=request-2" in str(error)


@pytest.mark.asyncio
async def test_client_rejects_old_or_invalid_error_contract_without_exposing_body(tmp_path) -> None:
    token = make_jwt()

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == f"Bearer {token}"
        return httpx.Response(500, json={"detail": "database password=not-for-output"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        with pytest.raises(OrbitAPIError) as raised:
            await OrbitClient(settings, http_client).list_projects()
    error = raised.value
    assert error.code is None
    assert error.request_id is None
    assert error.message == "request failed due to invalid error contract"
    assert "not-for-output" not in str(error)


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


@pytest.mark.asyncio
async def test_client_maps_gateway_read_and_create_routes(tmp_path) -> None:
    token = make_jwt()
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        assert request.headers["Authorization"] == f"Bearer {token}"
        if request.method == "GET":
            assert request.url.path == "/api/gateway"
            assert dict(request.url.params) == {"project_id": "project-1", "per_page": "100"}
            return httpx.Response(200, json={"items": [{"id": "gateway-1"}]})
        if request.method == "POST":
            assert request.url.path == "/api/gateway"
            assert dict(request.url.params) == {"project_id": "project-1"}
            assert json.loads(request.content) == {"project_id": "project-1", "code": "traefik", "name": "Traefik"}
            return httpx.Response(201, json={"id": "gateway-1", "kind": "gateway"})
        assert request.method == "PUT"
        assert request.url.path == "/api/gateway/gateway-1"
        assert json.loads(request.content) == {"rest_api_url": "http://localhost:8080"}
        return httpx.Response(200, json={"id": "gateway-1", "rest_api_url": "http://localhost:8080"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        client = OrbitClient(settings, http_client)
        assert await client.list_gateways("project-1") == [{"id": "gateway-1"}]
        assert await client.create_gateway(
            "project-1", {"project_id": "project-1", "code": "traefik", "name": "Traefik"}
        ) == {
            "id": "gateway-1",
            "kind": "gateway",
        }
        assert await client.update_gateway("gateway-1", {"rest_api_url": "http://localhost:8080"}) == {
            "id": "gateway-1",
            "rest_api_url": "http://localhost:8080",
        }
    assert [request.method for request in requests] == ["GET", "POST", "PUT"]


@pytest.mark.asyncio
async def test_client_maps_component_update_sections(tmp_path) -> None:
    token = make_jwt()
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        assert request.headers["Authorization"] == f"Bearer {token}"
        return httpx.Response(200, json={"id": "component-1"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        client = OrbitClient(settings, http_client)
        assert await client.get_version_component("version-1", "component-1") == {"id": "component-1"}
        assert await client.create_version_component(
            "version-1", {"name": "web", "image": "nginx:1.27", "command": "nginx -g 'daemon off;'"}
        ) == {"id": "component-1"}
        assert await client.update_version_component_basic(
            "version-1", "component-1", {"name": "web", "image": "nginx:1.27", "command": "nginx -g 'daemon off;'"}
        ) == {"id": "component-1"}
        assert await client.update_version_component_runtime("version-1", "component-1", {"healthcheck": None}) == {
            "id": "component-1"
        }
        assert await client.update_version_component_ports(
            "version-1", "component-1", {"ports": [{"host_port": 8080, "container_port": 80}]}
        ) == {"id": "component-1"}
        assert await client.update_version_component_env(
            "version-1", "component-1", {"env": [{"key": "MODE", "value": "production"}]}
        ) == {"id": "component-1"}
        assert await client.update_version_component_mounts(
            "version-1", "component-1", {"mounts": [{"source_type": "directory", "source": "data", "target": "/data"}]}
        ) == {"id": "component-1"}
        assert await client.update_version_component_dependencies(
            "version-1", "component-1", {"dependencies": [{"name": "database", "condition": "service_healthy"}]}
        ) == {"id": "component-1"}
        assert await client.update_version_component_devices(
            "version-1", "component-1", {"devices": [{"driver": "nvidia", "count": "all", "capabilities": ["gpu"]}]}
        ) == {"id": "component-1"}
        assert await client.update_version_component_advanced(
            "version-1", "component-1", {"resources": {"limit_memory": "512m"}, "tmpfs": [], "ulimits": []}
        ) == {"id": "component-1"}

    assert [(request.method, request.url.path) for request in requests] == [
        ("GET", "/api/version/version-1/component/component-1"),
        ("POST", "/api/version/version-1/component"),
        ("PUT", "/api/version/version-1/component/component-1/basic"),
        ("PUT", "/api/version/version-1/component/component-1/runtime"),
        ("PUT", "/api/version/version-1/component/component-1/ports"),
        ("PUT", "/api/version/version-1/component/component-1/env"),
        ("PUT", "/api/version/version-1/component/component-1/mounts"),
        ("PUT", "/api/version/version-1/component/component-1/dependencies"),
        ("PUT", "/api/version/version-1/component/component-1/devices"),
        ("PUT", "/api/version/version-1/component/component-1/advanced"),
    ]
    assert [json.loads(request.content) if request.content else None for request in requests] == [
        None,
        {"name": "web", "image": "nginx:1.27", "command": "nginx -g 'daemon off;'"},
        {"name": "web", "image": "nginx:1.27", "command": "nginx -g 'daemon off;'"},
        {"healthcheck": None},
        {"ports": [{"host_port": 8080, "container_port": 80}]},
        {"env": [{"key": "MODE", "value": "production"}]},
        {"mounts": [{"source_type": "directory", "source": "data", "target": "/data"}]},
        {"dependencies": [{"name": "database", "condition": "service_healthy"}]},
        {"devices": [{"driver": "nvidia", "count": "all", "capabilities": ["gpu"]}]},
        {"resources": {"limit_memory": "512m"}, "tmpfs": [], "ulimits": []},
    ]
