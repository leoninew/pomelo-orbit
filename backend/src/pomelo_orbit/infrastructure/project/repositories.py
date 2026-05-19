from sqlalchemy import func
from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.ci.models import RepositoryModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import UserMapper
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel, ProjectMemberModel, ProjectModel, UserModel
from pomelo_orbit.infrastructure.project.mappers import ProjectMapper


class ProjectRepositoryImpl(BaseRepository[Project, ProjectModel], ProjectRepository):
    def __init__(self, db: Session):
        super().__init__(db, ProjectModel, ProjectMapper)

    def find_by_member(self, user_id: str) -> list[Project]:
        orms = (
            self._session.query(ProjectModel)
            .join(ProjectMemberModel, ProjectMemberModel.project_id == ProjectModel.id)
            .filter(ProjectMemberModel.user_id == user_id)
            .order_by(ProjectModel.created_at.desc())
            .all()
        )
        return [ProjectMapper.to_domain(orm) for orm in orms]

    def find_active_by_member(self, user_id: str) -> list[Project]:
        orms = (
            self._session.query(ProjectModel)
            .join(ProjectMemberModel, ProjectMemberModel.project_id == ProjectModel.id)
            .filter(ProjectMemberModel.user_id == user_id, ProjectModel.is_active)
            .order_by(ProjectModel.created_at.desc())
            .all()
        )
        return [ProjectMapper.to_domain(orm) for orm in orms]

    def find_by_code(self, code: str) -> Project | None:
        orm = self._session.query(ProjectModel).filter(ProjectModel.code == code).first()
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

    def is_member(self, project_id: str, user_id: str) -> bool:
        return (
            self._session.query(ProjectMemberModel)
            .filter(ProjectMemberModel.project_id == project_id, ProjectMemberModel.user_id == user_id)
            .first()
            is not None
        )

    def list_members(self, project_id: str) -> list[User]:
        orms = (
            self._session.query(UserModel)
            .join(ProjectMemberModel, ProjectMemberModel.user_id == UserModel.id)
            .filter(ProjectMemberModel.project_id == project_id)
            .order_by(UserModel.username.asc())
            .all()
        )
        return [UserMapper.to_domain(orm) for orm in orms]

    def add_member(self, project_id: str, user_id: str) -> None:
        if self.is_member(project_id, user_id):
            return
        self._session.add(ProjectMemberModel(project_id=project_id, user_id=user_id))
        self._session.commit()

    def remove_member(self, project_id: str, user_id: str) -> None:
        self._session.query(ProjectMemberModel).filter(
            ProjectMemberModel.project_id == project_id,
            ProjectMemberModel.user_id == user_id,
        ).delete()
        self._session.commit()
