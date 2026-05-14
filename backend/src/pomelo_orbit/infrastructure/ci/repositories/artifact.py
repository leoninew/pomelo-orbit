"""CI 制品仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Artifact
from pomelo_orbit.domain.ci.repositories import ArtifactRepository
from pomelo_orbit.infrastructure.ci.mappers import ArtifactMapper
from pomelo_orbit.infrastructure.ci.models import ArtifactModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class ArtifactRepositoryImpl(BaseRepository[Artifact, ArtifactModel], ArtifactRepository):
    """制品仓储"""

    def __init__(self, session: Session):
        super().__init__(session, ArtifactModel, ArtifactMapper())

    def find_by_run(self, project_id: str, pipeline_run_id: str) -> list[Artifact]:
        """查询 pipeline run 的所有制品"""
        orms = (
            self._session.query(ArtifactModel)
            .filter(ArtifactModel.project_id == project_id, ArtifactModel.pipeline_run_id == pipeline_run_id)
            .order_by(ArtifactModel.created_at)
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_paginated_by_project_id(
        self,
        project_id: str,
        page: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
        search: str | None = None,
    ) -> tuple[list[Artifact], int]:
        """分页查询制品列表"""
        query = self._session.query(ArtifactModel).filter(ArtifactModel.project_id == project_id)
        if repository_id:
            query = query.filter(ArtifactModel.repository_id == repository_id)
        if template_id:
            query = query.filter(ArtifactModel.template_id == template_id)
        if search:
            query = query.filter(ArtifactModel.name.ilike(f"%{search}%") | ArtifactModel.path.ilike(f"%{search}%"))
        total = query.count()
        orms = query.order_by(ArtifactModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total
