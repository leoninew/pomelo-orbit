"""集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.shared.entities import User
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel


@pytest.fixture
def mock_user():
    return User(id="test-user-id", username="testuser", password_hash="")


@pytest.fixture
def auth_client(client, mock_user):
    from pomelo_orbit.interfaces.api.auth import get_current_user
    from pomelo_orbit.main import app

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield client
    app.dependency_overrides.pop(get_current_user, None)


@pytest.fixture
def test_app(db_session):
    app = ApplicationModel(
        name="test-app",
        code="test-app",
        enabled=True,
        image_pull_policy="missing",
    )
    db_session.add(app)
    db_session.commit()
    db_session.refresh(app)
    return app
