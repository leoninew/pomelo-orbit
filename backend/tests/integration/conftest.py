"""集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.auth.entities import User


@pytest.fixture
def mock_user():
    return User(id="test-user-id", username="testuser", password_hash="")


@pytest.fixture
def auth_client(client, mock_user):
    from pomelo_orbit.interfaces.api.auth.router import get_current_user
    from pomelo_orbit.main import app

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield client
    app.dependency_overrides.pop(get_current_user, None)
