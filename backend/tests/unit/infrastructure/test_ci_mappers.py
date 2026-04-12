"""CI Mappers 单元测试"""

from ulid import ULID

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    PipelineRun,
    PipelineTemplate,
    Repository,
    StageRun,
)
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactType,
    CredentialType,
    PipelineRunTrigger,
    VariableDeclaration,
    VariableSource,
)
from pomelo_orbit.infrastructure.ci.mappers import (
    ArtifactMapper,
    CredentialMapper,
    PipelineRunMapper,
    PipelineTemplateMapper,
    RepositoryMapper,
    StageRunMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
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
            variable_declarations=[VariableDeclaration(name="VAR1", value="value1")],
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
            variable_overrides='[{"name": "VAR1", "value": "override1"}]',
            default_branch="main",
            created_at=utc_now(),
            updated_at=utc_now(),
        )
        entity = RepositoryMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.code == orm.code
        assert len(entity.variable_overrides) == 1
        assert entity.variable_overrides[0].name == "VAR1"
        assert entity.variable_overrides[0].value == "override1"

    def test_to_orm(self):
        entity = Repository.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo",
            git_credential_id=str(ULID()),
            variable_overrides=[VariableDeclaration(name="VAR1", value="override1")],
            default_branch="main",
        )
        orm = RepositoryMapper.to_orm(entity)
        assert orm.code == entity.code
        assert '"name": "VAR1"' in orm.variable_overrides
        assert '"value": "override1"' in orm.variable_overrides


class TestPipelineRunMapper:
    def test_to_domain(self):
        orm = PipelineRunModel(
            id=str(ULID()),
            repository_id=str(ULID()),
            repository_name="test-project",
            snapshot_id=str(ULID()),
            template_id=str(ULID()),
            template_name="test-template",
            template_version=1,
            trigger="manual",
            trigger_ref="main",
            variables_snapshot='[{"name": "VAR1", "description": "", "value": "value1", "secret": false, "source": "template_custom"}]',
            status="running",
            retry_of=None,
            started_at=utc_now(),
            finished_at=None,
            created_at=utc_now(),
        )
        entity = PipelineRunMapper.to_domain(orm)
        assert entity.trigger == PipelineRunTrigger.MANUAL
        assert entity.status == TaskStatus.RUNNING
        assert len(entity.variables_snapshot) == 1
        assert entity.variables_snapshot[0].name == "VAR1"
        assert entity.variables_snapshot[0].value == "value1"

    def test_to_orm(self):
        entity = PipelineRun.create(
            repository_id=str(ULID()),
            repository_name="test-project",
            snapshot_id=str(ULID()),
            template_id=str(ULID()),
            template_name="test-template",
            template_version=1,
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="feature/test",
            variables_snapshot=[
                VariableDeclaration(name="VAR1", value="value1", source=VariableSource.TEMPLATE_CUSTOM)
            ],
        )
        orm = PipelineRunMapper.to_orm(entity)
        assert orm.trigger == "webhook"
        assert '"name": "VAR1"' in orm.variables_snapshot
        assert '"value": "value1"' in orm.variables_snapshot


class TestStageRunMapper:
    def test_to_domain(self):
        orm = StageRunModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            stage_id=str(ULID()),
            stage_name="build",
            status="ran_to_completion",
            started_at=utc_now(),
            finished_at=utc_now(),
            exit_code=0,
            error_message=None,
        )
        entity = StageRunMapper.to_domain(orm)
        assert entity.id == orm.id
        assert entity.stage_name == "build"
        assert entity.status == TaskStatus.RAN_TO_COMPLETION
        assert entity.exit_code == 0

    def test_to_orm(self):
        entity = StageRun.create(pipeline_run_id=str(ULID()), stage_id=str(ULID()), stage_name="test")
        orm = StageRunMapper.to_orm(entity)
        assert orm.id == entity.id
        assert orm.stage_name == entity.stage_name
        assert orm.status == "waiting_to_run"


class TestArtifactMapper:
    def test_to_domain(self):
        orm = ArtifactModel(
            id=str(ULID()),
            pipeline_run_id=str(ULID()),
            stage_name="build",
            type="docker_image",
            name="output.txt",
            path="/artifacts/output.txt",
            created_at=utc_now(),
        )
        entity = ArtifactMapper.to_domain(orm)
        assert entity.stage_name == orm.stage_name
        assert entity.type == ArtifactType.DOCKER_IMAGE

    def test_to_orm(self):
        entity = Artifact.create(
            pipeline_run_id=str(ULID()),
            stage_name="build",
            artifact_type=ArtifactType.DOCKER_IMAGE,
            name="myapp:latest",
        )
        orm = ArtifactMapper.to_orm(entity)
        assert orm.stage_name == entity.stage_name
        assert orm.type == "docker_image"
