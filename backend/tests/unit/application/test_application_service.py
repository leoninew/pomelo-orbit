"""
应用服务核心业务逻辑测试
"""

import asyncio
from unittest.mock import AsyncMock, Mock, create_autospec

import pytest

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.entities import Application, ApplicationConfigFile, Deployment, TriggerType
from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, DeployStatus, OperationType
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
    manager.read_deployment_log = Mock(return_value=("", 0))
    return manager


@pytest.fixture
def app_service(mock_app_repo, mock_deployment_repo, mock_config_file_repo, mock_app_manager):
    """创建应用服务实例"""
    return ApplicationService(
        app_repo=mock_app_repo,
        deployment_repo=mock_deployment_repo,
        config_file_repo=mock_config_file_repo,
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
        status=DeployStatus.WAITING_TO_RUN.value,
        operation_type=OperationType.DEPLOY,
        is_rollback=False,
    )


class TestApplicationServiceInit:
    """ApplicationService 初始化测试"""

    def test_creates_service_with_repositories(self, mock_app_repo, mock_deployment_repo, mock_config_file_repo):
        """测试使用仓储创建服务"""
        from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
        from pomelo_orbit.infrastructure.config import get_settings

        service = ApplicationService(
            app_repo=mock_app_repo,
            deployment_repo=mock_deployment_repo,
            config_file_repo=mock_config_file_repo,
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


class TestFindLastSuccessfulDeployment:
    """查找最近成功部署测试"""

    def test_finds_last_successful_deployment(self, app_service, mock_deployment_repo):
        """测试查找最近一次成功的部署"""
        deployment = Deployment(
            id="deploy-2",
            application_id="app-1",
            application_name="Test",
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.RAN_TO_COMPLETION.value,
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
        )
        mock_deployment_repo.find_last_successful_deploy.return_value = deployment

        result = app_service._find_last_successful_deployment("app-1")

        assert result is not None
        assert result.id == "deploy-2"
        mock_deployment_repo.find_last_successful_deploy.assert_called_once_with("app-1")

    def test_returns_none_when_no_successful_deployment(self, app_service, mock_deployment_repo):
        """测试没有成功部署时返回 None"""
        mock_deployment_repo.find_last_successful_deploy.return_value = None

        result = app_service._find_last_successful_deployment("app-1")

        assert result is None

    def test_ignores_non_deploy_operations(self, app_service, mock_deployment_repo):
        """测试仓储层负责过滤非部署操作，应用服务直接使用结果"""
        mock_deployment_repo.find_last_successful_deploy.return_value = None

        result = app_service._find_last_successful_deployment("app-1")

        assert result is None
        mock_deployment_repo.find_last_successful_deploy.assert_called_once_with("app-1")


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
        assert DeployStatus.RUNNING.value in saved_statuses

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

        assert sample_deployment.status == DeployStatus.RAN_TO_COMPLETION.value
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
        assert final_deployment.status == DeployStatus.FAULTED.value
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

    @pytest.mark.asyncio
    async def test_stop_uses_env_file_from_last_deployment(
        self, app_service, mock_app_repo, mock_deployment_repo, mock_app_manager, tmp_path
    ):
        """测试停止时使用最近部署的 env_file"""
        app = Application(
            id="app-1",
            name="Test",
            code="test-app",
            status=ApplicationStatus.DEPLOYED,
            image_pull_policy="IfNotPresent",
        )
        mock_app_repo.find_by_id.return_value = app

        last_deployment = Deployment(
            id="deploy-1",
            application_id="app-1",
            application_name="Test",
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.RAN_TO_COMPLETION.value,
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
            env_file=".env.production",
        )
        mock_deployment_repo.find_last_successful_deploy.return_value = last_deployment

        await app_service.stop_application("app-1")

        mock_app_manager.stop.assert_called_once()
        call_args = mock_app_manager.stop.call_args
        assert call_args[0][2] == ".env.production"


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

        assert result.status == DeployStatus.WAITING_TO_RUN.value


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
