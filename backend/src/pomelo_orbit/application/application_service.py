"""
应用服务
整合应用生命周期管理和部署功能
"""

import asyncio
import logging
import subprocess

from ulid import ULID

from pomelo_orbit.domain.application_lifecycle_service import ApplicationLifecycleDomainService
from pomelo_orbit.domain.application_manager import ApplicationManager
from pomelo_orbit.domain.entities import (
    Application,
    ApplicationConfigFile,
    Deployment,
    GitSource,
    ImageSource,
    TriggerType,
)
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.repositories import (
    ApplicationRepository,
    ConfigFileRepository,
    DeploymentRepository,
)
from pomelo_orbit.domain.value_objects import ApplicationStatus, DeployStatus, OperationType

logger = logging.getLogger(__name__)


class ApplicationService:
    """应用服务 - 整合应用生命周期管理和部署功能"""

    def __init__(
        self,
        app_repo: ApplicationRepository,
        deployment_repo: DeploymentRepository,
        config_file_repo: ConfigFileRepository,
        app_manager: ApplicationManager,
    ):
        self.app_repo = app_repo
        self.deployment_repo = deployment_repo
        self.config_file_repo = config_file_repo
        self.app_manager = app_manager
        self._locks: dict[str, asyncio.Lock] = {}

    def get_lock(self, application_id: str) -> asyncio.Lock:
        """获取应用锁, 防止同一应用并发操作"""
        if application_id not in self._locks:
            self._locks[application_id] = asyncio.Lock()
        return self._locks[application_id]

    async def deploy(
        self,
        application: Application,
        deployment: Deployment,
    ) -> bool:
        """
        执行部署

        Args:
            application: 应用配置
            deployment: 部署记录

        Returns:
            部署是否成功
        """
        # 领域验证
        ApplicationLifecycleDomainService.validate_deploy(application)

        lock = self.get_lock(application.id)

        async with lock:
            deployment.status = DeployStatus.RUNNING.value
            self.deployment_repo.save(deployment)

            try:
                config_files = self.config_file_repo.find_by_application(application.id)
                await self.app_manager.deploy(
                    application.code,
                    config_files=config_files,
                    pull_policy=application.image_pull_policy,
                    deployment_id=deployment.id,
                    env_file=deployment.env_file,
                )

                # 成功 - 使用领域服务更新状态
                ApplicationLifecycleDomainService.mark_deploy_success(deployment, application)
                self.deployment_repo.save(deployment)
                self.app_repo.save(application)

                logger.info(f"Deploy succeeded: app={application.code}, deployment={deployment.id}")
                return True

            except Exception as e:
                # 获取详细错误信息
                error_detail = str(e)
                if isinstance(e, subprocess.CalledProcessError) and e.output:
                    error_detail = f"{e}\nOutput:\n{e.output}"

                # 失败 - 使用领域服务标记
                ApplicationLifecycleDomainService.mark_deploy_failure(deployment, error_detail)
                self.deployment_repo.save(deployment)

                logger.error(
                    f"Deploy failed: app={application.code}, deployment={deployment.id}, error={e}", exc_info=True
                )
                return False

    async def start_application(self, application_id: str) -> bool:
        """
        启动应用

        Args:
            application_id: 应用 ID

        Returns:
            是否成功
        """
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        # 检查是否有部署记录
        deployments, _ = self.deployment_repo.find_by_application(application_id, page=1, per_page=1)
        has_deployment = len(deployments) > 0

        # 领域验证
        ApplicationLifecycleDomainService.validate_start(app, has_deployment)

        lock = self.get_lock(application_id)

        async with lock:
            try:
                await self.app_manager.start(app.code)
                logger.info(f"Application started: app={app.code}")
                return True
            except Exception as e:
                logger.error(f"Application start failed: app={app.code}, error={e}", exc_info=True)
                raise

    def _find_last_successful_deployment(self, application_id: str) -> Deployment | None:
        return self.deployment_repo.find_last_successful_deploy(application_id)

    async def stop_application(
        self, application_id: str, remove_volumes: bool = False, env_file: str | None = None
    ) -> Deployment:
        """
        停止应用

        Args:
            application_id: 应用 ID
            remove_volumes: 是否删除数据卷
            env_file: 环境文件名（可选，未指定时从最近部署记录获取）

        Returns:
            部署记录
        """
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        # 领域验证
        if not app.can_stop():
            raise BusinessError("Application is not running", status_code=400)

        # 如果未指定 env_file，从最近一次成功部署记录获取
        if env_file is None:
            last_deployment = self._find_last_successful_deployment(application_id)
            env_file = last_deployment.env_file if last_deployment else None

        # 创建停止记录（领域服务）
        deployment = ApplicationLifecycleDomainService.create_stop_record(app, env_file)
        self.deployment_repo.save(deployment)

        lock = self.get_lock(application_id)
        async with lock:
            try:
                await self.app_manager.stop(app.code, remove_volumes, env_file)

                # 标记操作成功并更新应用状态
                ApplicationLifecycleDomainService.mark_operation_success(deployment)
                app.mark_as_stopped()
                self.deployment_repo.save(deployment)
                self.app_repo.save(app)

                logger.info(f"Application stopped: app={app.code}")
                return deployment
            except Exception as e:
                ApplicationLifecycleDomainService.mark_operation_failure(deployment, str(e))
                self.deployment_repo.save(deployment)

                logger.error(f"Application stop failed: app={app.code}, error={e}", exc_info=True)
                raise

    async def restart_application(self, application_id: str, env_file: str | None = None) -> Deployment:
        """
        重启应用

        Args:
            application_id: 应用 ID
            env_file: 环境文件名（可选，未指定时从最近部署记录获取）

        Returns:
            部署记录
        """
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        # 检查是否有部署记录
        deployments, _ = self.deployment_repo.find_by_application(application_id, page=1, per_page=1)
        has_deployment = len(deployments) > 0

        # 领域验证
        ApplicationLifecycleDomainService.validate_restart(app, has_deployment)

        # 如果未指定 env_file，从最近一次成功部署记录获取
        if env_file is None:
            last_deployment = self._find_last_successful_deployment(application_id)
            env_file = last_deployment.env_file if last_deployment else None

        deployment = ApplicationLifecycleDomainService.create_restart_record(app, env_file)
        self.deployment_repo.save(deployment)

        lock = self.get_lock(application_id)
        async with lock:
            try:
                await self.app_manager.restart(app.code, env_file)

                ApplicationLifecycleDomainService.mark_operation_success(deployment)
                self.deployment_repo.save(deployment)

                logger.info(f"Application restarted: app={app.code}")
                return deployment
            except Exception as e:
                ApplicationLifecycleDomainService.mark_operation_failure(deployment, str(e))
                self.deployment_repo.save(deployment)

                logger.error(f"Application restart failed: app={app.code}, error={e}", exc_info=True)
                raise

    async def delete_application(self, application_id: str, remove_dir: bool = False) -> bool:
        """
        删除应用

        Args:
            application_id: 应用 ID
            remove_dir: 是否删除应用目录

        Returns:
            是否成功
        """
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        # 领域验证
        ApplicationLifecycleDomainService.validate_delete(app)

        lock = self.get_lock(application_id)

        async with lock:
            if remove_dir:
                self.app_manager.purge(app.code)

            self.app_repo.delete(app)
            logger.info(f"Application deleted: app={app.code}, remove_dir={remove_dir}")
            return True

    async def get_application_logs(self, application_code: str, tail: int = 100) -> str:
        """
        获取应用日志

        Args:
            application_code: 应用编码
            tail: 显示最后 N 行日志

        Returns:
            日志内容
        """
        return await self.app_manager.logs(application_code, tail)

    async def get_application_status(self, application_code: str) -> str:
        """
        获取应用状态

        Args:
            application_code: 应用编码

        Returns:
            状态信息
        """
        return await self.app_manager.status(application_code)

    # ==================== 应用 CRUD ====================

    def list_applications(self, page: int, per_page: int, search: str | None = None) -> tuple[list[Application], int]:
        """获取应用分页列表"""
        return self.app_repo.find_paginated(page, per_page, search)

    def get_application(self, application_id: str) -> Application:
        """获取应用详情"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)
        return app

    def create_application(
        self,
        name: str,
        code: str,
        enabled: bool,
        image_pull_policy: str,
        git_source_data: dict | None = None,
        image_source_data: dict | None = None,
    ) -> Application:
        """创建应用"""
        # 检查名称是否已存在
        existing = self.app_repo.find_by_name(name)
        if existing:
            raise BusinessError(f"Application '{name}' already exists", status_code=400)

        # 创建应用实体
        app_id = str(ULID())
        app = Application(
            id=app_id,
            name=name,
            code=code,
            enabled=enabled,
            image_pull_policy=image_pull_policy,
            status=ApplicationStatus.STOPPED,
        )

        # 创建源配置
        if git_source_data:
            app.git_source = GitSource(
                id=str(ULID()),
                application_id=app_id,
                repository_url=git_source_data["repository_url"],
                deploy_branches=git_source_data["deploy_branches"],
                auto_deploy=git_source_data["auto_deploy"],
            )
        if image_source_data:
            app.image_source = ImageSource(
                id=str(ULID()),
                application_id=app_id,
                image_name=image_source_data["image_name"],
                registry_url=image_source_data.get("registry_url"),
            )

        self.app_repo.save(app)
        return app

    def update_application(
        self,
        application_id: str,
        update_data: dict,
        git_source_data: dict | None = None,
        image_source_data: dict | None = None,
    ) -> Application:
        """更新应用"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        # 更新基本信息
        for key, value in update_data.items():
            if hasattr(app, key):
                setattr(app, key, value)

        # 更新源配置
        if git_source_data is not None:
            if app.git_source:
                for key, value in git_source_data.items():
                    setattr(app.git_source, key, value)
            else:
                app.git_source = GitSource(
                    id=str(ULID()),
                    application_id=app.id,
                    **git_source_data,
                )

        if image_source_data is not None:
            if app.image_source:
                for key, value in image_source_data.items():
                    setattr(app.image_source, key, value)
            else:
                app.image_source = ImageSource(
                    id=str(ULID()),
                    application_id=app.id,
                    **image_source_data,
                )

        self.app_repo.save(app)
        return app

    # ==================== 配置文件管理 ====================

    def get_config_files(self, application_id: str) -> list[ApplicationConfigFile]:
        """获取应用的所有配置文件"""
        # 验证应用存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        return self.config_file_repo.find_by_application(application_id)

    def get_config_file(self, application_id: str, config_file_id: str) -> ApplicationConfigFile:
        """获取单个配置文件"""
        # 验证应用存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_file = self.config_file_repo.find_by_id(config_file_id)
        if not config_file or config_file.application_id != application_id:
            raise BusinessError(f"Config file {config_file_id} not found", status_code=404)

        return config_file

    def create_config_file(self, application_id: str, path: str, content: str) -> ApplicationConfigFile:
        """创建配置文件"""
        # 验证应用存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_file = ApplicationConfigFile(
            id=str(ULID()),
            application_id=application_id,
            path=path,
            content=content,
        )
        self.config_file_repo.save(config_file)
        return config_file

    def delete_config_file(self, application_id: str, config_file_id: str) -> None:
        """删除配置文件"""
        # 验证应用存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_file = self.config_file_repo.find_by_id(config_file_id)
        if not config_file or config_file.application_id != application_id:
            raise BusinessError(f"Config file {config_file_id} not found", status_code=404)

        self.config_file_repo.delete(config_file)

    async def update_config_file(
        self, application_id: str, config_file_id: str, content: str, path: str | None = None
    ) -> ApplicationConfigFile:
        """写入配置文件内容到数据库和文件系统"""
        # 验证应用存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_file = self.config_file_repo.find_by_id(config_file_id)
        if not config_file or config_file.application_id != application_id:
            raise BusinessError(f"Config file {config_file_id} not found", status_code=404)

        if path and path != config_file.path:
            config_file.path = path

        config_file.content = content
        self.config_file_repo.save(config_file)
        return config_file

    def create_deployment(
        self,
        application_id: str,
        operation_type: OperationType,
        trigger_type: TriggerType,
        trigger_ref: str | None = None,
        env_file: str | None = None,
        is_rollback: bool = False,
    ) -> Deployment:
        """创建部署记录"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        deployment = Deployment(
            id=str(ULID()),
            application_id=app.id,
            application_name=app.name,
            operation_type=operation_type,
            trigger_type=trigger_type,
            trigger_ref=trigger_ref,
            status=DeployStatus.QUEUED,
            env_file=env_file,
            is_rollback=is_rollback,
        )

        self.deployment_repo.save(deployment)
        self.deployment_repo.commit()
        return deployment


__all__ = ["ApplicationService"]
