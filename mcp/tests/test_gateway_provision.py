from __future__ import annotations

import pytest

from pomelo_orbit_mcp.tools.orbit import provision_gateway


class ProvisionClient:
    def __init__(
        self,
        *,
        gateways: list[dict[str, str]] | None = None,
        versions: list[dict[str, str]] | None = None,
        services: list[dict[str, str]] | None = None,
        deployment_status: str = "ran_to_completion",
        timed_out: bool = False,
    ) -> None:
        self.gateways = gateways or []
        self.versions = versions or [{"id": "version-1", "status": "unpublished"}]
        self.services = services or []
        self.deployment_status = deployment_status
        self.timed_out = timed_out
        self.calls: list[tuple[str, object]] = []

    async def list_gateways(self, project_id: str) -> list[dict[str, str]]:
        self.calls.append(("list_gateways", project_id))
        return self.gateways

    async def create_gateway(self, project_id: str, payload: dict[str, str]) -> dict[str, str]:
        self.calls.append(("create_gateway", {"project_id": project_id, "payload": payload}))
        return {"id": "gateway-1", "code": "traefik"}

    async def list_versions(self, application_id: str) -> list[dict[str, str]]:
        self.calls.append(("list_versions", application_id))
        return self.versions

    async def list_application_services(self, application_id: str) -> list[dict[str, str]]:
        self.calls.append(("list_application_services", application_id))
        return self.services

    async def publish_version(self, version_id: str) -> dict[str, str]:
        self.calls.append(("publish_version", version_id))
        return {"id": version_id, "status": "published"}

    async def create_service(
        self,
        application_id: str,
        version_id: str,
        instance_key: str,
        runtime_config: dict[str, str],
        exposes: list[dict[str, object]],
    ) -> dict[str, str]:
        self.calls.append(
            (
                "create_service",
                {
                    "application_id": application_id,
                    "version_id": version_id,
                    "instance_key": instance_key,
                    "runtime_config": runtime_config,
                    "exposes": exposes,
                },
            )
        )
        return {"id": "service-1", "instance_key": instance_key, "version_id": version_id}

    async def get_service(self, service_id: str) -> dict[str, object]:
        self.calls.append(("get_service", service_id))
        return next(service for service in self.services if service["id"] == service_id)

    async def update_service_basic(self, service_id: str, version_id: str, instance_key: str) -> dict[str, object]:
        self.calls.append(
            (
                "update_service_basic",
                {"service_id": service_id, "version_id": version_id, "instance_key": instance_key},
            )
        )
        return {"id": service_id, "version_id": version_id, "instance_key": instance_key}

    async def deploy_service(self, service_id: str, force_recreate: bool) -> dict[str, str]:
        self.calls.append(("deploy_service", {"service_id": service_id, "force_recreate": force_recreate}))
        return {"deployment_id": "deployment-1"}

    async def wait_deployment(self, deployment_id: str, timeout_seconds: int | None) -> dict[str, object]:
        self.calls.append(("wait_deployment", {"deployment_id": deployment_id, "timeout_seconds": timeout_seconds}))
        return {
            "deployment": {"id": deployment_id, "status": self.deployment_status},
            "timed_out": self.timed_out,
            "timeout_seconds": timeout_seconds,
        }


class ProvisionRuntime:
    def __init__(self, *, network_driver: str = "bridge") -> None:
        self.network_driver = network_driver
        self.calls: list[str] = []

    async def external_network_inspect(self, network_name: str) -> dict[str, str]:
        self.calls.append(network_name)
        return {"name": network_name, "driver": self.network_driver, "scope": "local"}


@pytest.mark.asyncio
async def test_provision_gateway_creates_publishes_deploys_and_confirms_network() -> None:
    client = ProvisionClient()
    runtime = ProvisionRuntime()

    result = await provision_gateway(
        client,  # type: ignore[arg-type]
        runtime,  # type: ignore[arg-type]
        project_id="project-1",
        instance_key="default",
        runtime_config={"TOKEN": "secret-value"},
        force_recreate=True,
        timeout_seconds=30,
    )

    assert result["ready"] is True
    assert result["resource_ids"] == {
        "gateway_id": "gateway-1",
        "application_id": "gateway-1",
        "version_id": "version-1",
        "service_id": "service-1",
        "deployment_id": "deployment-1",
    }
    assert result["created"] == {"gateway": True, "service": True, "version_published": True}
    assert result["network"] == {
        "name": "traefik",
        "driver": "bridge",
        "scope": "local",
        "ready": True,
        "status": "ready",
    }
    assert "secret-value" not in str(result)
    assert [call[0] for call in client.calls] == [
        "list_gateways",
        "create_gateway",
        "list_versions",
        "list_application_services",
        "publish_version",
        "create_service",
        "deploy_service",
        "wait_deployment",
    ]
    assert runtime.calls == ["traefik"]


@pytest.mark.asyncio
async def test_provision_gateway_reuses_one_published_gateway_service_and_version() -> None:
    client = ProvisionClient(
        gateways=[{"id": "gateway-1", "code": "traefik"}],
        versions=[{"id": "version-1", "status": "published"}],
        services=[{"id": "service-1", "instance_key": "default", "version_id": "version-1"}],
    )
    runtime = ProvisionRuntime()

    result = await provision_gateway(
        client,  # type: ignore[arg-type]
        runtime,  # type: ignore[arg-type]
        project_id="project-1",
        instance_key="default",
        runtime_config=None,
        force_recreate=False,
        timeout_seconds=None,
    )

    assert result["ready"] is True
    assert result["created"] == {"gateway": False, "service": False, "version_published": False}
    assert "create_gateway" not in [call[0] for call in client.calls]
    assert "publish_version" not in [call[0] for call in client.calls]
    assert "create_service" not in [call[0] for call in client.calls]


@pytest.mark.asyncio
async def test_provision_gateway_rejects_duplicate_gateway_matches_before_writes() -> None:
    client = ProvisionClient(gateways=[{"id": "gateway-1", "code": "traefik"}, {"id": "gateway-2", "code": "traefik"}])

    with pytest.raises(ValueError, match="multiple traefik Gateways"):
        await provision_gateway(
            client,  # type: ignore[arg-type]
            ProvisionRuntime(),  # type: ignore[arg-type]
            project_id="project-1",
            instance_key="default",
            runtime_config=None,
            force_recreate=False,
            timeout_seconds=None,
        )

    assert client.calls == [("list_gateways", "project-1")]


@pytest.mark.asyncio
async def test_provision_gateway_rejects_runtime_config_when_reusing_service() -> None:
    client = ProvisionClient(
        gateways=[{"id": "gateway-1", "code": "traefik"}],
        services=[{"id": "service-1", "instance_key": "default", "version_id": "version-1"}],
    )

    with pytest.raises(ValueError, match="runtime_config cannot"):
        await provision_gateway(
            client,  # type: ignore[arg-type]
            ProvisionRuntime(),  # type: ignore[arg-type]
            project_id="project-1",
            instance_key="default",
            runtime_config={"TOKEN": "secret-value"},
            force_recreate=False,
            timeout_seconds=None,
        )

    assert [call[0] for call in client.calls] == ["list_gateways", "list_versions", "list_application_services"]


@pytest.mark.asyncio
async def test_provision_gateway_stops_before_network_check_when_deployment_is_unsuccessful() -> None:
    client = ProvisionClient(deployment_status="faulted")
    runtime = ProvisionRuntime()

    result = await provision_gateway(
        client,  # type: ignore[arg-type]
        runtime,  # type: ignore[arg-type]
        project_id="project-1",
        instance_key="default",
        runtime_config=None,
        force_recreate=False,
        timeout_seconds=None,
    )

    assert result["ready"] is False
    assert result["network"] == {"name": "traefik", "ready": False, "status": "not_checked"}
    assert runtime.calls == []
