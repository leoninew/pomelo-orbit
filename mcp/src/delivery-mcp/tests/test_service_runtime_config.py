from __future__ import annotations

import json

import httpx
import pytest

from pomelo_delivery_mcp.orbit_client import OrbitClient

from .conftest import make_settings


@pytest.mark.asyncio
async def test_service_client_creates_only_the_version_binding(tmp_path) -> None:
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
        }
        return httpx.Response(
            201,
            json={"id": "service-1", "application_id": "application-1", "version_id": "version-1", "status": "stopped"},
        )

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        created = await OrbitClient(settings, http_client).create_service("application-1", "version-1", "default")
    assert created["id"] == "service-1"
    assert len(requests) == 1


@pytest.mark.asyncio
async def test_service_component_overlay_client_targets_declared_component(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == f"Bearer {token}"
        assert request.method == "PUT"
        assert request.url.path == "/api/service/service-1/component/component-1"
        assert json.loads(request.content) == {
            "env": [{"key": "MYSQL_PASSWORD", "value": "runtime-test-value", "state": "override"}],
            "mounts": [{"target": "/data", "state": "deleted"}],
            "endpoints": [],
        }
        return httpx.Response(200, json={"id": "service-1"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        updated = await OrbitClient(settings, http_client).update_service_component_overlay(
            "service-1",
            "component-1",
            {
                "env": [{"key": "MYSQL_PASSWORD", "value": "runtime-test-value", "state": "override"}],
                "mounts": [{"target": "/data", "state": "deleted"}],
                "endpoints": [],
            },
        )
    assert updated["id"] == "service-1"


@pytest.mark.asyncio
async def test_service_environment_client_replaces_the_service_collection(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == f"Bearer {token}"
        assert request.method == "PUT"
        assert request.url.path == "/api/service/service-1/env"
        assert json.loads(request.content) == {
            "env": [
                {"key": "SHARED_DATABASE_PASSWORD", "value": "runtime-test-value"},
                {"key": "SHARED_DATABASE_USER", "value": "orbit"},
            ]
        }
        return httpx.Response(200, json={"id": "service-1"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        updated = await OrbitClient(settings, http_client).update_service_env(
            "service-1",
            {
                "env": [
                    {"key": "SHARED_DATABASE_PASSWORD", "value": "runtime-test-value"},
                    {"key": "SHARED_DATABASE_USER", "value": "orbit"},
                ]
            },
        )
    assert updated["id"] == "service-1"


@pytest.mark.asyncio
async def test_service_basic_client_sends_version_and_instance_key(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.headers["Authorization"] == f"Bearer {token}"
        assert request.method == "PUT"
        assert request.url.path == "/api/service/service-1/basic"
        assert json.loads(request.content) == {"version_id": "version-2", "instance_key": "default"}
        return httpx.Response(200, json={"id": "service-1", "version_id": "version-2"})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        updated = await OrbitClient(settings, http_client).update_service_basic("service-1", "version-2", "default")
    assert updated["version_id"] == "version-2"


@pytest.mark.asyncio
async def test_service_preview_and_deploy_clients_target_the_saved_service(tmp_path) -> None:
    token = "header.eyJleHAiOjQxMDI0NDQ4MDB9.signature"
    requests: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        assert request.headers["Authorization"] == f"Bearer {token}"
        if request.url.path == "/api/service/service-1/preview":
            assert request.method == "POST"
            assert json.loads(request.content) == {}
            return httpx.Response(200, json={"compose_yaml": "services: {}"})
        assert request.url.path == "/api/service/service-1/deploy"
        assert request.method == "POST"
        assert json.loads(request.content) == {"force_recreate": True}
        return httpx.Response(202, json={"deployment_id": "deployment-1", "warnings": ["Gateway entrypoint missing"]})

    settings = make_settings(tmp_path, jwt_from_environment=token)
    async with httpx.AsyncClient(base_url=settings.orbit_url, transport=httpx.MockTransport(handler)) as http_client:
        client = OrbitClient(settings, http_client)
        preview = await client.preview_service("service-1")
        deployment = await client.deploy_service("service-1", True)

    assert preview == {"compose_yaml": "services: {}"}
    assert deployment == {"deployment_id": "deployment-1", "warnings": ["Gateway entrypoint missing"]}
    assert [request.url.path for request in requests] == [
        "/api/service/service-1/preview",
        "/api/service/service-1/deploy",
    ]
