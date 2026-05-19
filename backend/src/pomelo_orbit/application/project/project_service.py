import ulid

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.time_utils import utc_now


class ProjectService:
    def __init__(self, project_repo: ProjectRepository):
        self.project_repo = project_repo

    def list_projects(self, user_id: str) -> list[Project]:
        return self.project_repo.find_by_member(user_id)

    def get_project(self, user_id: str, project_id: str) -> Project:
        project = self.project_repo.find_by_id(project_id)
        if not project:
            raise BusinessError(f"Project {project_id} not found", status_code=404)
        if not self.project_repo.is_member(project_id, user_id):
            raise BusinessError("Permission denied", status_code=403)
        return project

    def create_project(self, user_id: str, name: str, code: str) -> Project:
        existing = self.project_repo.find_by_code(code)
        if existing:
            raise BusinessError(f"Project code {code} already exists", status_code=409)
        now = utc_now()
        project = Project(
            id=str(ulid.ULID()),
            name=name,
            code=code,
            is_active=True,
            created_at=now,
            updated_at=now,
        )
        self.project_repo.save(project)
        self.project_repo.add_member(project.id, user_id)
        return project

    def update_project(self, user_id: str, project_id: str, name: str, code: str) -> Project:
        project = self.get_project(user_id, project_id)
        existing = self.project_repo.find_by_code(code)
        if existing and existing.id != project_id:
            raise BusinessError(f"Project code {code} already exists", status_code=409)
        project.name = name
        project.code = code
        project.updated_at = utc_now()
        self.project_repo.save(project)
        return project

    def deprecate_project(self, user_id: str, project_id: str) -> None:
        project = self.get_project(user_id, project_id)

        # Check if it's the last active project
        active_projects = self.project_repo.find_active_by_member(user_id)
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

    def list_members(self, user_id: str, project_id: str) -> list[User]:
        self.get_project(user_id, project_id)
        return self.project_repo.list_members(project_id)

    def add_member(self, user_id: str, project_id: str, member_user_id: str) -> list[User]:
        self.get_project(user_id, project_id)
        self.project_repo.add_member(project_id, member_user_id)
        return self.project_repo.list_members(project_id)

    def remove_member(self, user_id: str, project_id: str, member_user_id: str) -> list[User]:
        self.get_project(user_id, project_id)
        self.project_repo.remove_member(project_id, member_user_id)
        return self.project_repo.list_members(project_id)
