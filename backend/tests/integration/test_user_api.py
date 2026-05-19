from datetime import datetime

from pomelo_orbit.infrastructure.persistence.models import (
    PermissionModel,
    RoleModel,
    RolePermissionModel,
    UserModel,
    UserRoleModel,
)
from pomelo_orbit.infrastructure.security import hash_password, verify_password


def create_user(db_session, username: str, email: str | None = None, status: str = "enabled") -> UserModel:
    user = UserModel(
        username=username,
        email=email,
        password_hash=hash_password("old-password"),
        status=status,
    )
    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)
    return user


def create_role(db_session, code: str = "developer", name: str = "Developer") -> RoleModel:
    role = RoleModel(code=code, name=name)
    db_session.add(role)
    db_session.commit()
    db_session.refresh(role)
    return role


def test_create_and_list_users(auth_client, db_session):
    role = create_role(db_session)

    response = auth_client.post(
        "/api/user",
        json={
            "username": "alice",
            "email": "alice@example.com",
            "password": "secret",
            "role_ids": [role.id],
        },
    )

    assert response.status_code == 201
    body = response.json()
    assert body["username"] == "alice"
    assert body["email"] == "alice@example.com"
    assert body["status"] == "enabled"
    assert "is_active" not in body
    assert body["roles"] == ["developer"]
    assert body["role_items"] == [{"id": role.id, "code": "developer", "name": "Developer"}]
    assert "password_hash" not in body

    response = auth_client.get("/api/user", params={"search": "alice"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 1
    assert body["items"][0]["username"] == "alice"
    assert body["items"][0]["status"] == "enabled"
    assert "is_active" not in body["items"][0]
    assert body["items"][0]["role_items"] == [{"id": role.id, "code": "developer", "name": "Developer"}]
    assert "roles" not in body["items"][0]
    assert "permissions" not in body["items"][0]


def test_create_user_rejects_duplicate_username(auth_client, db_session):
    create_user(db_session, "alice")

    response = auth_client.post("/api/user", json={"username": "alice", "password": "secret"})

    assert response.status_code == 409


def test_user_password_requires_min_length(auth_client, db_session):
    user = create_user(db_session, "alice")

    response = auth_client.post("/api/user", json={"username": "bob", "password": "short"})
    assert response.status_code == 422

    response = auth_client.put(f"/api/user/{user.id}", json={"password": "short", "status": "enabled"})
    assert response.status_code == 422


def test_update_user_resets_password(auth_client, db_session):
    user = create_user(db_session, "alice", "alice@example.com")

    response = auth_client.put(f"/api/user/{user.id}", json={"password": "new-password", "status": "enabled"})

    assert response.status_code == 200
    body = response.json()
    assert body["username"] == "alice"
    assert body["email"] == "alice@example.com"
    assert body["role_items"] == []

    db_session.refresh(user)
    assert verify_password("new-password", user.password_hash)


def test_update_user_sets_roles(auth_client, db_session):
    user = create_user(db_session, "alice")
    role = create_role(db_session)
    old_updated_at = datetime(2024, 1, 1)
    user.updated_at = old_updated_at
    db_session.commit()

    response = auth_client.put(f"/api/user/{user.id}", json={"role_ids": [role.id], "status": "enabled"})

    assert response.status_code == 200
    body = response.json()
    assert body["roles"] == ["developer"]
    assert body["role_items"] == [{"id": role.id, "code": "developer", "name": "Developer"}]
    assert body["updated_at"] != "2024-01-01T00:00:00Z"
    assert db_session.get(UserRoleModel, (user.id, role.id)) is not None


def test_update_user_sets_status(auth_client, db_session):
    user = create_user(db_session, "alice")

    response = auth_client.put(f"/api/user/{user.id}", json={"status": "disabled"})

    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "disabled"
    assert "is_active" not in body
    db_session.refresh(user)
    assert user.status == "disabled"


def test_cannot_disable_current_user_via_update(auth_client, db_session):
    user = UserModel(
        id="test-user-id",
        username="current",
        password_hash=hash_password("password"),
    )
    db_session.add(user)
    db_session.commit()

    response = auth_client.put("/api/user/test-user-id", json={"status": "disabled"})

    assert response.status_code == 400
    db_session.refresh(user)
    assert user.status == "enabled"


def test_user_role_ids_must_be_unique(auth_client, db_session):
    user = create_user(db_session, "alice")
    role = create_role(db_session)

    response = auth_client.post(
        "/api/user",
        json={"username": "bob", "password": "secret", "role_ids": [role.id, role.id]},
    )
    assert response.status_code == 422

    response = auth_client.put(f"/api/user/{user.id}", json={"role_ids": [role.id, role.id], "status": "enabled"})
    assert response.status_code == 422


def test_user_write_cannot_assign_roles_without_role_write(user_write_client, db_session):
    user = create_user(db_session, "alice")
    role = create_role(db_session)

    response = user_write_client.put(f"/api/user/{user.id}", json={"role_ids": [role.id], "status": "enabled"})

    assert response.status_code == 403
    assert db_session.get(UserRoleModel, (user.id, role.id)) is None


def test_user_write_cannot_reset_role_write_user_password(user_write_client, db_session):
    user = create_user(db_session, "admin")
    role = create_role(db_session, "admin", "Admin")
    permission = PermissionModel(id="perm-test-role-write", code="role:write", name="Manage Roles")
    db_session.add(permission)
    db_session.add(RolePermissionModel(role_id=role.id, permission_id=permission.id))
    db_session.add(UserRoleModel(user_id=user.id, role_id=role.id))
    db_session.commit()

    response = user_write_client.put(f"/api/user/{user.id}", json={"password": "new-password", "status": "enabled"})

    assert response.status_code == 403
    db_session.refresh(user)
    assert verify_password("old-password", user.password_hash)


def test_disable_enable_and_delete_user(auth_client, db_session):
    user = create_user(db_session, "alice")

    response = auth_client.post(f"/api/user/{user.id}/disable")
    assert response.status_code == 204
    db_session.refresh(user)
    assert user.status == "disabled"

    response = auth_client.post(f"/api/user/{user.id}/enable")
    assert response.status_code == 204
    db_session.refresh(user)
    assert user.status == "enabled"

    response = auth_client.delete(f"/api/user/{user.id}")
    assert response.status_code == 204
    assert db_session.get(UserModel, user.id) is None


def test_cannot_disable_or_delete_current_user(auth_client, db_session):
    user = UserModel(
        id="test-user-id",
        username="current",
        password_hash=hash_password("password"),
    )
    db_session.add(user)
    db_session.commit()

    response = auth_client.post("/api/user/test-user-id/disable")
    assert response.status_code == 400

    response = auth_client.delete("/api/user/test-user-id")
    assert response.status_code == 400
