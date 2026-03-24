"""
凭据管理 API 集成测试
"""

import pytest

from pomelo_orbit.domain.entities import User
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel, CredentialModel


@pytest.fixture
def mock_user():
    """Mock 用户"""
    return User(id="test-user-id", username="testuser", password_hash="")


@pytest.fixture
def auth_client(client, mock_user):
    """带认证的客户端"""
    from pomelo_orbit.interfaces.api.auth import get_current_user
    from pomelo_orbit.main import app

    def override_get_current_user():
        return mock_user

    app.dependency_overrides[get_current_user] = override_get_current_user
    yield client
    app.dependency_overrides.pop(get_current_user, None)


@pytest.fixture
def test_app(db_session):
    """创建测试应用"""
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


@pytest.fixture
def test_credential(db_session, test_app):
    """创建测试凭据"""
    from pomelo_orbit.infrastructure.config import get_settings
    from pomelo_orbit.infrastructure.security import SecurityService

    security_service = SecurityService(get_settings())
    credential = CredentialModel(
        application_id=test_app.id,
        name="test-credential",
        type="github_token",
        value_encrypted=security_service.encrypt_value("test-token-value"),
    )
    db_session.add(credential)
    db_session.commit()
    db_session.refresh(credential)
    return credential


class TestCredentialAPI:
    """凭据管理 API 测试"""

    def test_create_credential(self, auth_client, test_app):
        """测试创建凭据"""
        response = auth_client.post(
            "/api/credential",
            json={
                "application_id": test_app.id,
                "name": "new-credential",
                "type": "github_token",
                "value": "secret-token",
            },
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "new-credential"
        assert data["type"] == "github_token"
        assert "value_encrypted" not in data

    def test_list_credentials(self, auth_client, test_credential):
        """测试列出凭据"""
        response = auth_client.get("/api/credential")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_get_credential(self, auth_client, test_credential):
        """测试获取凭据详情"""
        response = auth_client.get(f"/api/credential/{test_credential.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_credential.id
        assert data["name"] == "test-credential"
        assert data["value"] == "test-token-value"

    def test_update_credential(self, auth_client, db_session, test_credential):
        """测试更新凭据"""
        response = auth_client.put(
            f"/api/credential/{test_credential.id}",
            json={"name": "updated-credential", "value": "new-secret"},
        )

        assert response.status_code == 200
        assert response.json()["name"] == "updated-credential"

        db_session.expire_all()
        credential = db_session.query(CredentialModel).filter_by(id=test_credential.id).first()
        assert credential is not None
        assert credential.name == "updated-credential"

    def test_delete_credential(self, auth_client, db_session, test_credential):
        """测试删除凭据"""
        response = auth_client.delete(f"/api/credential/{test_credential.id}")

        assert response.status_code == 204

        db_session.expire_all()
        credential = db_session.query(CredentialModel).filter_by(id=test_credential.id).first()
        assert credential is None

    def test_create_duplicate_name(self, auth_client, test_credential, test_app):
        """测试创建重复名称凭据"""
        response = auth_client.post(
            "/api/credential",
            json={
                "application_id": test_app.id,
                "name": "test-credential",
                "type": "github_token",
                "value": "another-token",
            },
        )

        assert response.status_code == 400
        assert "already exists" in response.json()["detail"]

    def test_create_with_nonexistent_app(self, auth_client, test_app):
        """测试为不存在的应用创建凭据"""
        response = auth_client.post(
            "/api/credential",
            json={
                "application_id": "nonexistent-id",
                "name": "test-credential",
                "type": "github_token",
                "value": "token",
            },
        )

        assert response.status_code == 404
