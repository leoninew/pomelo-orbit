from sqlalchemy import func
from sqlalchemy.orm import Session

from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.ci.models import RepositoryModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel, ProjectModel
from pomelo_orbit.infrastructure.project.mappers import ProjectMapper


class ProjectRepositoryImpl(BaseRepository[Project, ProjectModel], ProjectRepository):
    def __init__(self, db: Session):
        super().__init__(db, ProjectModel, ProjectMapper)

    def find_by_owner(self, owner_user_id: str) -> list[Project]:
        orms = (
            self._session.query(ProjectModel)
            .filter(ProjectModel.owner_user_id == owner_user_id)
            .order_by(ProjectModel.created_at.desc())
            .all()
        )
        return [ProjectMapper.to_domain(orm) for orm in orms]

    def find_active_by_owner(self, owner_user_id: str) -> list[Project]:
        orms = (
            self._session.query(ProjectModel)
            .filter(ProjectModel.owner_user_id == owner_user_id, ProjectModel.is_active)
            .order_by(ProjectModel.created_at.desc())
            .all()
        )
        return [ProjectMapper.to_domain(orm) for orm in orms]

    def find_by_owner_and_code(self, owner_user_id: str, code: str) -> Project | None:
        orm = (
            self._session.query(ProjectModel)
            .filter(ProjectModel.owner_user_id == owner_user_id, ProjectModel.code == code)
            .first()
        )
        return ProjectMapper.to_domain(orm) if orm else None

    def count_repositories(self, project_id: str) -> int:
        return (
            self._session.query(func.count(RepositoryModel.id))
            .filter(RepositoryModel.project_id == project_id)
            .scalar()
            or 0
        )

    def count_applications(self, project_id: str) -> int:
        return (
            self._session.query(func.count(ApplicationModel.id))
            .filter(ApplicationModel.project_id == project_id)
            .scalar()
            or 0
        )
