from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.infrastructure.persistence.models import UserModel
from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestProjectRepository:
    def test_save_and_find_by_member(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        project = Project(
            id="project-1",
            name="Project One",
            code="project-one",
            is_active=True,
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        repo.save(project)
        repo.add_member("project-1", "user-1")
        projects = repo.find_by_member("user-1")

        assert len(projects) == 1
        assert projects[0].id == "project-1"

    def test_find_by_id_returns_project(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                is_active=True,
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )

        assert repo.find_by_id("project-1") is not None
        assert repo.find_by_id("non-existent") is None

    def test_find_by_code(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                is_active=True,
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )

        found = repo.find_by_code("project-one")

        assert found is not None
        assert found.id == "project-1"

    def test_list_members(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        db_session.add(UserModel(id="user-1", username="alice", password_hash="hashed", status="enabled"))
        db_session.commit()
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                is_active=True,
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )
        repo.add_member("project-1", "user-1")

        members = repo.list_members("project-1")

        assert len(members) == 1
        assert members[0].username == "alice"

    def test_remove_member(self, db_session):
        repo = ProjectRepositoryImpl(db_session)
        db_session.add(UserModel(id="user-1", username="alice", password_hash="hashed", status="enabled"))
        db_session.commit()
        repo.save(
            Project(
                id="project-1",
                name="Project One",
                code="project-one",
                is_active=True,
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )
        repo.add_member("project-1", "user-1")

        repo.remove_member("project-1", "user-1")

        assert repo.list_members("project-1") == []
