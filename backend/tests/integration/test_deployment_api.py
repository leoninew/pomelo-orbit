"""
部署 API 集成测试
"""

from unittest.mock import patch

import pytest

from pomelo_orbit.domain.entities import TriggerType, User
from pomelo_orbit.domain.value_objects import DeployStatus, OperationType
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel, DeploymentModel


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
def test_deployment(db_session, test_app):
    """创建测试部署"""
    deployment = DeploymentModel(
        application_id=test_app.id,
        application_name=test_app.name,
        operation_type=OperationType.DEPLOY,
        trigger_type=TriggerType.MANUAL,
        status=DeployStatus.SUCCESS,
    )
    db_session.add(deployment)
    db_session.commit()
    db_session.refresh(deployment)
    return deployment


class TestDeploymentAPI:
    """部署 API 测试"""

    def test_list_deployments(self, auth_client, test_deployment):
        """测试列出部署"""
        response = auth_client.get("/api/deployment")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_list_deployments_with_filters(self, auth_client, test_app, test_deployment):
        """测试带过滤条件列出部署"""
        response = auth_client.get(f"/api/deployment?application_id={test_app.id}&status={DeployStatus.SUCCESS}")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert all(d["application_id"] == test_app.id for d in data["items"])

    def test_get_deployment(self, auth_client, test_deployment):
        """测试获取部署详情"""
        response = auth_client.get(f"/api/deployment/{test_deployment.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_deployment.id
        assert data["application_name"] == "test-app"

    def test_get_deployment_not_found(self, auth_client):
        """测试获取不存在的部署"""
        response = auth_client.get("/api/deployment/nonexistent-id")

        assert response.status_code == 404
        assert "not found" in response.json()["detail"].lower()

    @patch("pomelo_orbit.application.deployment_service.DeploymentService.read_deployment_log")
    def test_get_deployment_logs(self, mock_read_log, auth_client, test_deployment, tmp_path):
        """测试获取部署日志"""
        mock_read_log.return_value = ("deployment log content", 100, True)

        response = auth_client.get(f"/api/deployment/{test_deployment.id}/logs")

        assert response.status_code == 200
        data = response.json()
        assert "logs" in data
        assert data["is_complete"] is True

    def test_cancel_deployment(self, auth_client, db_session, test_app):
        """测试取消部署"""
        deployment = DeploymentModel(
            application_id=test_app.id,
            application_name=test_app.name,
            operation_type=OperationType.DEPLOY,
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.RUNNING,
        )
        db_session.add(deployment)
        db_session.commit()
        db_session.refresh(deployment)

        response = auth_client.post(f"/api/deployment/{deployment.id}/cancel")

        assert response.status_code == 200
        data = response.json()
        assert "message" in data

        # 验证数据库更新
        db_session.expire_all()
        updated = db_session.query(DeploymentModel).filter_by(id=deployment.id).first()
        assert updated.status == DeployStatus.FAILED
        assert "Cancelled" in updated.error_message
