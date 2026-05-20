"""
部署 API 集成测试
"""

from unittest.mock import patch

from pomelo_orbit.domain.cd.entities import TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType, TaskStatus
from pomelo_orbit.infrastructure.persistence.models import DeploymentModel, ProjectModel


class TestDeploymentAPI:
    """部署 API 测试"""

    def test_list_deployments(self, auth_client, test_deployment):
        """测试列出部署"""
        from tests.integration.conftest import DEFAULT_CI_PROJECT_ID

        response = auth_client.get(f"/api/cd/deployment?project_id={DEFAULT_CI_PROJECT_ID}")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_list_deployments_rejects_non_member_project(self, auth_client, db_session):
        """非项目成员不能列出部署"""
        db_session.add(ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True))
        db_session.commit()

        response = auth_client.get("/api/cd/deployment?project_id=other-project-id")

        assert response.status_code == 403
        assert response.json()["detail"] == "Permission denied"

    def test_list_deployments_with_filters(self, auth_client, test_app, test_deployment):
        """测试带过滤条件列出部署"""
        from tests.integration.conftest import DEFAULT_CI_PROJECT_ID

        response = auth_client.get(
            f"/api/cd/deployment?project_id={DEFAULT_CI_PROJECT_ID}&application_id={test_app.id}&status={TaskStatus.RAN_TO_COMPLETION}"
        )

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert all(d["application_id"] == test_app.id for d in data["items"])

    def test_get_deployment(self, auth_client, test_deployment):
        """测试获取部署详情"""
        response = auth_client.get(f"/api/cd/deployment/{test_deployment.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_deployment.id
        assert data["application_name"] == "test-app"

    def test_get_deployment_rejects_non_member_project(self, auth_client, db_session, test_app):
        """非项目成员不能查看其他项目部署"""
        db_session.add(ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True))
        deployment = DeploymentModel(
            project_id="other-project-id",
            application_id=test_app.id,
            application_name=test_app.name,
            operation_type=OperationType.DEPLOY,
            trigger_type=TriggerType.MANUAL,
            status=TaskStatus.RUNNING,
        )
        db_session.add(deployment)
        db_session.commit()
        db_session.refresh(deployment)

        response = auth_client.get(f"/api/cd/deployment/{deployment.id}")

        assert response.status_code == 403
        assert response.json()["detail"] == "Permission denied"

    def test_get_deployment_not_found(self, auth_client):
        """测试获取不存在的部署"""
        response = auth_client.get("/api/cd/deployment/nonexistent-id")

        assert response.status_code == 404
        assert "not found" in response.json()["detail"].lower()

    @patch("pomelo_orbit.application.cd.deployment_service.DeploymentService.read_deployment_log")
    def test_get_deployment_logs(self, mock_read_log, auth_client, test_deployment, tmp_path):
        """测试获取部署日志"""
        mock_read_log.return_value = ("deployment log content", 100, True)

        response = auth_client.get(f"/api/cd/deployment/{test_deployment.id}/logs")

        assert response.status_code == 200
        data = response.json()
        assert "logs" in data
        assert data["is_complete"] is True

    def test_cancel_deployment(self, auth_client, db_session, test_app):
        """测试取消部署"""
        from tests.integration.conftest import DEFAULT_CI_PROJECT_ID

        deployment = DeploymentModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            application_id=test_app.id,
            application_name=test_app.name,
            operation_type=OperationType.DEPLOY,
            trigger_type=TriggerType.MANUAL,
            status=TaskStatus.RUNNING,
        )
        db_session.add(deployment)
        db_session.commit()
        db_session.refresh(deployment)

        response = auth_client.post(f"/api/cd/deployment/{deployment.id}/cancel")

        assert response.status_code == 200
        data = response.json()
        assert "message" in data

        # 验证数据库更新
        db_session.expire_all()
        updated = db_session.query(DeploymentModel).filter_by(id=deployment.id).first()
        assert updated.status == TaskStatus.CANCELED
        assert "Cancelled" in updated.error_message
