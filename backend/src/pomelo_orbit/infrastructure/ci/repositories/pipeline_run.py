"""CI Pipeline 运行仓储实现"""

from datetime import datetime

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import PipelineRun
from pomelo_orbit.domain.ci.repositories import PipelineRunRepository
from pomelo_orbit.infrastructure.ci.mappers import PipelineRunMapper
from pomelo_orbit.infrastructure.ci.models import PipelineRunModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class PipelineRunRepositoryImpl(BaseRepository[PipelineRun, PipelineRunModel], PipelineRunRepository):
    """Pipeline 运行仓储"""

    def __init__(self, session: Session):
        super().__init__(session, PipelineRunModel, PipelineRunMapper())

    def find_by_id_in_project(self, project_id: str, run_id: str) -> PipelineRun | None:
        orm = (
            self._session.query(PipelineRunModel)
            .filter(PipelineRunModel.project_id == project_id, PipelineRunModel.id == run_id)
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_paginated_with_filters(
        self,
        project_id: str,
        page: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
        date_from: datetime | None = None,
        date_to: datetime | None = None,
    ) -> tuple[list[PipelineRun], int]:
        """分页查询运行列表（支持按项目/模板/日期过滤）"""
        query = self._session.query(PipelineRunModel).filter(PipelineRunModel.project_id == project_id)
        if repository_id:
            query = query.filter(PipelineRunModel.repository_id == repository_id)
        if template_id:
            query = query.filter(PipelineRunModel.template_id == template_id)
        if date_from:
            query = query.filter(PipelineRunModel.created_at >= date_from)
        if date_to:
            query = query.filter(PipelineRunModel.created_at < date_to)
        total = query.count()
        orms = query.order_by(PipelineRunModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total
