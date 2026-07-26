from __future__ import annotations

import pytest

from pomelo_orbit_mcp.tools.runtime import _gateway_target

from .conftest import make_settings


class StandardApplicationClient:
    async def get_application(self, application_id: str) -> dict[str, str]:
        return {"id": application_id, "kind": "standard"}


class GatewayApplicationClient:
    async def get_application(self, application_id: str) -> dict[str, str]:
        return {"id": application_id, "kind": "gateway", "code": "traefik"}

    async def list_application_services(self, application_id: str) -> list[dict[str, str]]:
        return [{"id": "service-1", "application_id": application_id, "instance_key": "default"}]


@pytest.mark.asyncio
async def test_gateway_target_rejects_non_gateway_application(tmp_path) -> None:
    with pytest.raises(ValueError, match="must identify an Orbit Gateway Application"):
        await _gateway_target(StandardApplicationClient(), make_settings(tmp_path), "application-1", "default")


@pytest.mark.asyncio
async def test_gateway_target_resolves_gateway_workspace(tmp_path) -> None:
    target = await _gateway_target(GatewayApplicationClient(), make_settings(tmp_path), "gateway-1", "default")

    assert target.application_code == "traefik"
    assert target.compose_project == "traefik-default"
