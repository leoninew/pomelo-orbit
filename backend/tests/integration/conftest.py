"""集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.infrastructure.persistence.models import (
    PermissionModel,
    ProjectMemberModel,
    ProjectModel,
    RoleModel,
    RolePermissionModel,
    UserModel,
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
    return User(
        id="test-user-id",
        username="testuser",
        password_hash="",
        status="enabled",
        oauth_provider="",
        oauth_provider_id="",
        email=None,
        auth_source="password",
    )


def seed_user(db_session, user_id: str, username: str = "testuser") -> None:
    if db_session.get(UserModel, user_id):
        return
    db_session.add(UserModel(id=user_id, username=username, password_hash="hashed", status="enabled"))


def seed_auth_permissions(db_session, user_id: str, permission_codes: list[str]) -> None:
    permission_names = {
        "login:read": "View Login History",
        "user:read": "View Users",
        "user:write": "Manage Users",
        "role:read": "View Roles",
        "role:write": "Manage Roles",
        "setting:read": "View Settings",
        "setting:write": "Manage Settings",
    }
    permissions = []
    for code in permission_codes:
        permission = db_session.query(PermissionModel).filter(PermissionModel.code == code).one_or_none()
        if permission is None:
            permission = PermissionModel(
                id=f"perm-{code.replace(':', '-')}",
                code=code,
                name=permission_names[code],
            )
            db_session.add(permission)
        permissions.append(permission)

    role = db_session.get(RoleModel, "role-test-admin")
    if role is None:
        role = RoleModel(id="role-test-admin", code="test-admin", name="Test Admin")
        db_session.add(role)
    db_session.flush()

    for permission in permissions:
        if db_session.get(RolePermissionModel, (role.id, permission.id)) is None:
            db_session.add(RolePermissionModel(role_id=role.id, permission_id=permission.id))
    if db_session.get(UserRoleModel, (user_id, role.id)) is None:
        db_session.add(UserRoleModel(user_id=user_id, role_id=role.id))


def seed_project(db_session, user_id: str) -> None:
    if db_session.get(ProjectModel, DEFAULT_CI_PROJECT_ID) is None:
        db_session.add(
            ProjectModel(
                id=DEFAULT_CI_PROJECT_ID,
                name="Test Project",
                code="test",
                is_active=True,
            )
        )
    if db_session.get(ProjectMemberModel, (DEFAULT_CI_PROJECT_ID, user_id)) is None:
        db_session.add(ProjectMemberModel(project_id=DEFAULT_CI_PROJECT_ID, user_id=user_id))


@pytest.fixture
def auth_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
    from pomelo_orbit.main import app

    seed_user(db_session, mock_user.id)
    seed_auth_permissions(
        db_session,
        mock_user.id,
        [
            "login:read",
            "user:read",
            "user:write",
            "role:read",
            "role:write",
            "setting:read",
            "setting:write",
        ],
    )
    seed_project(db_session, mock_user.id)
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)


@pytest.fixture
def user_write_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
    from pomelo_orbit.main import app

    seed_user(db_session, mock_user.id)
    seed_auth_permissions(db_session, mock_user.id, ["user:read", "user:write"])
    seed_project(db_session, mock_user.id)
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)


@pytest.fixture
def setting_read_client(client, db_session, mock_user):
    from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
    from pomelo_orbit.main import app

    seed_user(db_session, mock_user.id)
    seed_auth_permissions(db_session, mock_user.id, ["setting:read"])
    seed_project(db_session, mock_user.id)
    db_session.commit()

    app.dependency_overrides[get_current_user] = lambda: mock_user
    yield AuthClient(client)
    app.dependency_overrides.pop(get_current_user, None)
