"""
应用服务
整合应用生命周期管理和部署功能
"""

import asyncio
import logging
import subprocess

from ulid import ULID

from pomelo_orbit.domain.cd.application_lifecycle_service import ApplicationLifecycleDomainService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.entities import (
    Application,
    ApplicationConfigFile,
    Deployment,
    TriggerType,
)
from pomelo_orbit.domain.cd.repositories import (
    ApplicationRepository,
    ConfigFileRepository,
    DeploymentRepository,
)
from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, DeployStatus, OperationType
from pomelo_orbit.domain.exceptions import BusinessError

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
        执行部署（后台任务）

        Args:
            application: 应用配置
            deployment: 部署记录

        Returns:
            部署是否成功
        """
        ApplicationLifecycleDomainService.validate_deploy(application)

        lock = self.get_lock(application.id)

        async with lock:
            # 标记应用为部署中
            application.mark_as_deploying()
            self.app_repo.save(application)

            deployment.status = DeployStatus.RUNNING.value
            self.deployment_repo.save(deployment)
            self.deployment_repo.commit()

            try:
                config_files = self.config_file_repo.find_by_application(application.id)
                await self.app_manager.deploy(
                    application.code,
                    config_files=config_files,
                    pull_policy=application.image_pull_policy,
                    deployment_id=deployment.id,
                    env_file=deployment.env_file,
                )

                ApplicationLifecycleDomainService.mark_deploy_success(deployment, application)
                self.deployment_repo.save(deployment)
                self.app_repo.save(application)
                self.deployment_repo.commit()

                logger.info(f"Deploy succeeded: app={application.code}, deployment={deployment.id}")
                return True

            except Exception as e:
                error_detail = str(e)
                if isinstance(e, subprocess.CalledProcessError) and e.output:
                    error_detail = f"{e}\nOutput:\n{e.output}"

                ApplicationLifecycleDomainService.mark_deploy_failure(deployment, application, error_detail)
                self.deployment_repo.save(deployment)
                self.app_repo.save(application)
                self.deployment_repo.commit()

                logger.error(
                    f"Deploy failed: app={application.code}, deployment={deployment.id}, error={e}", exc_info=True
                )
                return False

    async def execute_restart(
        self,
        application: Application,
        deployment: Deployment,
    ) -> bool:
        """
        执行重启（后台任务）

        Args:
            application: 应用配置
            deployment: 部署记录

        Returns:
            重启是否成功
        """
        lock = self.get_lock(application.id)

        async with lock:
            application.mark_as_deploying()
            self.app_repo.save(application)

            deployment.status = DeployStatus.RUNNING.value
            self.deployment_repo.save(deployment)
            self.deployment_repo.commit()

            try:
                await self.app_manager.restart(application.code, deployment.env_file)

                ApplicationLifecycleDomainService.mark_deploy_success(deployment, application)
                self.deployment_repo.save(deployment)
                self.app_repo.save(application)
                self.deployment_repo.commit()

                logger.info(f"Restart succeeded: app={application.code}, deployment={deployment.id}")
                return True

            except Exception as e:
                error_detail = str(e)
                if isinstance(e, subprocess.CalledProcessError) and e.output:
                    error_detail = f"{e}\nOutput:\n{e.output}"

                ApplicationLifecycleDomainService.mark_deploy_failure(deployment, application, error_detail)
                self.deployment_repo.save(deployment)
                self.app_repo.save(application)
                self.deployment_repo.commit()

                logger.error(
                    f"Restart failed: app={application.code}, deployment={deployment.id}, error={e}", exc_info=True
                )
                return False

    def _find_last_successful_deployment(self, application_id: str) -> Deployment | None:
        return self.deployment_repo.find_last_successful_deploy(application_id)

    async def stop_application(
        self, application_id: str, remove_volumes: bool = False, env_file: str | None = None
    ) -> Deployment:
        """
        停止应用（同步）

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

        ApplicationLifecycleDomainService.validate_stop(app)

        # 如果未指定 env_file，从最近一次成功部署记录获取
        if env_file is None:
            last_deployment = self._find_last_successful_deployment(application_id)
            env_file = last_deployment.env_file if last_deployment else None

        deployment = ApplicationLifecycleDomainService.create_stop_record(app, env_file)
        self.deployment_repo.save(deployment)

        lock = self.get_lock(application_id)
        async with lock:
            try:
                await self.app_manager.stop(app.code, remove_volumes, env_file)

                ApplicationLifecycleDomainService.mark_stop_success(deployment, app)
                self.deployment_repo.save(deployment)
                self.app_repo.save(app)

                logger.info(f"Application stopped: app={app.code}")
                return deployment
            except Exception as e:
                ApplicationLifecycleDomainService.mark_stop_failure(deployment, app, str(e))
                self.deployment_repo.save(deployment)
                self.app_repo.save(app)

                logger.error(f"Application stop failed: app={app.code}, error={e}", exc_info=True)
                raise

    async def restart_application(self, application_id: str, env_file: str | None = None) -> Deployment:
        """
        重启应用（异步，返回部署记录，后台执行）

        Args:
            application_id: 应用 ID
            env_file: 环境文件名（可选，未指定时从最近部署记录获取）

        Returns:
            部署记录
        """
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        ApplicationLifecycleDomainService.validate_restart(app)

        # 如果未指定 env_file，从最近一次成功部署记录获取
        if env_file is None:
            last_deployment = self._find_last_successful_deployment(application_id)
            env_file = last_deployment.env_file if last_deployment else None

        deployment = ApplicationLifecycleDomainService.create_restart_record(app, env_file=env_file)
        self.deployment_repo.save(deployment)
        self.deployment_repo.commit()

        return deployment

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
        获取应用运行状态

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
        image_pull_policy: str,
    ) -> Application:
        """创建应用"""
        existing = self.app_repo.find_by_name(name)
        if existing:
            raise BusinessError(f"Application '{name}' already exists", status_code=400)

        app = Application(
            id=str(ULID()),
            name=name,
            code=code,
            image_pull_policy=image_pull_policy,
            status=ApplicationStatus.UNDEPLOYED,
        )

        self.app_repo.save(app)
        return app

    def update_application(
        self,
        application_id: str,
        update_data: dict,
    ) -> Application:
        """更新应用"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        for key, value in update_data.items():
            if hasattr(app, key):
                setattr(app, key, value)

        self.app_repo.save(app)
        return app

    # ==================== 配置文件管理 ====================

    def get_config_files(self, application_id: str) -> list[ApplicationConfigFile]:
        """获取应用的所有配置文件"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        return self.config_file_repo.find_by_application(application_id)

    def get_config_file(self, application_id: str, config_file_id: str) -> ApplicationConfigFile:
        """获取单个配置文件"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_file = self.config_file_repo.find_by_id(config_file_id)
        if not config_file or config_file.application_id != application_id:
            raise BusinessError(f"Config file {config_file_id} not found", status_code=404)

        return config_file

    def create_config_file(self, application_id: str, path: str, content: str) -> ApplicationConfigFile:
        """创建配置文件"""
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
            status=DeployStatus.WAITING_TO_RUN,
            env_file=env_file,
            is_rollback=is_rollback,
        )

        self.deployment_repo.save(deployment)
        self.deployment_repo.commit()
        return deployment

    # ==================== 导入/导出 ====================

    def export_application(self, application_id: str) -> dict:
        """导出应用及其关联数据"""
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application {application_id} not found", status_code=404)

        config_files = self.config_file_repo.find_by_application(application_id)

        return {
            "name": app.name,
            "code": app.code,
            "image_pull_policy": app.image_pull_policy,
            "config_files": [{"path": cf.path, "content": cf.content} for cf in config_files],
        }

    def import_application(self, data: dict) -> Application:
        """导入应用数据"""
        existing_name = self.app_repo.find_by_name(data["name"])
        if existing_name:
            raise BusinessError(f"Application name '{data['name']}' already exists", status_code=400)

        existing_code = self.app_repo.find_by_code(data["code"])
        if existing_code:
            raise BusinessError(f"Application code '{data['code']}' already exists", status_code=400)

        app = Application(
            id=str(ULID()),
            name=data["name"],
            code=data["code"],
            image_pull_policy=data.get("image_pull_policy", "missing"),
            status=ApplicationStatus.UNDEPLOYED,
        )

        self.app_repo.save(app)

        for cf_data in data.get("config_files", []):
            config_file = ApplicationConfigFile(
                id=str(ULID()),
                application_id=app.id,
                path=cf_data["path"],
                content=cf_data.get("content", ""),
            )
            self.config_file_repo.save(config_file)

        return app


__all__ = ["ApplicationService"]
