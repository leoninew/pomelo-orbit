from __future__ import annotations

import pytest

from pomelo_orbit_mcp.tools.orbit import _update_component_advanced_section


class AdvancedComponentClient:
    def __init__(self) -> None:
        self.update_payload: dict[str, object] | None = None

    async def get_version_component(self, version_id: str, component_id: str) -> dict[str, object]:
        assert (version_id, component_id) == ("version-1", "component-1")
        return {
            "resources": {"limit_memory": "512m"},
            "tmpfs": [{"target": "/tmp", "size_bytes": 67_108_864, "mode": "1777"}],
            "ulimits": [{"name": "nofile", "soft": 65_535, "hard": 65_535}],
        }

    async def update_version_component_advanced(
        self, version_id: str, component_id: str, payload: dict[str, object]
    ) -> dict[str, object]:
        assert (version_id, component_id) == ("version-1", "component-1")
        self.update_payload = payload
        return {"id": component_id}


@pytest.mark.asyncio
async def test_advanced_section_update_preserves_other_sections() -> None:
    client = AdvancedComponentClient()
    value = [{"target": "/run", "size_bytes": 16_777_216, "mode": "0755"}]

    body, updated = await _update_component_advanced_section(client, "version-1", "component-1", "tmpfs", value)

    assert body == {
        "resources": {"limit_memory": "512m"},
        "tmpfs": value,
        "ulimits": [{"name": "nofile", "soft": 65_535, "hard": 65_535}],
    }
    assert client.update_payload == body
    assert updated == {"id": "component-1"}
