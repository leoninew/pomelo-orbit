"""
认证 API 集成测试
"""

import pytest

from pomelo_orbit.infrastructure import hash_password
from pomelo_orbit.infrastructure.persistence.models import UserModel


@pytest.fixture
def test_user(db_session):
    """创建测试用户"""
    user = UserModel(
        username="testuser",
        password_hash=hash_password("testpass123"),
    )
    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)
    return user


class TestAuthAPI:
    """认证 API 测试"""

    def test_login_success(self, client, test_user):
        """测试成功登录"""
        response = client.post(
            "/api/auth/login",
            json={"username": "testuser", "password": "testpass123"},
        )

        assert response.status_code == 200
        data = response.json()
        assert "access_token" in data
        assert data["access_token"] != ""

    def test_login_wrong_password(self, client, test_user):
        """测试密码错误"""
        response = client.post(
            "/api/auth/login",
            json={"username": "testuser", "password": "wrongpass"},
        )

        assert response.status_code == 400
        assert "用户名或密码错误" in response.json()["detail"]

    def test_login_user_not_found(self, client, test_user):
        """测试用户不存在"""
        response = client.post(
            "/api/auth/login",
            json={"username": "nonexistent", "password": "anypass"},
        )

        assert response.status_code == 400
        assert "用户名或密码错误" in response.json()["detail"]

    def test_get_me_with_valid_token(self, client, test_user):
        """测试使用有效 Token 获取用户信息"""
        # 先登录获取 token
        login_response = client.post(
            "/api/auth/login",
            json={"username": "testuser", "password": "testpass123"},
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

    def test_list_login_history(self, client, test_user):
        """测试查询登录历史"""
        # 先登录生成历史记录
        login_response = client.post(
            "/api/auth/login",
            json={"username": "testuser", "password": "testpass123"},
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
