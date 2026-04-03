"""CI Repositories 单元测试"""

from unittest.mock import Mock

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    JobLogModel,
    JobModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    CredentialRepositoryImpl,
    JobLogRepositoryImpl,
    JobRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    ProjectRepositoryImpl,
)


class TestCredentialRepository:
    """测试 CredentialRepository"""

    def test_is_referenced_by_projects_true(self):
        """测试凭据被项目引用"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = ProjectModel()

        repo = CredentialRepositoryImpl(session)
        result = repo.is_referenced_by_projects("cred-1")

        assert result is True
        session.query.assert_called_once_with(ProjectModel)

    def test_is_referenced_by_projects_false(self):
        """测试凭据未被项目引用"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None

        repo = CredentialRepositoryImpl(session)
        result = repo.is_referenced_by_projects("cred-1")

        assert result is False


class TestPipelineTemplateRepository:
    """测试 PipelineTemplateRepository"""

    def test_is_referenced_by_projects_true(self):
        """测试模板被项目引用"""
        session = Mock()
        query_mock = Mock()
        join_mock = Mock()
        session.query.return_value = query_mock
        query_mock.join.return_value = join_mock
        join_mock.filter.return_value = join_mock
        join_mock.first.return_value = ProjectModel()

        repo = PipelineTemplateRepositoryImpl(session)
        result = repo.is_referenced_by_projects("template-1")

        assert result is True

    def test_is_referenced_by_projects_false(self):
        """测试模板未被项目引用"""
        session = Mock()
        query_mock = Mock()
        join_mock = Mock()
        session.query.return_value = query_mock
        query_mock.join.return_value = join_mock
        join_mock.filter.return_value = join_mock
        join_mock.first.return_value = None

        repo = PipelineTemplateRepositoryImpl(session)
        result = repo.is_referenced_by_projects("template-1")

        assert result is False

    def test_find_builtin(self):
        """测试查询内置模板"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock

        # 模拟返回的 ORM 对象
        template_orm = Mock(spec=PipelineTemplateModel)
        template_orm.id = "template-1"
        template_orm.name = "Builtin Template"
        template_orm.is_builtin = 1
        query_mock.all.return_value = [template_orm]

        repo = PipelineTemplateRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=PipelineTemplate)

        result = repo.find_builtin()

        assert len(result) == 1
        query_mock.filter.assert_called_once()


class TestProjectRepository:
    """测试 ProjectRepository"""

    def test_has_running_pipelines_true(self):
        """测试项目有运行中的 pipeline"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = PipelineRunModel()

        repo = ProjectRepositoryImpl(session)
        result = repo.has_running_pipelines("project-1")

        assert result is True

    def test_has_running_pipelines_false(self):
        """测试项目没有运行中的 pipeline"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None

        repo = ProjectRepositoryImpl(session)
        result = repo.has_running_pipelines("project-1")

        assert result is False

    def test_find_by_repository_url(self):
        """测试根据仓库 URL 查找项目"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock

        # 模拟返回的 ORM 对象
        project_orm = Mock(spec=ProjectModel)
        project_orm.id = "project-1"
        project_orm.repository_url = "https://github.com/user/repo.git"
        query_mock.all.return_value = [project_orm]

        repo = ProjectRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Project)

        result = repo.find_by_repository_url("https://github.com/user/repo.git")

        assert len(result) == 1

    def test_find_by_repository_url_empty(self):
        """测试根据仓库 URL 查找项目返回空列表"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.all.return_value = []

        repo = ProjectRepositoryImpl(session)
        repo._mapper = Mock()

        result = repo.find_by_repository_url("https://github.com/user/nonexist.git")

        assert result == []


class TestPipelineRunRepository:
    """测试 PipelineRunRepository"""

    def test_find_by_project(self):
        """测试按项目查询运行列表"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        query_mock.count.return_value = 5
        query_mock.offset.return_value = query_mock
        query_mock.limit.return_value = query_mock

        # 模拟返回的 ORM 对象
        run_orm = Mock(spec=PipelineRunModel)
        run_orm.id = "run-1"
        query_mock.all.return_value = [run_orm]

        repo = PipelineRunRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=PipelineRun)

        runs, total = repo.find_paginated_with_filters(page=1, per_page=20, project_id="project-1")

        assert len(runs) == 1
        assert total == 5
        query_mock.offset.assert_called_once_with(0)
        query_mock.limit.assert_called_once_with(20)

    def test_find_by_project_pagination(self):
        """测试分页查询"""
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

        _, total = repo.find_paginated_with_filters(page=3, per_page=10, project_id="project-1")

        assert total == 100
        query_mock.offset.assert_called_once_with(20)  # (3-1) * 10
        query_mock.limit.assert_called_once_with(10)


class TestJobRepository:
    """测试 JobRepository"""

    def test_find_by_run(self):
        """测试查询 pipeline run 的所有 jobs"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock

        # 模拟返回的 ORM 对象
        job_orm = Mock(spec=JobModel)
        job_orm.id = "job-1"
        query_mock.all.return_value = [job_orm]

        repo = JobRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Job)

        result = repo.find_by_run("run-1")

        assert len(result) == 1

    def test_find_by_parent(self):
        """测试查询子 jobs"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock

        # 模拟返回的 ORM 对象
        job_orm = Mock(spec=JobModel)
        job_orm.id = "child-job-1"
        query_mock.all.return_value = [job_orm]

        repo = JobRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Job)

        result = repo.find_by_parent("parent-job-1")

        assert len(result) == 1


class TestJobLogRepository:
    """测试 JobLogRepository"""

    def test_find_by_job_exists(self):
        """测试查询 job 的日志存在"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock

        # 模拟返回的 ORM 对象
        log_orm = Mock(spec=JobLogModel)
        log_orm.id = "log-1"
        query_mock.first.return_value = log_orm

        repo = JobLogRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=JobLog)

        result = repo.find_by_job("job-1")

        assert result is not None

    def test_find_by_job_not_exists(self):
        """测试查询 job 的日志不存在"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.first.return_value = None

        repo = JobLogRepositoryImpl(session)
        repo._mapper = Mock()

        result = repo.find_by_job("job-1")

        assert result is None


class TestArtifactRepository:
    """测试 ArtifactRepository"""

    def test_find_by_run(self):
        """测试查询 pipeline run 的所有制品"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock

        # 模拟返回的 ORM 对象
        artifact_orm = Mock(spec=ArtifactModel)
        artifact_orm.id = "artifact-1"
        query_mock.all.return_value = [artifact_orm]

        repo = ArtifactRepositoryImpl(session)

        # Mock mapper
        repo._mapper = Mock()
        repo._mapper.to_domain.return_value = Mock(spec=Artifact)

        result = repo.find_by_run("run-1")

        assert len(result) == 1
        query_mock.order_by.assert_called_once()

    def test_find_by_run_empty(self):
        """测试查询 pipeline run 的制品为空"""
        session = Mock()
        query_mock = Mock()
        session.query.return_value = query_mock
        query_mock.filter.return_value = query_mock
        query_mock.order_by.return_value = query_mock
        query_mock.all.return_value = []

        repo = ArtifactRepositoryImpl(session)
        repo._mapper = Mock()

        result = repo.find_by_run("run-1")

        assert result == []
