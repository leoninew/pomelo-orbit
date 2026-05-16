from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.infrastructure.persistence.models import ProjectModel


class ProjectMapper:
    @staticmethod
    def to_domain(model: ProjectModel) -> Project:
        return Project(
            id=model.id,
            name=model.name,
            code=model.code,
            owner_user_id=model.owner_user_id,
            is_active=model.is_active,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: Project) -> ProjectModel:
        return ProjectModel(
            id=entity.id,
            name=entity.name,
            code=entity.code,
            owner_user_id=entity.owner_user_id,
            is_active=entity.is_active,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )
