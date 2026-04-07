"""
部署记录仓储实现
"""

from datetime import datetime
from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import Deployment
from pomelo_orbit.domain.cd.repositories import DeploymentRepository
from pomelo_orbit.domain.cd.value_objects import OperationType, TaskStatus
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.mappers import DeploymentMapper
from pomelo_orbit.infrastructure.persistence.models import DeploymentModel


class DeploymentRepositoryImpl(BaseRepository[Deployment, DeploymentModel], DeploymentRepository):
    """部署记录仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, DeploymentModel, DeploymentMapper)

    def find_by_application(self, app_id: str, page: int = 1, per_page: int = 20) -> tuple[list[Deployment], int]:
        """分页查询应用的部署记录"""
        query = self._session.query(DeploymentModel).filter(DeploymentModel.application_id == app_id)
        total = query.count()
        offset = (page - 1) * per_page
        orms = query.order_by(DeploymentModel.started_at.desc()).offset(offset).limit(per_page).all()
        return [DeploymentMapper.to_domain(orm) for orm in orms], total

    def find_last_successful_deploy(self, app_id: str) -> Deployment | None:
        """查找最近一次成功的部署记录"""
        orm = (
            self._session.query(DeploymentModel)
            .filter(
                DeploymentModel.application_id == app_id,
                DeploymentModel.operation_type == OperationType.DEPLOY,
                DeploymentModel.status == TaskStatus.RAN_TO_COMPLETION.value,
            )
            .order_by(DeploymentModel.started_at.desc())
            .first()
        )
        return DeploymentMapper.to_domain(orm) if orm else None

    def find_paginated_with_filters(
        self,
        page: int,
        per_page: int,
        application_id: str | None = None,
        status: str | None = None,
        search: str | None = None,
        date_from: datetime | None = None,
        date_to: datetime | None = None,
    ) -> tuple[list[Deployment], int]:
        """分页查询部署记录（支持过滤）"""
        query = self._session.query(DeploymentModel)

        if application_id:
            query = query.filter(DeploymentModel.application_id == application_id)
        if status:
            query = query.filter(DeploymentModel.status == status)
        if search:
            query = query.filter(DeploymentModel.application_name.contains(search))
        if date_from:
            query = query.filter(DeploymentModel.started_at >= date_from)
        if date_to:
            query = query.filter(DeploymentModel.started_at < date_to)

        total = query.count()
        offset = (page - 1) * per_page
        orms = query.order_by(DeploymentModel.started_at.desc()).offset(offset).limit(per_page).all()

        return [DeploymentMapper.to_domain(orm) for orm in orms], total


def get_deployment_repository(db: Annotated[Session, Depends(get_db)]) -> DeploymentRepository:
    """获取部署记录仓储实例（依赖注入）- 返回接口类型"""
    return DeploymentRepositoryImpl(db)
