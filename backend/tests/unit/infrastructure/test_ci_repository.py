"""CI Repositories 单元测试"""

from unittest.mock import Mock

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    PipelineRun,
    Repository,
    StageRun,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    PipelineRunModel,
    ProjectModel,
    StageRunModel,
)
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    CredentialRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    RepositoryRepositoryImpl,
    StageRunRepositoryImpl,
)


class TestCredentialRepository:
    def test_is_referenced_by_projects_true(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = ProjectModel()
        assert CredentialRepositoryImpl(session).is_referenced_by_projects("cred-1") is True

    def test_is_referenced_by_projects_false(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None
        assert CredentialRepositoryImpl(session).is_referenced_by_projects("cred-1") is False


class TestPipelineTemplateRepository:
    def test_is_referenced_by_webhooks_true(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = Mock()
        assert PipelineTemplateRepositoryImpl(session).is_referenced_by_webhooks("template-1") is True

    def test_is_referenced_by_webhooks_false(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None
        assert PipelineTemplateRepositoryImpl(session).is_referenced_by_webhooks("template-1") is False


class TestProjectRepository:
    def test_has_running_pipelines_true(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = PipelineRunModel()
        assert RepositoryRepositoryImpl(session).has_running_pipelines("project-1") is True

    def test_has_running_pipelines_false(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None
        assert RepositoryRepositoryImpl(session).has_running_pipelines("project-1") is False

    def test_find_by_repository_url(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        project_orm = Mock(spec=ProjectModel)
        query_mock.all.return_value = [project_orm]
        repo = RepositoryRepositoryImpl(session)
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Repository)
        assert len(repo.find_by_repository_url("https://github.com/user/repo.git")) == 1

    def test_find_by_repository_url_empty(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.all.return_value = []
        repo = RepositoryRepositoryImpl(session)
        repo._mapper = Mock()
        assert repo.find_by_repository_url("https://github.com/user/nonexist.git") == []


class TestPipelineRunRepository:
    def test_find_by_project(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        query_mock.count.return_value = 5
        query_mock.offset.return_value = query_mock
        query_mock.limit.return_value = query_mock
        run_orm = Mock(spec=PipelineRunModel)
        query_mock.all.return_value = [run_orm]
        repo = PipelineRunRepositoryImpl(session)
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=PipelineRun)
        runs, total = repo.find_paginated_with_filters(page=1, per_page=20, repository_id="project-1")
        assert len(runs) == 1 and total == 5

    def test_find_by_project_pagination(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        query_mock.count.return_value = 100
        query_mock.offset.return_value = query_mock
        query_mock.limit.return_value = query_mock
        query_mock.all.return_value = []
        repo = PipelineRunRepositoryImpl(session)
        repo._mapper = Mock()
        _, total = repo.find_paginated_with_filters(page=3, per_page=10, repository_id="project-1")
        assert total == 100
        query_mock.offset.assert_called_once_with(20)


class TestStageRunRepository:
    def test_find_by_run(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        sr_orm = Mock(spec=StageRunModel)
        query_mock.all.return_value = [sr_orm]
        repo = StageRunRepositoryImpl(session)
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=StageRun)
        assert len(repo.find_by_run("run-1")) == 1


class TestArtifactRepository:
    def test_find_by_run(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        artifact_orm = Mock(spec=ArtifactModel)
        query_mock.all.return_value = [artifact_orm]
        repo = ArtifactRepositoryImpl(session)
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Artifact)
        assert len(repo.find_by_run("run-1")) == 1

    def test_find_by_run_empty(self):
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        query_mock.all.return_value = []
        repo = ArtifactRepositoryImpl(session)
        repo._mapper = Mock()
        assert repo.find_by_run("run-1") == []
