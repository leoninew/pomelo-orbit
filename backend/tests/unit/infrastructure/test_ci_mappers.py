"""CI Mappers 单元测试"""

from ulid import ULID

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    PipelineRun,
    PipelineTemplate,
    Project,
    StageLog,
    StageRun,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.ci.mappers import (
    ArtifactMapper,
    CredentialMapper,
    PipelineRunMapper,
    PipelineTemplateMapper,
    ProjectMapper,
    StageLogMapper,
    StageRunMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
    StageLogModel,
    StageRunModel,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestCredentialMapper:
    def test_to_domain(self):
        orm = CredentialModel(
            id=str(ULID()), name="test-cred", type="git_ssh", encrypted_data="encrypted", created_at=utc_now()
        )
        entity = CredentialMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.type == CredentialType.GIT_SSH

    def test_to_orm(self):
        entity = Credential.create(name="test-cred", type=CredentialType.GIT_TOKEN, encrypted_data="encrypted")
        orm = CredentialMapper.to_orm(entity)
        assert orm.id == entity.id
        assert orm.type == "git_token"


class TestPipelineTemplateMapper:
    def test_to_domain(self):
        orm = PipelineTemplateModel(
            id=str(ULID()),
            name="test-template",
            description="Test template",
            variable_declarations='[{"name": "VAR1", "default": "value1"}]',
            created_at=utc_now(),
            updated_at=utc_now(),
        )
        entity = PipelineTemplateMapper.to_domain(orm, orchestration=[], stages=[])
        assert entity.name == orm.name
        assert len(entity.variable_declarations) == 1

    def test_to_orm(self):
        entity = PipelineTemplate.create(
            name="test-template",
            description="Test template",
            variable_declarations=[VariableDeclaration(name="VAR1", default="value1")],
        )
        orm = PipelineTemplateMapper.to_orm(entity)
        assert orm.id == entity.id
        assert '"name": "VAR1"' in orm.variable_declarations


class TestProjectMapper:
    def test_to_domain(self):
        orm = ProjectModel(
            id=str(ULID()),
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo",
            git_credential_id=str(ULID()),
            variable_overrides='{"VAR1": "override1"}',
            default_branch="main",
            created_at=utc_now(),
            updated_at=utc_now(),
        )
        entity = ProjectMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.code == orm.code
        assert entity.variable_overrides == {"VAR1": "override1"}

    def test_to_orm(self):
        entity = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo",
            git_credential_id=str(ULID()),
            variable_overrides={"VAR1": "override1"},
            default_branch="main",
        )
        orm = ProjectMapper.to_orm(entity)
        assert orm.code == entity.code
        assert '"VAR1": "override1"' in orm.variable_overrides


class TestPipelineRunMapper:
    def test_to_domain(self):
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
        assert entity.trigger == PipelineRunTrigger.MANUAL
        assert entity.status == TaskStatus.RUNNING
        assert entity.variables_snapshot == {"VAR1": "value1"}

    def test_to_orm(self):
        entity = PipelineRun.create(
            project_id=str(ULID()),
            pipeline_snapshot_id=str(ULID()),
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="feature/test",
            variables_snapshot={"VAR1": "value1"},
        )
        orm = PipelineRunMapper.to_orm(entity)
        assert orm.trigger == "webhook"
        assert '"VAR1": "value1"' in orm.variables_snapshot


class TestStageRunMapper:
    def test_to_domain(self):
        orm = StageRunModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            name="build",
            status="ran_to_completion",
            started_at=utc_now(),
            finished_at=utc_now(),
            exit_code=0,
            error_message=None,
        )
        entity = StageRunMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.name == "build"
        assert entity.status == TaskStatus.RAN_TO_COMPLETION
        assert entity.exit_code == 0

    def test_to_orm(self):
        entity = StageRun.create(pipeline_run_id=str(ULID()), name="test")
        orm = StageRunMapper.to_orm(entity)
        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.status == "waiting_to_run"


class TestStageLogMapper:
    def test_to_domain(self):
        orm = StageLogModel(
            id=str(ULID()),
            stage_run_id=str(ULID()),
            content="Log content",
            created_at=utc_now(),
        )
        entity = StageLogMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.stage_run_id == orm.stage_run_id
        assert entity.content == orm.content

    def test_to_orm(self):
        entity = StageLog.create(stage_run_id=str(ULID()), content="Log content")
        orm = StageLogMapper.to_orm(entity)
        assert orm.stage_run_id == entity.stage_run_id
        assert orm.content == entity.content


class TestArtifactMapper:
    def test_to_domain(self):
        orm = ArtifactModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            stage_name="build",
            type="file",
            name="output.txt",
            path="/artifacts/output.txt",
            created_at=utc_now(),
        )
        entity = ArtifactMapper.to_domain(orm)
        assert entity.stage_name == orm.stage_name
        assert entity.type == orm.type

    def test_to_orm(self):
        entity = Artifact.create(
            pipeline_run_id=str(ULID()),
            stage_name="build",
            artifact_type="docker_image",
            name="myapp:latest",
        )
        orm = ArtifactMapper.to_orm(entity)
        assert orm.stage_name == entity.stage_name
        assert orm.type == "docker_image"
