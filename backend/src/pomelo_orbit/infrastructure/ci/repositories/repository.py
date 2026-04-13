"""CI 项目仓储实现"""

from sqlalchemy import or_
from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.domain.ci.repositories import RepositoryRepository
from pomelo_orbit.infrastructure.ci.mappers import RepositoryMapper
from pomelo_orbit.infrastructure.ci.models import PipelineRunModel, ProjectModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class RepositoryRepositoryImpl(BaseRepository[Repository, ProjectModel], RepositoryRepository):
    """项目仓储"""

    def __init__(self, session: Session):
        super().__init__(session, ProjectModel, RepositoryMapper())

    def find_paginated(
        self, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[Repository], int]:
        """分页查询，支持按名称或仓库地址搜索"""
        query = self._session.query(ProjectModel)
        if search:
            pattern = f"%{search}%"
            query = query.filter(
                or_(
                    ProjectModel.name.ilike(pattern),
                    ProjectModel.repository_url.ilike(pattern),
                )
            )
        total = query.count()
        orms = query.order_by(ProjectModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total

    def has_running_pipelines(self, repository_id: str) -> bool:
        """检查项目是否有运行中的 pipeline"""
        return (
            self._session.query(PipelineRunModel)
            .filter(
                PipelineRunModel.repository_id == repository_id,
                PipelineRunModel.status.in_(["waiting_to_run", "running"]),
            )
            .first()
            is not None
        )

    def find_by_repository_url(self, repository_url: str) -> list[Repository]:
        """根据仓库 URL 查找项目（精确匹配，多个项目可能共用同一仓库）"""
        orms = self._session.query(ProjectModel).filter(ProjectModel.repository_url == repository_url).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_code(self, code: str) -> Repository | None:
        """根据项目编码查找项目"""
        orm = self._session.query(ProjectModel).filter(ProjectModel.code == code).first()
        return self._mapper.to_domain(orm) if orm else None
