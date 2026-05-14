from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestProjectRepository:
    def test_save_and_find_by_owner(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        project = Project(
            id="project-1",
            name="Project One",
            code="project-one",
            owner_user_id="user-1",
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        repo.save(project)
        projects = repo.find_by_owner("user-1")

        assert len(projects) == 1
        assert projects[0].id == "project-1"

    def test_find_by_owner_and_id_returns_only_owned_project(self, db_session):
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

        assert repo.find_by_owner_and_id("user-1", "project-1") is not None
        assert repo.find_by_owner_and_id("user-2", "project-1") is None

    def test_find_by_owner_and_code(self, db_session):
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

        found = repo.find_by_owner_and_code("user-1", "project-one")

        assert found is not None
        assert found.id == "project-1"
