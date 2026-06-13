"""测试 CI Repositories"""

import ulid
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.infrastructure.ci.models import Base
from pomelo_orbit.infrastructure.ci.repositories import RepositoryRepositoryImpl

PROJECT_ID = "project-1"


class TestProjectRepository:
    def setup_method(self):
        self.engine = create_engine("sqlite:///:memory:")
        Base.metadata.create_all(self.engine)
        session_local = sessionmaker(bind=self.engine)
        self.session = session_local()
        self.repo = RepositoryRepositoryImpl(self.session)

    def teardown_method(self):
        self.session.close()
        Base.metadata.drop_all(self.engine)

    def test_save_and_find_by_id(self):
        project = Repository.create(
            project_id=PROJECT_ID,
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            default_branch="main",
        )

        self.repo.save(project)
        self.session.commit()

        found = self.repo.find_by_id(project.id)

        assert found is not None
        assert found.id == project.id
        assert found.name == project.name
        assert found.default_branch == "main"

    def test_find_by_repository_url(self):
        repo_url = "https://github.com/test/repo.git"

        project1 = Repository.create(project_id=PROJECT_ID, name="project1", code="project1", repository_url=repo_url)
        project2 = Repository.create(project_id=PROJECT_ID, name="project2", code="project2", repository_url=repo_url)
        project3 = Repository.create(
            project_id=PROJECT_ID,
            name="project3",
            code="project3",
            repository_url="https://github.com/other/repo.git",
        )

        self.repo.save(project1)
        self.repo.save(project2)
        self.repo.save(project3)
        self.session.commit()

        found = self.repo.find_by_repository_url(PROJECT_ID, repo_url)

        assert len(found) == 2
        assert {p.name for p in found} == {"project1", "project2"}

    def test_find_by_repository_url_not_found(self):
        found = self.repo.find_by_repository_url(PROJECT_ID, "https://github.com/nonexistent/repo.git")
        assert len(found) == 0

    def test_update_project(self):
        project = Repository.create(
            project_id=PROJECT_ID,
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
        )

        self.repo.save(project)
        self.session.commit()

        project.update(name="updated-project")
        self.repo.save(project)
        self.session.commit()

        found = self.repo.find_by_id(project.id)

        assert found is not None
        assert found.name == "updated-project"
