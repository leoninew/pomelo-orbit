from sqlalchemy.orm import Session

from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.models import ProjectModel
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

    def find_by_owner_and_code(self, owner_user_id: str, code: str) -> Project | None:
        orm = (
            self._session.query(ProjectModel)
            .filter(ProjectModel.owner_user_id == owner_user_id, ProjectModel.code == code)
            .first()
        )
        return ProjectMapper.to_domain(orm) if orm else None
