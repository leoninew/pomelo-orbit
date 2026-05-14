import pytest

from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now
from pomelo_orbit.interfaces.api.project.dependencies import get_current_project


class TestProjectDependency:
    def test_get_current_project_uses_project_id_query_value(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                owner_user_id="user-1",
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )
        service = ProjectService(repo)
        user = User(id="user-1", username="user", password_hash="")

        project = get_current_project("project-1", user, service)

        assert project.id == "project-1"

    def test_get_current_project_rejects_foreign_project(self, db_session):
        from pomelo_orbit.domain.exceptions import BusinessError

        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                owner_user_id="user-1",
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )
        service = ProjectService(repo)
        user = User(id="user-2", username="user", password_hash="")

        with pytest.raises(BusinessError):
            get_current_project("project-1", user, service)
