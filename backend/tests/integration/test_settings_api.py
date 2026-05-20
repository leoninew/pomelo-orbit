import pytest


class FakeSettingService:
    def __init__(self) -> None:
        self.items = [
            {
                "key": "jwt__expire_minutes",
                "value": 30,
                "default": 30,
                "is_overridden": False,
            }
        ]
        self.updated: tuple[str, object] | None = None
        self.reset_keys: list[str] = []

    def get_config(self) -> dict:
        return {"items": self.items}

    def update_config(self, key: str, value: object) -> dict:
        self.updated = (key, value)
        self.items = [
            {
                "key": key,
                "value": value,
                "default": 30,
                "is_overridden": True,
            }
        ]
        return {"items": self.items}

    def reset_config(self, keys: list[str]) -> dict:
        self.reset_keys = keys
        self.items = [
            {
                "key": key,
                "value": 30,
                "default": 30,
                "is_overridden": False,
            }
            for key in keys
        ]
        return {"items": self.items}


@pytest.fixture
def fake_setting_service():
    from pomelo_orbit.application.cd.di import get_setting_service
    from pomelo_orbit.main import app

    service = FakeSettingService()
    app.dependency_overrides[get_setting_service] = lambda: service
    yield service
    app.dependency_overrides.pop(get_setting_service, None)


def test_get_config_with_setting_read(auth_client, fake_setting_service):
    response = auth_client.get("/api/settings/config")

    assert response.status_code == 200
    body = response.json()
    assert body["items"][0]["key"] == "jwt__expire_minutes"


def test_get_config_without_setting_read_returns_403(user_write_client, fake_setting_service):
    response = user_write_client.get("/api/settings/config")

    assert response.status_code == 403


def test_update_config_with_setting_write(auth_client, fake_setting_service):
    response = auth_client.put(
        "/api/settings/config",
        json={"key": "jwt__expire_minutes", "value": 60},
    )

    assert response.status_code == 200
    assert fake_setting_service.updated == ("jwt__expire_minutes", 60)
    body = response.json()
    assert body["items"][0]["value"] == 60
    assert body["items"][0]["is_overridden"] is True


def test_reset_config_with_setting_write(auth_client, fake_setting_service):
    response = auth_client.request(
        "DELETE",
        "/api/settings/config",
        json={"keys": ["jwt__expire_minutes"]},
    )

    assert response.status_code == 200
    assert fake_setting_service.reset_keys == ["jwt__expire_minutes"]
    body = response.json()
    assert body["items"][0]["is_overridden"] is False


def test_setting_read_cannot_update_config(setting_read_client, fake_setting_service):
    response = setting_read_client.put(
        "/api/settings/config",
        json={"key": "jwt__expire_minutes", "value": 60},
    )

    assert response.status_code == 403


def test_setting_read_cannot_reset_config(setting_read_client, fake_setting_service):
    response = setting_read_client.request(
        "DELETE",
        "/api/settings/config",
        json={"keys": ["jwt__expire_minutes"]},
    )

    assert response.status_code == 403
