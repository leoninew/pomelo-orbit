import ulid

from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.time_utils import utc_now


class ProjectService:
    def __init__(self, project_repo: ProjectRepository):
        self.project_repo = project_repo

    def list_projects(self, owner_user_id: str) -> list[Project]:
        return self.project_repo.find_by_owner(owner_user_id)

    def get_project(self, owner_user_id: str, project_id: str) -> Project:
        project = self.project_repo.find_by_id(project_id)
        if not project:
            raise BusinessError(f"Project {project_id} not found", status_code=404)
        if project.owner_user_id != owner_user_id:
            raise BusinessError(f"Project {project_id} owner not match", status_code=400)
        return project

    def create_project(self, owner_user_id: str, name: str, code: str) -> Project:
        existing = self.project_repo.find_by_owner_and_code(owner_user_id, code)
        if existing:
            raise BusinessError(f"Project code {code} already exists", status_code=409)
        now = utc_now()
        project = Project(
            id=str(ulid.ULID()),
            name=name,
            code=code,
            owner_user_id=owner_user_id,
            is_active=True,
            created_at=now,
            updated_at=now,
        )
        self.project_repo.save(project)
        return project

    def update_project(self, owner_user_id: str, project_id: str, name: str, code: str) -> Project:
        project = self.get_project(owner_user_id, project_id)
        existing = self.project_repo.find_by_owner_and_code(owner_user_id, code)
        if existing and existing.id != project_id:
            raise BusinessError(f"Project code {code} already exists", status_code=409)
        project.name = name
        project.code = code
        project.updated_at = utc_now()
        self.project_repo.save(project)
        return project

    def deprecate_project(self, owner_user_id: str, project_id: str) -> None:
        project = self.get_project(owner_user_id, project_id)

        # Check if it's the last active project
        active_projects = self.project_repo.find_active_by_owner(owner_user_id)
        if len(active_projects) <= 1:
            raise BusinessError("Cannot deprecate the last active project", status_code=400)

        # Check if project has repositories
        repo_count = self.project_repo.count_repositories(project_id)
        if repo_count > 0:
            raise BusinessError(f"Cannot deprecate project with {repo_count} repositories", status_code=400)

        # Check if project has applications
        app_count = self.project_repo.count_applications(project_id)
        if app_count > 0:
            raise BusinessError(f"Cannot deprecate project with {app_count} applications", status_code=400)

        project.is_active = False
        project.updated_at = utc_now()
        self.project_repo.save(project)
