"""CI 项目仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Project
from pomelo_orbit.domain.ci.repositories import ProjectRepository
from pomelo_orbit.infrastructure.ci.mappers import ProjectMapper
from pomelo_orbit.infrastructure.ci.models import PipelineRunModel, ProjectModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class ProjectRepositoryImpl(BaseRepository[Project, ProjectModel], ProjectRepository):
    """项目仓储"""

    def __init__(self, session: Session):
        super().__init__(session, ProjectModel, ProjectMapper())

    def has_running_pipelines(self, project_id: str) -> bool:
        """检查项目是否有运行中的 pipeline"""
        return (
            self._session.query(PipelineRunModel)
            .filter(
                PipelineRunModel.project_id == project_id,
                PipelineRunModel.status.in_(["waiting_to_run", "running"]),
            )
            .first()
            is not None
        )

    def find_by_repository_url(self, repository_url: str) -> list[Project]:
        """根据仓库 URL 查找项目（精确匹配，多个项目可能共用同一仓库）"""
        orms = self._session.query(ProjectModel).filter(ProjectModel.repository_url == repository_url).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_code(self, code: str) -> Project | None:
        """根据项目编码查找项目"""
        orm = self._session.query(ProjectModel).filter(ProjectModel.code == code).first()
        return self._mapper.to_domain(orm) if orm else None
