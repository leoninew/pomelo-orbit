"""Deployment application service - handles deployment business logic."""

import logging

from pomelo_orbit.domain.application_manager import ApplicationManager
from pomelo_orbit.domain.entities import Deployment
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.repositories import ApplicationRepository, DeploymentRepository
from pomelo_orbit.domain.value_objects import DeployStatus
from pomelo_orbit.infrastructure.time_utils import from_iso8601, utc_now

logger = logging.getLogger(__name__)


class DeploymentService:
    """部署应用服务 - 处理部署记录的业务逻辑"""

    def __init__(
        self,
        deployment_repo: DeploymentRepository,
        app_repo: ApplicationRepository,
        app_manager: ApplicationManager,
    ):
        self.deployment_repo = deployment_repo
        self.app_repo = app_repo
        self.app_manager = app_manager

    def list_deployments(
        self,
        page: int,
        per_page: int,
        application_id: str | None = None,
        status_filter: str | None = None,
        search: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
    ) -> tuple[list[Deployment], int]:
        """列出所有部署（分页，支持过滤）"""
        # 转换日期字符串为 datetime
        date_from_dt = from_iso8601(date_from) if date_from else None
        date_to_dt = from_iso8601(date_to) if date_to else None

        return self.deployment_repo.find_paginated_with_filters(
            page=page,
            per_page=per_page,
            application_id=application_id,
            status=status_filter,
            search=search,
            date_from=date_from_dt,
            date_to=date_to_dt,
        )

    def get_deployment(self, deployment_id: str) -> Deployment:
        """获取部署详情"""
        deployment = self.deployment_repo.find_by_id(deployment_id)
        if not deployment:
            raise BusinessError(f"Deployment {deployment_id} not found", status_code=404)
        return deployment

    def cancel_deployment(self, deployment_id: str) -> Deployment:
        """取消正在进行的部署"""
        deployment = self.deployment_repo.find_by_id(deployment_id)
        if not deployment:
            raise BusinessError(f"Deployment {deployment_id} not found", status_code=404)

        if deployment.status not in (DeployStatus.WAITING_TO_RUN, DeployStatus.RUNNING):
            raise BusinessError("Deployment is not in a cancellable state", status_code=400)

        deployment.status = DeployStatus.CANCELED
        deployment.error_message = "Cancelled by user"
        deployment.finished_at = utc_now()

        self.deployment_repo.save(deployment)

        return deployment

    def read_deployment_log(self, deployment_id: str, offset: int = 0) -> tuple[str, int, bool]:
        """
        读取部署日志（增量）

        Args:
            deployment_id: 部署 ID
            offset: 读取偏移量

        Returns:
            tuple[str, int, bool]: (日志内容, 当前偏移量, 是否完成)
        """
        deployment = self.deployment_repo.find_by_id(deployment_id)
        if not deployment:
            raise BusinessError(f"Deployment {deployment_id} not found", status_code=404)

        if not deployment.application_id:
            raise BusinessError(f"Deployment {deployment_id} has no associated application", status_code=400)

        # 获取应用信息
        app = self.app_repo.find_by_id(deployment.application_id)
        if not app:
            raise BusinessError(f"Application {deployment.application_id} not found", status_code=404)

        # 读取日志
        logs, current_offset = self.app_manager.read_deployment_log(app.code, deployment_id, offset)

        # 判断是否完成
        is_complete = deployment.status in (DeployStatus.RAN_TO_COMPLETION, DeployStatus.FAULTED, DeployStatus.CANCELED)

        return logs, current_offset, is_complete
