"""
认证 API 集成测试
"""

import pytest

from pomelo_orbit.infrastructure import hash_password
from pomelo_orbit.infrastructure.persistence.models import UserModel
from tests.unit.api.conftest import seed_auth_permissions


@pytest.fixture
def test_user(db_session):
    """创建测试用户"""
    user = UserModel(
        username="testuser",
        password_hash=hash_password("testpass123"),
        status="enabled",
    )
    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)
    return user


def get_csrf_token(client):
    """获取 CSRF Token"""
    response = client.get("/api/auth/csrf-token")
    assert response.status_code == 200
    return response.json()["token"]


def get_captcha(client):
    """获取验证码"""
    response = client.get("/api/auth/captcha")
    assert response.status_code == 200
    data = response.json()
    assert "token" in data
    assert "image" in data
    return data["token"], data["image"]


def decode_captcha_answer(captcha_token: str) -> str:
    """解码验证码答案（仅用于测试）"""
    from jose import jwt

    from pomelo_orbit.infrastructure.config import get_settings

    settings = get_settings()
    payload = jwt.decode(captcha_token, settings.jwt.secret_key, algorithms=["HS256"])
    answer: str = payload["answer"]
    return answer


class TestAuthAPI:
    """认证 API 测试"""

    def test_login_success(self, client, test_user):
        """测试成功登录"""
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert "access_token" in data
        assert data["access_token"] != ""

    def test_login_wrong_password(self, client, test_user):
        """测试密码错误"""
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "wrongpass",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )

        assert response.status_code == 400
        assert "用户名或密码错误" in response.json()["detail"]

    def test_login_user_not_found(self, client, test_user):
        """测试用户不存在"""
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        response = client.post(
            "/api/auth/login",
            json={
                "username": "nonexistent",
                "password": "anypass",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )

        assert response.status_code == 400
        assert "用户名或密码错误" in response.json()["detail"]

    def test_get_me_with_valid_token(self, client, test_user):
        """测试使用有效 Token 获取用户信息"""
        # 先登录获取 token
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        login_response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )
        token = login_response.json()["access_token"]

        # 使用 token 获取用户信息
        response = client.get(
            "/api/auth/me",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["username"] == "testuser"
        assert data["id"] == test_user.id

    def test_get_me_with_invalid_token(self, client):
        """测试使用无效 Token"""
        response = client.get(
            "/api/auth/me",
            headers={"Authorization": "Bearer invalid-token"},
        )

        assert response.status_code == 401
        assert "登录已过期" in response.json()["detail"]

    def test_list_login_history(self, client, db_session, test_user):
        """测试查询登录历史"""
        seed_auth_permissions(db_session, test_user.id, ["login:read"])
        db_session.commit()

        # 先登录生成历史记录
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        login_response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )
        token = login_response.json()["access_token"]

        # 查询登录历史
        response = client.get(
            "/api/auth/login-history",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1
        assert data["items"][0]["username"] == "testuser"
        assert data["items"][0]["success"] is True

    def test_list_login_history_without_permission_returns_403(self, client, test_user):
        """测试无权限查询登录历史返回 403"""
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)
        captcha_answer = decode_captcha_answer(captcha_token)

        login_response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": captcha_answer,
            },
        )
        token = login_response.json()["access_token"]

        response = client.get(
            "/api/auth/login-history",
            headers={"Authorization": f"Bearer {token}"},
        )

        assert response.status_code == 403

    def test_captcha_wrong_answer(self, client, test_user):
        """测试验证码错误"""
        csrf_token = get_csrf_token(client)
        captcha_token, _ = get_captcha(client)

        response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
                "captcha_token": captcha_token,
                "captcha_answer": "WRONG",
            },
        )

        assert response.status_code == 400
        assert "验证码错误或已过期" in response.json()["detail"]

    def test_captcha_missing(self, client, test_user):
        """测试缺少验证码"""
        csrf_token = get_csrf_token(client)

        response = client.post(
            "/api/auth/login",
            json={
                "username": "testuser",
                "password": "testpass123",
                "csrf_token": csrf_token,
            },
        )

        assert response.status_code == 422  # Validation error

    def test_get_captcha(self, client):
        """测试获取验证码"""
        captcha_token, captcha_image = get_captcha(client)
        assert captcha_token != ""
        assert captcha_image.startswith("data:image/png;base64,")
