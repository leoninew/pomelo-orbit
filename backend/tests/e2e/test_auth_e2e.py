"""Backend authentication E2E tests."""

from fastapi.testclient import TestClient
from sqlalchemy.orm import Session

from tests.e2e.conftest import auth_headers, decode_captcha_answer, login, seed_user


def test_health_runs_after_lifespan_migrations(e2e_client: TestClient) -> None:
    response = e2e_client.get("/api/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_login_me_and_logout_use_real_token(e2e_client: TestClient, e2e_db: Session) -> None:
    user = seed_user(e2e_db, permission_codes=["login:read"])

    headers = auth_headers(e2e_client)
    me_response = e2e_client.get("/api/auth/me", headers=headers)
    logout_response = e2e_client.post("/api/auth/logout", headers=headers)

    assert me_response.status_code == 200
    assert me_response.json()["id"] == user.id
    assert me_response.json()["username"] == user.username
    assert "login:read" in me_response.json()["permissions"]
    assert logout_response.status_code == 200
    assert logout_response.json() == {"message": "Logged out successfully"}


def test_change_password_updates_real_login_credentials(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)

    wrong_old_response = e2e_client.put(
        "/api/auth/password",
        headers=headers,
        json={"old_password": "wrong-password", "new_password": "newpass123"},
    )
    change_response = e2e_client.put(
        "/api/auth/password",
        headers=headers,
        json={"old_password": "testpass123", "new_password": "newpass123"},
    )

    assert wrong_old_response.status_code == 400
    assert change_response.status_code == 200
    assert change_response.json() == {"message": "Password changed successfully"}

    csrf_token = e2e_client.get("/api/auth/csrf-token").json()["token"]
    captcha_token = e2e_client.get("/api/auth/captcha").json()["token"]
    old_login_response = e2e_client.post(
        "/api/auth/login",
        json={
            "username": "e2euser",
            "password": "testpass123",
            "csrf_token": csrf_token,
            "captcha_token": captcha_token,
            "captcha_answer": decode_captcha_answer(captcha_token),
        },
    )
    assert old_login_response.status_code == 400

    new_token = login(e2e_client, password="newpass123")
    assert new_token


def test_login_history_requires_permission(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db, user_id="history-denied-user", username="historydenied")
    denied_headers = auth_headers(e2e_client, username="historydenied")

    denied_response = e2e_client.get("/api/auth/login-history", headers=denied_headers)

    assert denied_response.status_code == 403

    seed_user(
        e2e_db,
        user_id="history-allowed-user",
        username="historyallowed",
        permission_codes=["login:read"],
    )
    allowed_headers = auth_headers(e2e_client, username="historyallowed")
    allowed_response = e2e_client.get("/api/auth/login-history", headers=allowed_headers)

    assert allowed_response.status_code == 200
    data = allowed_response.json()
    assert data["total"] >= 1
    assert any(item["username"] == "historyallowed" for item in data["items"])
