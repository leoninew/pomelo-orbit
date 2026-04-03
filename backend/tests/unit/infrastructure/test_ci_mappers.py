"""CI Mappers 单元测试"""

from ulid import ULID

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineRunStatus,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.ci.mappers import (
    ArtifactMapper,
    CredentialMapper,
    JobLogMapper,
    JobMapper,
    PipelineRunMapper,
    PipelineTemplateMapper,
    ProjectMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    JobLogModel,
    JobModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestCredentialMapper:
    """测试 CredentialMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = CredentialModel(
            id=str(ULID()),
            name="test-cred",
            type="git_ssh",
            encrypted_data="encrypted",
            created_at=utc_now(),
        )

        entity = CredentialMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.type == CredentialType.GIT_SSH
        assert entity.encrypted_data == orm.encrypted_data
        assert entity.created_at == orm.created_at

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = Credential.create(
            name="test-cred",
            type=CredentialType.GIT_TOKEN,
            encrypted_data="encrypted",
        )

        orm = CredentialMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.type == "git_token"
        assert orm.encrypted_data == entity.encrypted_data
        assert orm.created_at == entity.created_at


class TestPipelineTemplateMapper:
    """测试 PipelineTemplateMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = PipelineTemplateModel(
            id=str(ULID()),
            name="test-template",
            description="Test template",
            stages='[{"name": "build", "type": "checkout", "config": {"ref": "master"}}]',
            variable_declarations='[{"name": "VAR1", "default": "value1"}]',
            is_builtin=1,
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        entity = PipelineTemplateMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.description == orm.description
        assert len(entity.stages) == 1
        assert entity.stages[0].name == "build"
        assert entity.stages[0].type.value == "checkout"
        assert len(entity.variable_declarations) == 1
        assert entity.variable_declarations[0].name == "VAR1"
        assert entity.variable_declarations[0].default == "value1"
        assert entity.is_builtin is True

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        from pomelo_orbit.domain.ci.value_objects import CheckoutStageConfig, StageDefinition, StageType

        entity = PipelineTemplate.create(
            name="test-template",
            description="Test template",
            stages=[
                StageDefinition(
                    name="build",
                    type=StageType.CHECKOUT,
                    config=CheckoutStageConfig(ref="master"),
                )
            ],
            variable_declarations=[
                VariableDeclaration(name="VAR1", default="value1"),
                VariableDeclaration(name="VAR2", default="value2"),
            ],
            is_builtin=False,
        )

        orm = PipelineTemplateMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.description == entity.description
        assert orm.is_builtin == 0
        assert '"name": "VAR1"' in orm.variable_declarations
        assert '"name": "VAR2"' in orm.variable_declarations


class TestProjectMapper:
    """测试 ProjectMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = ProjectModel(
            id=str(ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo",
            pipeline_snapshot_id=str(ULID()),
            git_credential_id=str(ULID()),
            variable_overrides='{"VAR1": "override1"}',
            default_branch="main",
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        entity = ProjectMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.repository_url == orm.repository_url
        assert entity.pipeline_snapshot_id == orm.pipeline_snapshot_id
        assert entity.git_credential_id == orm.git_credential_id
        assert entity.variable_overrides == {"VAR1": "override1"}
        assert entity.default_branch == "main"

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo",
            pipeline_snapshot_id=str(ULID()),
            git_credential_id=str(ULID()),
            variable_overrides={"VAR1": "override1", "VAR2": "override2"},
            default_branch="main",
        )

        orm = ProjectMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.repository_url == entity.repository_url
        assert '"VAR1": "override1"' in orm.variable_overrides
        assert '"VAR2": "override2"' in orm.variable_overrides


class TestPipelineRunMapper:
    """测试 PipelineRunMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = PipelineRunModel(
            id=str(ULID()),
            project_id=str(ULID()),
            pipeline_snapshot_id=str(ULID()),
            trigger="manual",
            trigger_ref="main",
            variables_snapshot='{"VAR1": "value1"}',
            status="running",
            retry_of=None,
            started_at=utc_now(),
            finished_at=None,
            created_at=utc_now(),
        )

        entity = PipelineRunMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.project_id == orm.project_id
        assert entity.pipeline_snapshot_id == orm.pipeline_snapshot_id
        assert entity.trigger == PipelineRunTrigger.MANUAL
        assert entity.trigger_ref == orm.trigger_ref
        assert entity.variables_snapshot == {"VAR1": "value1"}
        assert entity.status == PipelineRunStatus.RUNNING

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = PipelineRun.create(
            project_id=str(ULID()),
            pipeline_snapshot_id=str(ULID()),
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="feature/test",
            variables_snapshot={"VAR1": "value1", "VAR2": "value2"},
        )

        orm = PipelineRunMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.project_id == entity.project_id
        assert orm.pipeline_snapshot_id == entity.pipeline_snapshot_id
        assert orm.trigger == "webhook"
        assert orm.trigger_ref == entity.trigger_ref
        assert '"VAR1": "value1"' in orm.variables_snapshot


class TestJobMapper:
    """测试 JobMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = JobModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            name="build",
            parent_job_id=None,
            status="success",
            started_at=utc_now(),
            finished_at=utc_now(),
            exit_code=0,
            error_message=None,
        )

        entity = JobMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.pipeline_run_id == orm.pipeline_run_id
        assert entity.name == orm.name
        assert entity.status == JobStatus.SUCCESS
        assert entity.exit_code == 0

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = Job.create(
            pipeline_run_id=str(ULID()),
            name="test",
            parent_job_id=str(ULID()),
        )

        orm = JobMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.pipeline_run_id == entity.pipeline_run_id
        assert orm.name == entity.name
        assert orm.parent_job_id == entity.parent_job_id
        assert orm.status == "waiting"


class TestJobLogMapper:
    """测试 JobLogMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = JobLogModel(
            id=str(ULID()),
            job_id=str(ULID()),
            content="Log content",
            created_at=utc_now(),
        )

        entity = JobLogMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.job_id == orm.job_id
        assert entity.content == orm.content
        assert entity.created_at == orm.created_at

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = JobLog.create(
            job_id=str(ULID()),
            content="Log content",
        )

        orm = JobLogMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.job_id == entity.job_id
        assert orm.content == entity.content


class TestArtifactMapper:
    """测试 ArtifactMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = ArtifactModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            job_name="build",
            type="file",
            name="output.txt",
            path="/artifacts/output.txt",
            created_at=utc_now(),
        )

        entity = ArtifactMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.pipeline_run_id == orm.pipeline_run_id
        assert entity.job_name == orm.job_name
        assert entity.type == orm.type
        assert entity.name == orm.name
        assert entity.path == orm.path

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = Artifact.create(
            pipeline_run_id=str(ULID()),
            job_name="build",
            artifact_type="docker_image",
            name="myapp:latest",
            path="/artifacts/myapp-latest.tar",
        )

        orm = ArtifactMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.pipeline_run_id == entity.pipeline_run_id
        assert orm.job_name == entity.job_name
        assert orm.type == "docker_image"
        assert orm.name == entity.name
        assert orm.path == entity.path
