"""CI 项目仓储实现"""

from sqlalchemy import or_
from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.domain.ci.repositories import RepositoryRepository
from pomelo_orbit.infrastructure.ci.mappers import RepositoryMapper
from pomelo_orbit.infrastructure.ci.models import PipelineRunModel, RepositoryModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class RepositoryRepositoryImpl(BaseRepository[Repository, RepositoryModel], RepositoryRepository):
    """项目仓储"""

    def __init__(self, session: Session):
        super().__init__(session, RepositoryModel, RepositoryMapper())

    def find_paginated_by_project_id(
        self, project_id: str, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[Repository], int]:
        """分页查询，支持按名称或仓库地址搜索"""
        query = self._session.query(RepositoryModel).filter(RepositoryModel.project_id == project_id)
        if search:
            pattern = f"%{search}%"
            query = query.filter(
                or_(
                    RepositoryModel.name.ilike(pattern),
                    RepositoryModel.repository_url.ilike(pattern),
                )
            )
        total = query.count()
        orms = query.order_by(RepositoryModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total

    def has_running_pipelines(self, project_id: str, repository_id: str) -> bool:
        """检查项目是否有运行中的 pipeline"""
        return (
            self._session.query(PipelineRunModel)
            .filter(
                PipelineRunModel.project_id == project_id,
                PipelineRunModel.repository_id == repository_id,
                PipelineRunModel.status.in_(["waiting_to_run", "running"]),
            )
            .first()
            is not None
        )

    def find_by_repository_url(self, project_id: str, repository_url: str) -> list[Repository]:
        """根据仓库 URL 查找项目（精确匹配，多个项目可能共用同一仓库）"""
        orms = (
            self._session.query(RepositoryModel)
            .filter(RepositoryModel.project_id == project_id, RepositoryModel.repository_url == repository_url)
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_code(self, project_id: str, code: str) -> Repository | None:
        """根据项目编码查找项目"""
        orm = (
            self._session.query(RepositoryModel)
            .filter(RepositoryModel.project_id == project_id, RepositoryModel.code == code)
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None
