"""
应用服务核心业务逻辑测试
"""

import asyncio
from unittest.mock import AsyncMock, Mock, create_autospec

import pytest

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.entities import (
    Application,
    ApplicationConfigFile,
    ApplicationServiceConfig,
    Deployment,
    TriggerType,
)
from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, OperationType, TaskStatus
from pomelo_orbit.domain.exceptions import BusinessError


@pytest.fixture
def mock_app_repo():
    """Mock 应用仓储"""
    return Mock()


@pytest.fixture
def mock_deployment_repo():
    """Mock 部署仓储"""
    return Mock()


@pytest.fixture
def mock_config_file_repo():
    """Mock 配置文件仓储"""
    return Mock()


@pytest.fixture
def mock_app_service_config_repo():
    """Mock 应用 service 配置仓储"""
    repo = Mock()
    repo.find_by_application.return_value = []
    repo.find_by_application_and_service.return_value = None
    return repo


@pytest.fixture
def mock_app_manager():
    """Mock 应用管理器（自动对齐 ApplicationManager 接口）"""
    manager = create_autospec(ApplicationManager, instance=True)
    manager.deploy = AsyncMock()
    manager.start = AsyncMock()
    manager.stop = AsyncMock()
    manager.restart = AsyncMock()
    manager.status = AsyncMock(return_value="status")
    manager.logs = AsyncMock(return_value="logs")
    manager.purge = Mock()
    manager.render_compose = Mock(side_effect=lambda _app, content, _filename: content)
    manager.preview_docker_compose = Mock(return_value="version: '3'\nservices: {}\n")
    manager.get_domain_suffix = Mock(return_value="example.com")
    manager.read_deployment_log = Mock(return_value=("", 0))
    return manager


@pytest.fixture
def mock_app_route_repo():
    """Mock 应用路由仓储"""
    return Mock()


@pytest.fixture
def app_service(
    mock_app_repo,
    mock_deployment_repo,
    mock_config_file_repo,
    mock_app_route_repo,
    mock_app_service_config_repo,
    mock_app_manager,
):
    """创建应用服务实例"""
    return ApplicationService(
        app_repo=mock_app_repo,
        deployment_repo=mock_deployment_repo,
        config_file_repo=mock_config_file_repo,
        app_route_repo=mock_app_route_repo,
        app_service_config_repo=mock_app_service_config_repo,
        app_manager=mock_app_manager,
    )


@pytest.fixture
def sample_application():
    """创建示例应用"""
    return Application(
        id="app-1",
        name="Test App",
        code="test-app",
        status=ApplicationStatus.UNDEPLOYED,
        image_pull_policy="IfNotPresent",
    )


@pytest.fixture
def sample_deployment():
    """创建示例部署记录"""
    return Deployment(
        id="deploy-1",
        application_id="app-1",
        application_name="Test App",
        trigger_type=TriggerType.MANUAL,
        status=TaskStatus.WAITING_TO_RUN.value,
        operation_type=OperationType.DEPLOY,
        is_rollback=False,
    )


class TestApplicationServiceInit:
    """ApplicationService 初始化测试"""

    def test_creates_service_with_repositories(
        self, mock_app_repo, mock_deployment_repo, mock_config_file_repo, mock_app_service_config_repo
    ):
        """测试使用仓储创建服务"""
        from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
        from pomelo_orbit.infrastructure.config import get_settings

        service = ApplicationService(
            app_repo=mock_app_repo,
            deployment_repo=mock_deployment_repo,
            config_file_repo=mock_config_file_repo,
            app_route_repo=Mock(),
            app_service_config_repo=mock_app_service_config_repo,
            app_manager=ApplicationManagerImpl(get_settings()),
        )

        assert service.app_repo == mock_app_repo
        assert service.deployment_repo == mock_deployment_repo
        assert service.config_file_repo == mock_config_file_repo
        assert service.app_manager is not None


class TestGetLock:
    """应用锁获取测试"""

    def test_creates_lock_for_new_application(self, app_service):
        """测试为新应用创建锁"""
        lock = app_service.get_lock("app-1")

        assert isinstance(lock, asyncio.Lock)
        assert "app-1" in app_service._locks

    def test_returns_same_lock_for_same_application(self, app_service):
        """测试同一应用返回相同的锁"""
        lock1 = app_service.get_lock("app-1")
        lock2 = app_service.get_lock("app-1")

        assert lock1 is lock2

    def test_creates_different_locks_for_different_applications(self, app_service):
        """测试不同应用创建不同的锁"""
        lock1 = app_service.get_lock("app-1")
        lock2 = app_service.get_lock("app-2")

        assert lock1 is not lock2


class TestConfigFileManagement:
    """配置文件管理测试"""

    def test_get_config_files(self, app_service, mock_config_file_repo):
        """测试获取应用的所有配置文件"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path=".env", content="KEY=value"),
            ApplicationConfigFile(id="cfg-2", application_id="app-1", path="config.yml", content="port: 8080"),
        ]
        mock_config_file_repo.find_by_application.return_value = config_files

        result = app_service.get_config_files("app-1")

        assert len(result) == 2
        assert result[0].path == ".env"
        assert result[1].path == "config.yml"
        mock_config_file_repo.find_by_application.assert_called_once_with("app-1")

    def test_get_config_file_success(self, app_service, mock_config_file_repo):
        """测试成功获取单个配置文件"""
        config_file = ApplicationConfigFile(id="cfg-1", application_id="app-1", path=".env", content="KEY=value")
        mock_config_file_repo.find_by_id.return_value = config_file

        result = app_service.get_config_file("app-1", "cfg-1")

        assert result is not None
        assert result.id == "cfg-1"
        assert result.application_id == "app-1"

    def test_get_config_file_wrong_application(self, app_service, mock_config_file_repo, mock_app_repo):
        """测试获取不属于该应用的配置文件"""
        from pomelo_orbit.domain.cd.value_objects import ApplicationStatus

        app = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            image_pull_policy="Always",
            status=ApplicationStatus.UNDEPLOYED,
        )
        mock_app_repo.find_by_id.return_value = app

        config_file = ApplicationConfigFile(id="cfg-1", application_id="app-2", path=".env", content="KEY=value")
        mock_config_file_repo.find_by_id.return_value = config_file

        with pytest.raises(BusinessError) as exc_info:
            app_service.get_config_file("app-1", "cfg-1")

        assert exc_info.value.status_code == 404
        assert "not found" in str(exc_info.value)

    def test_get_config_file_not_found(self, app_service, mock_config_file_repo, mock_app_repo):
        """测试获取不存在的配置文件"""
        from pomelo_orbit.domain.cd.value_objects import ApplicationStatus

        app = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            image_pull_policy="Always",
            status=ApplicationStatus.UNDEPLOYED,
        )
        mock_app_repo.find_by_id.return_value = app

        mock_config_file_repo.find_by_id.return_value = None

        with pytest.raises(BusinessError) as exc_info:
            app_service.get_config_file("app-1", "cfg-1")

        assert exc_info.value.status_code == 404
        assert "not found" in str(exc_info.value)

    def test_delete_config_file_success(self, app_service, mock_config_file_repo):
        """测试成功删除配置文件"""
        config_file = ApplicationConfigFile(id="cfg-1", application_id="app-1", path=".env", content="KEY=value")
        mock_config_file_repo.find_by_id.return_value = config_file

        app_service.delete_config_file("app-1", "cfg-1")

        mock_config_file_repo.delete.assert_called_once_with(config_file)

    def test_delete_config_file_not_found(self, app_service, mock_config_file_repo):
        """测试删除不存在的配置文件"""
        mock_config_file_repo.find_by_id.return_value = None

        from pomelo_orbit.domain.exceptions import BusinessError

        with pytest.raises(BusinessError, match="Config file cfg-1 not found"):
            app_service.delete_config_file("app-1", "cfg-1")


class TestServiceConfigManagement:
    """service 级配置管理测试"""

    def test_list_service_configs_merges_compose_and_override(
        self, app_service, sample_application, mock_app_repo, mock_config_file_repo, mock_app_service_config_repo
    ):
        mock_app_repo.find_by_id.return_value = sample_application
        mock_config_file_repo.find_by_application.return_value = [
            ApplicationConfigFile(
                id="cfg-1",
                application_id="app-1",
                path="docker-compose.yml",
                content=(
                    "services:\n"
                    "  web:\n"
                    "    image: nginx:1.25\n"
                    "    ports:\n"
                    '      - "8080:80"\n'
                    "  worker:\n"
                    "    image: busybox:1.36\n"
                ),
            )
        ]
        mock_app_service_config_repo.find_by_application.return_value = [
            ApplicationServiceConfig(
                id="svc-1",
                application_id="app-1",
                service_name="web",
                image="nginx:1.27",
            )
        ]

        result = app_service.list_service_configs("app-1")

        assert result[0]["service_name"] == "web"
        assert result[0]["base_image"] == "nginx:1.25"
        assert result[0]["image"] == "nginx:1.27"
        assert result[0]["default_domain"] == "test-app.example.com"
        assert result[0]["default_port"] == 80
        assert result[1]["service_name"] == "worker"
        assert result[1]["image"] is None

    def test_update_service_config_upserts_image(
        self, app_service, sample_application, mock_app_repo, mock_config_file_repo, mock_app_service_config_repo
    ):
        mock_app_repo.find_by_id.return_value = sample_application
        mock_config_file_repo.find_by_application.return_value = [
            ApplicationConfigFile(
                id="cfg-1",
                application_id="app-1",
                path="docker-compose.yml",
                content="services:\n  web:\n    image: nginx:1.25\n",
            )
        ]

        result = app_service.update_service_config("app-1", "web", " nginx:1.27 ")

        saved = mock_app_service_config_repo.save.call_args[0][0]
        assert saved.service_name == "web"
        assert saved.image == "nginx:1.27"
        assert result["image"] == "nginx:1.27"

    def test_update_service_config_rejects_empty_image(
        self, app_service, sample_application, mock_app_repo, mock_config_file_repo, mock_app_service_config_repo
    ):
        mock_app_repo.find_by_id.return_value = sample_application
        mock_config_file_repo.find_by_application.return_value = [
            ApplicationConfigFile(
                id="cfg-1",
                application_id="app-1",
                path="docker-compose.yml",
                content="services:\n  web:\n    image: nginx:1.25\n",
            )
        ]
        with pytest.raises(BusinessError, match="Image cannot be empty"):
            app_service.update_service_config("app-1", "web", "   ")

        mock_app_service_config_repo.delete.assert_not_called()
        mock_app_service_config_repo.save.assert_not_called()


class TestDeployBusinessLogic:
    """部署业务逻辑测试"""

    @pytest.mark.asyncio
    async def test_deploy_updates_deployment_status_to_running(
        self, app_service, sample_application, sample_deployment, mock_deployment_repo, mock_config_file_repo
    ):
        """测试部署开始时更新状态为 RUNNING"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        mock_config_file_repo.find_by_application.return_value = config_files

        # 用于捕获中间状态
        saved_statuses = []

        def capture_status(deployment):
            saved_statuses.append(deployment.status)

        mock_deployment_repo.save.side_effect = capture_status

        await app_service.deploy(sample_application, sample_deployment)

        # 验证第一次保存时状态为 RUNNING
        assert TaskStatus.RUNNING.value in saved_statuses

    @pytest.mark.asyncio
    async def test_deploy_success_updates_application_status(
        self,
        app_service,
        sample_application,
        sample_deployment,
        mock_deployment_repo,
        mock_config_file_repo,
        mock_app_repo,
    ):
        """测试部署成功时更新应用状态为 DEPLOYED"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        mock_config_file_repo.find_by_application.return_value = config_files

        result = await app_service.deploy(sample_application, sample_deployment)

        assert result is True
        assert sample_application.status == ApplicationStatus.DEPLOYED
        mock_app_repo.save.assert_called_with(sample_application)

    @pytest.mark.asyncio
    async def test_deploy_success_updates_deployment_record(
        self, app_service, sample_application, sample_deployment, mock_deployment_repo, mock_config_file_repo
    ):
        """测试部署成功时更新部署记录"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        mock_config_file_repo.find_by_application.return_value = config_files

        await app_service.deploy(sample_application, sample_deployment)

        assert sample_deployment.status == TaskStatus.RAN_TO_COMPLETION.value
        assert sample_deployment.finished_at is not None
        assert sample_deployment.duration_ms is not None

    @pytest.mark.asyncio
    async def test_deploy_failure_records_error(
        self,
        app_service,
        sample_application,
        sample_deployment,
        mock_deployment_repo,
        mock_config_file_repo,
        mock_app_manager,
    ):
        """测试部署失败时记录错误信息"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        mock_config_file_repo.find_by_application.return_value = config_files
        mock_app_manager.deploy = AsyncMock(side_effect=Exception("Docker error"))

        result = await app_service.deploy(sample_application, sample_deployment)

        assert result is False
        calls = mock_deployment_repo.save.call_args_list
        final_deployment = calls[-1][0][0]
        assert final_deployment.status == TaskStatus.FAULTED.value
        assert "Docker error" in final_deployment.error_message
        assert final_deployment.finished_at is not None

    @pytest.mark.asyncio
    async def test_deploy_uses_application_lock(
        self, app_service, sample_application, sample_deployment, mock_config_file_repo
    ):
        """测试部署使用应用锁防止并发"""
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        mock_config_file_repo.find_by_application.return_value = config_files

        lock = app_service.get_lock(sample_application.id)
        assert not lock.locked()

        await app_service.deploy(sample_application, sample_deployment)

        assert not lock.locked()

    @pytest.mark.asyncio
    async def test_deploy_passes_service_configs_to_app_manager(
        self,
        app_service,
        sample_application,
        sample_deployment,
        mock_config_file_repo,
        mock_app_service_config_repo,
        mock_app_manager,
    ):
        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="app-1", path="docker-compose.yml", content="version: '3'")
        ]
        service_configs = [
            ApplicationServiceConfig(
                id="svc-1",
                application_id="app-1",
                service_name="web",
                image="nginx:1.27",
            )
        ]
        mock_config_file_repo.find_by_application.return_value = config_files
        mock_app_service_config_repo.find_by_application.return_value = service_configs

        await app_service.deploy(sample_application, sample_deployment)

        assert mock_app_manager.deploy.await_args.kwargs["service_configs"] == service_configs


class TestStopApplicationBusinessLogic:
    """停止应用业务逻辑测试"""

    @pytest.mark.asyncio
    async def test_stop_requires_application_exists(self, app_service, mock_app_repo):
        """测试停止应用时应用必须存在"""
        from pomelo_orbit.domain.exceptions import BusinessError

        mock_app_repo.find_by_id.return_value = None

        with pytest.raises(BusinessError, match="Application app-1 not found"):
            await app_service.stop_application("app-1")

    @pytest.mark.asyncio
    async def test_stop_requires_application_started(self, app_service, mock_app_repo):
        """测试只能停止已启动的应用"""
        from pomelo_orbit.domain.exceptions import BusinessError

        app = Application(
            id="app-1",
            name="Test",
            code="test",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        with pytest.raises(BusinessError, match="应用未在运行中"):
            await app_service.stop_application("app-1")

    @pytest.mark.asyncio
    async def test_stop_creates_deployment_record(self, app_service, mock_app_repo, mock_deployment_repo, tmp_path):
        """测试停止应用时创建部署记录"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_deployment_repo.find_by_application.return_value = ([], 0)

        await app_service.stop_application("app-1")

        calls = mock_deployment_repo.save.call_args_list
        assert len(calls) >= 1
        first_deployment = calls[0][0][0]
        assert first_deployment.operation_type == OperationType.STOP
        assert first_deployment.trigger_type == TriggerType.MANUAL

    @pytest.mark.asyncio
    async def test_stop_success_updates_application_status(
        self, app_service, mock_app_repo, mock_deployment_repo, tmp_path
    ):
        """测试停止成功时更新应用状态为 UNDEPLOYED"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_deployment_repo.find_by_application.return_value = ([], 0)

        await app_service.stop_application("app-1")

        assert app.status == ApplicationStatus.UNDEPLOYED
        mock_app_repo.save.assert_called_once_with(app)


class TestRestartApplicationBusinessLogic:
    """重启应用业务逻辑测试"""

    @pytest.mark.asyncio
    async def test_restart_requires_application_started(self, app_service, mock_app_repo, mock_deployment_repo):
        """测试只能重启已启动的应用"""
        app = Application(
            id="app-1",
            name="Test",
            code="test",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_deployment_repo.find_by_application.return_value = ([], 0)

        with pytest.raises(BusinessError, match="应用未在运行中"):
            await app_service.restart_application("app-1")

    @pytest.mark.asyncio
    async def test_restart_creates_deployment_record(self, app_service, mock_app_repo, mock_deployment_repo, tmp_path):
        """测试重启应用时创建部署记录"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_deployment_repo.find_by_application.return_value = ([], 0)

        await app_service.restart_application("app-1")

        calls = mock_deployment_repo.save.call_args_list
        first_deployment = calls[0][0][0]
        assert first_deployment.operation_type == OperationType.RESTART
        assert first_deployment.trigger_type == TriggerType.MANUAL

    @pytest.mark.asyncio
    async def test_restart_success_updates_deployment_record(
        self, app_service, mock_app_repo, mock_deployment_repo, tmp_path
    ):
        """测试重启返回 WAITING_TO_RUN 状态的部署记录（后台异步执行）"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_deployment_repo.find_by_application.return_value = ([], 0)

        result = await app_service.restart_application("app-1")

        assert result.status == TaskStatus.WAITING_TO_RUN.value


class TestDeleteApplicationBusinessLogic:
    """删除应用业务逻辑测试"""

    @pytest.mark.asyncio
    async def test_delete_requires_application_stopped(self, app_service, mock_app_repo):
        """测试只能删除已停止的应用"""
        app = Application(
            id="app-1",
            name="Test",
            code="test",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        with pytest.raises(BusinessError, match="应用正在运行中"):
            await app_service.delete_application("app-1")

    @pytest.mark.asyncio
    async def test_delete_without_removing_directory(self, app_service, mock_app_repo, tmp_path):
        """测试删除应用但不删除目录"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        app_dir = tmp_path / "test-app"
        app_dir.mkdir()

        result = await app_service.delete_application("app-1", remove_dir=False)

        assert result is True
        assert app_dir.exists()

    @pytest.mark.asyncio
    async def test_delete_with_removing_directory(self, app_service, mock_app_repo, mock_app_manager, tmp_path):
        """测试删除应用并删除目录"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        app_dir = tmp_path / "test-app"
        app_dir.mkdir()
        (app_dir / "test.txt").write_text("test")

        mock_app_manager.purge = Mock(side_effect=lambda _: __import__("shutil").rmtree(app_dir))

        result = await app_service.delete_application("app-1", remove_dir=True)

        assert result is True
        assert not app_dir.exists()


class TestPreviewComposeYaml:
    """docker-compose 预览"""

    def test_preview_returns_manager_output(
        self, app_service, mock_app_repo, mock_config_file_repo, mock_app_service_config_repo, mock_app_manager
    ):
        mock_app_repo.find_by_id.return_value = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_config_file_repo.find_by_application.return_value = [
            ApplicationConfigFile(
                id="cfg-1",
                application_id="app-1",
                path="docker-compose.yml",
                content="services:\n  web:\n    image: nginx:1.25\n",
            )
        ]
        mock_app_manager.preview_docker_compose.return_value = "rendered-yaml"

        result = app_service.preview_compose_yaml("app-1")

        assert result == "rendered-yaml"
        mock_app_manager.preview_docker_compose.assert_called_once()

    def test_preview_requires_compose_file(self, app_service, mock_app_repo, mock_config_file_repo):
        mock_app_repo.find_by_id.return_value = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_config_file_repo.find_by_application.return_value = []

        with pytest.raises(BusinessError, match="No docker-compose"):
            app_service.preview_compose_yaml("app-1")

    def test_preview_with_route_managed(
        self,
        app_service,
        mock_app_repo,
        mock_config_file_repo,
        mock_app_service_config_repo,
        mock_app_route_repo,
        mock_app_manager,
    ):
        """route_managed=True 时应传入路由列表"""
        from pomelo_orbit.domain.cd.entities import ApplicationRoute

        mock_app_repo.find_by_id.return_value = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
            route_managed=True,
        )
        mock_config_file_repo.find_by_application.return_value = [
            ApplicationConfigFile(
                id="cfg-1",
                application_id="app-1",
                path="docker-compose.yml",
                content="services:\n  web:\n    image: nginx:1.25\n",
            )
        ]
        routes = [
            ApplicationRoute(
                id="route-1",
                application_id="app-1",
                service_name="web",
                domain="app.example.com",
                port=80,
            )
        ]
        mock_app_route_repo.find_by_application.return_value = routes
        mock_app_manager.preview_docker_compose.return_value = "rendered-with-labels"

        result = app_service.preview_compose_yaml("app-1")

        assert result == "rendered-with-labels"
        call_kwargs = mock_app_manager.preview_docker_compose.call_args
        # 确认 routes 参数被传入（非 None）
        assert call_kwargs.args[4] == routes or call_kwargs.kwargs.get("routes") == routes


class TestSimplifiedDeploymentFlow:
    """测试简化后的部署流程（移除 environment 和 env_file）"""

    @pytest.mark.asyncio
    async def test_deploy_without_env_file(
        self, app_service, mock_app_repo, mock_deployment_repo, mock_config_file_repo, mock_app_manager
    ):
        """测试部署不再需要 env_file 参数"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.UNDEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app
        mock_config_file_repo.find_by_application.return_value = []

        deployment = Deployment(
            id="deploy-1",
            application_id="app-1",
            application_name="Test",
            trigger_type=TriggerType.MANUAL,
            status=TaskStatus.WAITING_TO_RUN.value,
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
        )

        await app_service.deploy(app, deployment)

        # 验证 deploy 调用不包含 env_file 参数
        mock_app_manager.deploy.assert_called_once()
        call_kwargs = mock_app_manager.deploy.call_args.kwargs
        assert "env_file" not in call_kwargs

    @pytest.mark.asyncio
    async def test_stop_without_env_file_lookup(
        self, app_service, mock_app_repo, mock_deployment_repo, mock_app_manager
    ):
        """测试停止操作不再查找最后一次部署的 env_file"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        await app_service.stop_application("app-1")

        # 验证没有调用 find_last_successful_deploy
        assert (
            not hasattr(mock_deployment_repo, "find_last_successful_deploy")
            or not mock_deployment_repo.find_last_successful_deploy.called
        )

        # 验证 stop 调用只包含 application_code 和 remove_volumes 参数
        mock_app_manager.stop.assert_called_once()
        call_args = mock_app_manager.stop.call_args
        # stop 接收 application_code 和 remove_volumes 两个参数
        assert len(call_args.args) == 2  # application_code, remove_volumes
        assert call_args.args[0] == "test-app"
        assert call_args.args[1] is False  # remove_volumes 默认为 False
        assert "env_file" not in call_args.kwargs

    def test_deployment_entity_no_env_fields(self):
        """测试 Deployment 实体不再包含 environment 和 env_file 字段"""
        deployment = Deployment(
            id="deploy-1",
            application_id="app-1",
            application_name="Test",
            trigger_type=TriggerType.MANUAL,
            status=TaskStatus.WAITING_TO_RUN.value,
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
        )

        # 验证实体不包含这些字段
        assert not hasattr(deployment, "environment")
        assert not hasattr(deployment, "env_file")
