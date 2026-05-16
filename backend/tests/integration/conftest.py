"""集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.infrastructure.persistence.models import ProjectModel

DEFAULT_CI_PROJECT_ID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"


class AuthClient:
    def __init__(self, client):
        self._client = client

    def __getattr__(self, name):
        return getattr(self._client, name)


@pytest.fixture
def mock_user():
    return User(id="test-user-id", username="testuser", password_hash="")


@pytest.fixture
def auth_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.router import get_current_user
    from pomelo_orbit.main import app

    db_session.add(
        ProjectModel(
            id=DEFAULT_CI_PROJECT_ID,
            name="Test Project",
            code="test",
            owner_user_id=mock_user.id,
        )
    )
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)
