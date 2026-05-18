"""集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.infrastructure.persistence.models import (
    PermissionModel,
    ProjectModel,
    RoleModel,
    RolePermissionModel,
    UserRoleModel,
)

DEFAULT_CI_PROJECT_ID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"


class AuthClient:
    def __init__(self, client):
        self._client = client

    def __getattr__(self, name):
        return getattr(self._client, name)


@pytest.fixture
def mock_user():
    return User(id="test-user-id", username="testuser", password_hash="")


def seed_auth_permissions(db_session, user_id: str, permission_codes: list[str]) -> None:
    permission_names = {
        "user:read": "View Users",
        "user:write": "Manage Users",
        "role:read": "View Roles",
        "role:write": "Manage Roles",
    }
    permissions = [
        PermissionModel(
            id=f"perm-{code.replace(':', '-')}",
            code=code,
            name=permission_names[code],
        )
        for code in permission_codes
    ]
    role = RoleModel(id="role-test-admin", code="test-admin", name="Test Admin")
    db_session.add_all(
        [
            *permissions,
            role,
            *[RolePermissionModel(role_id=role.id, permission_id=permission.id) for permission in permissions],
            UserRoleModel(user_id=user_id, role_id=role.id),
        ]
    )


def seed_project(db_session, user_id: str) -> None:
    db_session.add(
        ProjectModel(
            id=DEFAULT_CI_PROJECT_ID,
            name="Test Project",
            code="test",
            owner_user_id=user_id,
        )
    )


@pytest.fixture
def auth_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.router import get_current_user
    from pomelo_orbit.main import app

    seed_auth_permissions(db_session, mock_user.id, ["user:read", "user:write", "role:read", "role:write"])
    seed_project(db_session, mock_user.id)
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)


@pytest.fixture
def user_write_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.router import get_current_user
    from pomelo_orbit.main import app

    seed_auth_permissions(db_session, mock_user.id, ["user:read", "user:write"])
    seed_project(db_session, mock_user.id)
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)
