from pomelo_orbit.infrastructure.persistence.models import UserModel
from pomelo_orbit.infrastructure.security import hash_password, verify_password


def create_user(db_session, username: str, email: str | None = None, is_active: bool = True) -> UserModel:
    user = UserModel(
        username=username,
        email=email,
        password_hash=hash_password("old-password"),
        is_active=is_active,
    )
    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)
    return user


def test_create_and_list_users(auth_client, db_session):
    response = auth_client.post(
        "/api/user",
        json={"username": "alice", "email": "alice@example.com", "password": "secret"},
    )

    assert response.status_code == 201
    body = response.json()
    assert body["username"] == "alice"
    assert body["email"] == "alice@example.com"
    assert body["is_active"] is True
    assert "password_hash" not in body

    response = auth_client.get("/api/user", params={"search": "alice"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 1
    assert body["items"][0]["username"] == "alice"


def test_create_user_rejects_duplicate_username(auth_client, db_session):
    create_user(db_session, "alice")

    response = auth_client.post("/api/user", json={"username": "alice", "password": "secret"})

    assert response.status_code == 409


def test_user_password_requires_min_length(auth_client, db_session):
    user = create_user(db_session, "alice")

    response = auth_client.post("/api/user", json={"username": "bob", "password": "short"})
    assert response.status_code == 422

    response = auth_client.put(f"/api/user/{user.id}", json={"password": "short"})
    assert response.status_code == 422


def test_update_user_resets_password(auth_client, db_session):
    user = create_user(db_session, "alice", "alice@example.com")

    response = auth_client.put(f"/api/user/{user.id}", json={"password": "new-password"})

    assert response.status_code == 200
    body = response.json()
    assert body["username"] == "alice"
    assert body["email"] == "alice@example.com"

    db_session.refresh(user)
    assert verify_password("new-password", user.password_hash)


def test_disable_enable_and_delete_user(auth_client, db_session):
    user = create_user(db_session, "alice")

    response = auth_client.post(f"/api/user/{user.id}/disable")
    assert response.status_code == 204
    db_session.refresh(user)
    assert not bool(user.is_active)

    response = auth_client.post(f"/api/user/{user.id}/enable")
    assert response.status_code == 204
    db_session.refresh(user)
    assert bool(user.is_active)

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
