"""
应用管理 API 集成测试
"""

from unittest.mock import patch

import pytest

from pomelo_orbit.domain.entities import User
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
)


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
def test_config_file(db_session, test_app):
    """创建测试配置文件"""
    config_file = ApplicationConfigFileModel(
        application_id=test_app.id,
        path=".env",
        content="KEY=value",
    )
    db_session.add(config_file)
    db_session.commit()
    db_session.refresh(config_file)
    return config_file


class TestApplicationAPI:
    """应用管理 API 测试"""

    def test_database_setup(self, db_session):
        """验证测试数据库已正确设置"""
        from sqlalchemy import inspect

        inspector = inspect(db_session.bind)
        tables = inspector.get_table_names()

        assert "application" in tables
        assert "user" in tables

    def test_create_application_directly(self, db_session):
        """直接创建应用（不通过 API）"""
        app = ApplicationModel(
            name="direct-app",
            code="direct-app",
            enabled=True,
            image_pull_policy="missing",
        )
        db_session.add(app)
        db_session.commit()
        db_session.refresh(app)

        # 验证创建成功
        assert app.id is not None
        assert app.name == "direct-app"

    def test_create_application(self, auth_client, test_app):
        """测试创建应用"""
        # test_app 确保数据库已初始化并有数据
        response = auth_client.post(
            "/api/application",
            json={
                "name": "new-app",
                "code": "new-app",
                "image_pull_policy": "always",
                "enabled": True,
            },
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "new-app"
        assert data["code"] == "new-app"

    def test_list_applications(self, auth_client, test_app):
        """测试列出应用"""
        response = auth_client.get("/api/application")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_get_application(self, auth_client, test_app):
        """测试获取应用详情"""
        response = auth_client.get(f"/api/application/{test_app.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_app.id
        assert data["name"] == "test-app"

    def test_update_application(self, auth_client, db_session, test_app):
        """测试更新应用（验证持久化）"""
        response = auth_client.put(
            f"/api/application/{test_app.id}",
            json={"name": "updated-app"},
        )

        assert response.status_code == 200
        assert response.json()["name"] == "updated-app"

        # 验证数据库持久化
        db_session.expire_all()
        app = db_session.query(ApplicationModel).filter_by(id=test_app.id).first()
        assert app is not None
        assert app.name == "updated-app"

    @patch("pomelo_orbit.infrastructure.docker.manager.ApplicationManagerImpl.purge")
    def test_delete_application(self, mock_purge, auth_client, db_session, test_app):
        """测试删除应用"""
        mock_purge.return_value = None

        response = auth_client.delete(f"/api/application/{test_app.id}")

        assert response.status_code == 204

        # 验证数据库删除
        db_session.expire_all()
        app = db_session.query(ApplicationModel).filter_by(id=test_app.id).first()
        assert app is None

    def test_list_application_files(self, auth_client, test_app, test_config_file):
        """测试列出配置文件"""
        response = auth_client.get(f"/api/application/{test_app.id}/files")

        assert response.status_code == 200
        data = response.json()
        assert len(data) >= 1
        assert data[0]["path"] == ".env"

    def test_create_application_file(self, auth_client, db_session, test_app):
        """测试创建配置文件"""
        response = auth_client.post(
            f"/api/application/{test_app.id}/file",
            json={"path": "config.yaml", "content": "key: value"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["path"] == "config.yaml"

        # 验证数据库持久化
        db_session.expire_all()
        config_file = (
            db_session.query(ApplicationConfigFileModel)
            .filter_by(application_id=test_app.id, path="config.yaml")
            .first()
        )
        assert config_file is not None
        assert config_file.content == "key: value"

    def test_read_application_file(self, auth_client, test_app, test_config_file):
        """测试读取配置文件"""
        response = auth_client.get(f"/api/application/{test_app.id}/file/{test_config_file.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["path"] == ".env"
        assert data["content"] == "KEY=value"

    @patch("pomelo_orbit.application.application_service.ApplicationService.update_config_file")
    def test_write_application_file(self, mock_write, auth_client, test_app, test_config_file):
        """测试更新配置文件"""
        from pomelo_orbit.domain.entities import ApplicationConfigFile

        updated_file = ApplicationConfigFile(
            id=test_config_file.id,
            application_id=test_app.id,
            path=".env",
            content="KEY=new_value",
        )
        mock_write.return_value = updated_file

        response = auth_client.put(
            f"/api/application/{test_app.id}/file/{test_config_file.id}",
            json={"path": ".env", "content": "KEY=new_value"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["path"] == ".env"
        mock_write.assert_called_once()

    def test_delete_application_file(self, auth_client, db_session, test_app, test_config_file):
        """测试删除配置文件"""
        response = auth_client.delete(f"/api/application/{test_app.id}/file/{test_config_file.id}")

        assert response.status_code == 204

        # 验证数据库删除
        db_session.expire_all()
        config_file = db_session.query(ApplicationConfigFileModel).filter_by(id=test_config_file.id).first()
        assert config_file is None

    @patch("pomelo_orbit.application.application_service.ApplicationService.deploy")
    def test_deploy_application(self, mock_deploy, auth_client, db_session, test_app):
        """测试部署应用"""
        response = auth_client.post(
            f"/api/application/{test_app.id}/deploy",
            json={"branch": "main", "env": "production"},
        )

        assert response.status_code == 200
        data = response.json()
        assert "deployment_id" in data

    @patch("pomelo_orbit.application.application_service.ApplicationService.stop_application")
    def test_stop_application(self, mock_stop, auth_client, test_app):
        """测试停止应用"""
        from pomelo_orbit.domain.entities import Deployment, TriggerType
        from pomelo_orbit.domain.value_objects import DeployStatus, OperationType

        mock_deployment = Deployment(
            id="test-deployment-id",
            application_id=test_app.id,
            application_name=test_app.name,
            operation_type=OperationType.STOP,
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.WAITING_TO_RUN,
            is_rollback=False,
        )
        mock_stop.return_value = mock_deployment

        response = auth_client.post(
            f"/api/application/{test_app.id}/stop",
            json={"remove_volumes": False},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["deployment_id"] == "test-deployment-id"

    @patch("pomelo_orbit.application.application_service.ApplicationService.restart_application")
    def test_restart_application(self, mock_restart, auth_client, test_app):
        """测试重启应用"""
        from pomelo_orbit.domain.entities import Deployment, TriggerType
        from pomelo_orbit.domain.value_objects import DeployStatus, OperationType

        mock_deployment = Deployment(
            id="test-deployment-id",
            application_id=test_app.id,
            application_name=test_app.name,
            operation_type=OperationType.RESTART,
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.WAITING_TO_RUN,
            is_rollback=False,
        )
        mock_restart.return_value = mock_deployment

        response = auth_client.post(f"/api/application/{test_app.id}/restart")

        assert response.status_code == 200
        data = response.json()
        assert data["deployment_id"] == "test-deployment-id"

    @patch("pomelo_orbit.application.application_service.ApplicationService.get_application_status")
    def test_get_application_status(self, mock_status, auth_client, test_app):
        """测试获取应用状态"""
        mock_status.return_value = {"running": True, "container_id": "abc123"}

        response = auth_client.get(f"/api/application/{test_app.id}/status")

        assert response.status_code == 200
        data = response.json()
        assert "status" in data
        assert data["status"]["running"] is True

    @patch("pomelo_orbit.application.application_service.ApplicationService.get_application_logs")
    def test_get_application_logs(self, mock_logs, auth_client, test_app):
        """测试获取应用日志"""
        mock_logs.return_value = "log line 1\nlog line 2"

        response = auth_client.get(f"/api/application/{test_app.id}/logs?tail=50")

        assert response.status_code == 200
        data = response.json()
        assert "logs" in data
