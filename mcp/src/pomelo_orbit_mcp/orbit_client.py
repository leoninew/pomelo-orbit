"""Authenticated client for the existing Pomelo Orbit HTTP API."""

from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable, Mapping
from dataclasses import dataclass
from typing import Any

import httpx

from .settings import JWTCache, Settings, is_jwt_valid

TERMINAL_DEPLOYMENT_STATUSES = {"ran_to_completion", "faulted", "canceled"}


@dataclass(frozen=True)
class OrbitAPIError(RuntimeError):
    method: str
    path: str
    status_code: int | None
    message: str
    code: str | None = None
    request_id: str | None = None

    def __str__(self) -> str:
        status = str(self.status_code) if self.status_code is not None else "network"
        context = [status]
        if self.code is not None:
            context.append(self.code)
        if self.request_id is not None:
            context.append(f"request_id={self.request_id}")
        return f"Orbit API {self.method} {self.path} failed ({', '.join(context)}): {self.message}"


class OrbitClient:
    """Uses an existing username/password login and short-lived Bearer JWT."""

    def __init__(self, settings: Settings, http_client: httpx.AsyncClient | None = None) -> None:
        self.settings = settings
        self._cache = JWTCache(settings)
        self._token: str | None = None
        self._client = http_client or httpx.AsyncClient(
            base_url=settings.orbit_url, timeout=30.0, follow_redirects=False
        )
        self._owns_client = http_client is None

    async def aclose(self) -> None:
        if self._owns_client:
            await self._client.aclose()

    async def _authenticate(self) -> str:
        csrf = await self._request_without_auth("GET", "/api/auth/csrf-token")
        csrf_token = str(csrf.get("token") or "")
        if not csrf_token:
            raise OrbitAPIError("GET", "/api/auth/csrf-token", 200, "response did not contain a CSRF token")
        login = await self._request_without_auth(
            "POST",
            "/api/auth/login",
            json_body={
                "username": self.settings.username,
                "password": self.settings.password,
                "csrf_token": csrf_token,
            },
        )
        token = str(login.get("access_token") or "")
        if not is_jwt_valid(token):
            raise OrbitAPIError("POST", "/api/auth/login", 200, "response did not contain a valid JWT")
        if self.settings.jwt_from_environment is None:
            self._cache.write(token)
        self._token = token
        return token

    async def _bearer_token(self, *, force_login: bool = False) -> str:
        if not force_login and is_jwt_valid(self._token):
            return self._token  # type: ignore[return-value]
        cached = None if force_login else self._cache.read()
        if is_jwt_valid(cached):
            self._token = cached
            return cached  # type: ignore[return-value]
        return await self._authenticate()

    async def _request_without_auth(
        self,
        method: str,
        path: str,
        *,
        params: Mapping[str, Any] | None = None,
        json_body: Mapping[str, Any] | None = None,
    ) -> dict[str, Any]:
        try:
            response = await self._client.request(method, path, params=params, json=json_body)
        except httpx.RequestError as error:
            raise OrbitAPIError(method, path, None, "network request failed") from error
        if response.is_error:
            # Login responses may contain sensitive echo data on a proxy or a
            # non-standard error path, so never include their body.
            if path.startswith("/api/auth/"):
                raise OrbitAPIError(method, path, response.status_code, "authentication request failed")
            message, code, request_id = _error_summary(response)
            raise OrbitAPIError(method, path, response.status_code, message, code, request_id)
        return _json_object(response, method, path)

    async def request(
        self,
        method: str,
        path: str,
        *,
        params: Mapping[str, Any] | None = None,
        json_body: Mapping[str, Any] | None = None,
    ) -> dict[str, Any]:
        for retry_after_refresh in (False, True):
            token = await self._bearer_token(force_login=retry_after_refresh)
            try:
                response = await self._client.request(
                    method,
                    path,
                    params=params,
                    json=json_body,
                    headers={"Authorization": f"Bearer {token}"},
                )
            except httpx.RequestError as error:
                raise OrbitAPIError(method, path, None, "network request failed") from error
            if response.status_code == 401 and not retry_after_refresh:
                self._token = None
                self._cache.clear()
                continue
            if response.is_error:
                message, code, request_id = _error_summary(response)
                raise OrbitAPIError(method, path, response.status_code, message, code, request_id)
            return _json_object(response, method, path)
        raise OrbitAPIError(method, path, 401, "authentication retry failed")

    async def list_projects(self) -> list[dict[str, Any]]:
        return _items(await self.request("GET", "/api/project"))

    async def list_applications(self, project_id: str, kind: str | None = None) -> list[dict[str, Any]]:
        params: dict[str, Any] = {"project_id": project_id, "per_page": 100}
        if kind:
            params["kind"] = kind
        return _items(await self.request("GET", "/api/application", params=params))

    async def list_gateways(self, project_id: str) -> list[dict[str, Any]]:
        return _items(await self.request("GET", "/api/gateway", params={"project_id": project_id, "per_page": 100}))

    async def create_gateway(self, project_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", "/api/gateway", params={"project_id": project_id}, json_body=payload)

    async def get_gateway(self, gateway_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/gateway/{gateway_id}")

    async def update_gateway(self, gateway_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("PUT", f"/api/gateway/{gateway_id}", json_body=payload)

    async def create_application(self, project_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", "/api/application", params={"project_id": project_id}, json_body=payload)

    async def get_application(self, application_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/application/{application_id}")

    async def delete_application(self, application_id: str, remove_dir: bool) -> None:
        await self.request(
            "DELETE", f"/api/application/{application_id}", params={"remove_dir": str(remove_dir).lower()}
        )

    async def list_versions(self, application_id: str) -> list[dict[str, Any]]:
        return _items(await self.request("GET", f"/api/application/{application_id}/version", params={"per_page": 100}))

    async def get_version(self, version_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/version/{version_id}")

    async def get_version_component(self, version_id: str, component_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/version/{version_id}/component/{component_id}")

    async def create_version_component(self, version_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", f"/api/version/{version_id}/component", json_body=payload)

    async def create_version(self, application_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        body = {"application_id": application_id, **payload}
        return await self.request("POST", f"/api/application/{application_id}/version", json_body=body)

    async def update_version(self, version_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("PUT", f"/api/version/{version_id}", json_body=payload)

    async def update_version_component_basic(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request("PUT", f"/api/version/{version_id}/component/{component_id}/basic", json_body=payload)

    async def update_version_component_runtime(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request(
            "PUT", f"/api/version/{version_id}/component/{component_id}/runtime", json_body=payload
        )

    async def update_version_component_ports(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request("PUT", f"/api/version/{version_id}/component/{component_id}/ports", json_body=payload)

    async def update_version_component_env(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request("PUT", f"/api/version/{version_id}/component/{component_id}/env", json_body=payload)

    async def update_version_component_mounts(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request(
            "PUT", f"/api/version/{version_id}/component/{component_id}/mounts", json_body=payload
        )

    async def update_version_component_dependencies(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request(
            "PUT", f"/api/version/{version_id}/component/{component_id}/dependencies", json_body=payload
        )

    async def update_version_component_advanced(
        self, version_id: str, component_id: str, payload: Mapping[str, Any]
    ) -> dict[str, Any]:
        return await self.request(
            "PUT", f"/api/version/{version_id}/component/{component_id}/advanced", json_body=payload
        )

    async def publish_version(self, version_id: str) -> dict[str, Any]:
        return await self.request("POST", f"/api/version/{version_id}/publish", json_body={})

    async def delete_version(self, version_id: str) -> None:
        await self.request("DELETE", f"/api/version/{version_id}")

    async def preview_version(self, version_id: str, instance_key: str) -> dict[str, Any]:
        return await self.request(
            "POST",
            f"/api/version/{version_id}/preview",
            json_body={"instance_key": instance_key},
        )

    async def list_application_services(self, application_id: str) -> list[dict[str, Any]]:
        return _items(await self.request("GET", f"/api/application/{application_id}/service"))

    async def get_service(self, service_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/service/{service_id}")

    async def create_service(
        self,
        application_id: str,
        version_id: str,
        instance_key: str,
        runtime_config: Mapping[str, str],
    ) -> dict[str, Any]:
        return await self.request(
            "POST",
            "/api/service",
            json_body={
                "application_id": application_id,
                "version_id": version_id,
                "instance_key": instance_key,
                "runtime_config": dict(runtime_config),
            },
        )

    async def get_service_runtime_config(self, service_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/service/{service_id}/runtime-config")

    async def update_service_runtime_config(self, service_id: str, runtime_config: Mapping[str, str]) -> dict[str, Any]:
        return await self.request(
            "PUT", f"/api/service/{service_id}/runtime-config", json_body={"runtime_config": dict(runtime_config)}
        )

    async def deploy_application(self, application_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", f"/api/application/{application_id}/deploy", json_body=payload)

    async def stop_application(self, application_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", f"/api/application/{application_id}/stop", json_body=payload)

    async def restart_application(self, application_id: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("POST", f"/api/application/{application_id}/restart", json_body=payload)

    async def get_deployment(self, deployment_id: str) -> dict[str, Any]:
        return await self.request("GET", f"/api/deployment/{deployment_id}")

    async def get_deployment_logs(self, deployment_id: str, offset: int = 0) -> dict[str, Any]:
        return await self.request("GET", f"/api/deployment/{deployment_id}/logs", params={"offset": offset})

    async def wait_deployment(
        self,
        deployment_id: str,
        timeout_seconds: int | None = None,
        sleep: Callable[[float], Awaitable[None]] = asyncio.sleep,
    ) -> dict[str, Any]:
        timeout = self.settings.wait_timeout_seconds if timeout_seconds is None else timeout_seconds
        loop = asyncio.get_running_loop()
        deadline = loop.time() + timeout
        latest = await self.get_deployment(deployment_id)
        while latest.get("status") not in TERMINAL_DEPLOYMENT_STATUSES and loop.time() < deadline:
            await sleep(min(2, max(0.0, deadline - loop.time())))
            latest = await self.get_deployment(deployment_id)
        return {
            "deployment": latest,
            "timed_out": latest.get("status") not in TERMINAL_DEPLOYMENT_STATUSES,
            "timeout_seconds": timeout,
        }


def _items(value: Mapping[str, Any]) -> list[dict[str, Any]]:
    items = value.get("items")
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def _json_object(response: httpx.Response, method: str, path: str) -> dict[str, Any]:
    if response.status_code == 204 or not response.content:
        return {}
    try:
        value = response.json()
    except ValueError as error:
        raise OrbitAPIError(method, path, response.status_code, "response was not JSON") from error
    if not isinstance(value, dict):
        raise OrbitAPIError(method, path, response.status_code, "response JSON was not an object")
    return value


def _error_summary(response: httpx.Response) -> tuple[str, str | None, str | None]:
    try:
        value = response.json()
    except ValueError:
        return "request failed due to invalid error contract", None, None
    if not isinstance(value, dict):
        return "request failed due to invalid error contract", None, None
    code = value.get("code")
    message = value.get("error")
    request_id = value.get("requestId")
    if not isinstance(code, str) or not code.strip():
        return "request failed due to invalid error contract", None, None
    if not isinstance(message, str) or not message.strip():
        return "request failed due to invalid error contract", None, None
    if not isinstance(request_id, str) or not request_id.strip():
        return "request failed due to invalid error contract", None, None
    return message, code, request_id
