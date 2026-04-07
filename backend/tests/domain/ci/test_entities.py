"""测试 CI 实体"""

import time

import ulid

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
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


class TestProject:
    def test_create_project(self):
        project = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
        )
        assert project.id is not None
        assert project.name == "test-project"
        assert project.variable_overrides == {"KEY": "value"}
        assert project.created_at is not None
        assert project.updated_at is not None

    def test_update_project(self):
        project = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
        )
        original_updated_at = project.updated_at
        time.sleep(0.001)
        project.update(name="new-name", variable_overrides={"NEW_KEY": "new_value"})
        assert project.name == "new-name"
        assert project.variable_overrides == {"NEW_KEY": "new_value"}
        assert project.updated_at >= original_updated_at


class TestCredential:
    def test_create_credential(self):
        credential = Credential.create(
            name="test-credential",
            type=CredentialType.GIT_SSH,
            encrypted_data="encrypted_data_here",
        )
        assert credential.id is not None
        assert credential.name == "test-credential"
        assert credential.type == CredentialType.GIT_SSH
        assert credential.encrypted_data == "encrypted_data_here"
        assert credential.created_at is not None


class TestPipelineTemplate:
    def test_create_template(self):
        var_decl = VariableDeclaration(name="IMAGE_NAME", required=True)
        template = PipelineTemplate.create(
            name="test-template",
            variable_declarations=[var_decl],
            description="Test template",
        )
        assert template.id is not None
        assert template.name == "test-template"
        assert len(template.variable_declarations) == 1

    def test_update_template(self):
        template = PipelineTemplate.create(name="test-template", variable_declarations=[])
        original_updated_at = template.updated_at
        time.sleep(0.001)
        template.update(name="new-template", description="Updated")
        assert template.name == "new-template"
        assert template.description == "Updated"
        assert template.updated_at >= original_updated_at


class TestPipelineRun:
    def test_create_pipeline_run(self):
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={"KEY": "value"},
        )
        assert run.id is not None
        assert run.status == TaskStatus.WAITING_TO_RUN
        assert run.started_at is None

    def test_pipeline_run_lifecycle(self):
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run.start()
        assert run.status == TaskStatus.RUNNING
        assert run.started_at is not None
        run.complete_success()
        assert run.status == TaskStatus.RAN_TO_COMPLETION  # type: ignore[comparison-overlap]
        assert run.finished_at is not None  # type: ignore[unreachable]

    def test_pipeline_run_failure(self):
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run.start()
        run.complete_failed()
        assert run.status == TaskStatus.FAULTED
        assert run.finished_at is not None

    def test_create_pipeline_run_with_retry_of(self):
        original_run_id = str(ulid.ULID())
        retry_run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={"KEY": "value"},
            retry_of=original_run_id,
        )
        assert retry_run.id is not None
        assert retry_run.retry_of == original_run_id
        assert retry_run.status == TaskStatus.WAITING_TO_RUN


class TestStageRun:
    def test_create_stage_run(self):
        sr = StageRun.create(pipeline_run_id=str(ulid.ULID()), name="build")
        assert sr.id is not None
        assert sr.name == "build"
        assert sr.status == TaskStatus.WAITING_TO_RUN

    def test_stage_run_lifecycle_success(self):
        sr = StageRun.create(pipeline_run_id=str(ulid.ULID()), name="build")
        sr.start()
        assert sr.status == TaskStatus.RUNNING
        assert sr.started_at is not None
        sr.complete_success(exit_code=0)
        assert sr.status == TaskStatus.RAN_TO_COMPLETION  # type: ignore[comparison-overlap]
        assert sr.exit_code == 0  # type: ignore[unreachable]
        assert sr.finished_at is not None

    def test_stage_run_lifecycle_failed(self):
        sr = StageRun.create(pipeline_run_id=str(ulid.ULID()), name="build")
        sr.start()
        sr.complete_failed(exit_code=1, error_message="Command failed")
        assert sr.status == TaskStatus.FAULTED
        assert sr.exit_code == 1
        assert sr.error_message == "Command failed"
        assert sr.finished_at is not None

    def test_stage_run_lifecycle_faulted(self):
        sr = StageRun.create(pipeline_run_id=str(ulid.ULID()), name="build")
        sr.start()
        sr.complete_faulted(error_message="Container timeout")
        assert sr.status == TaskStatus.FAULTED
        assert sr.error_message == "Container timeout"
        assert sr.finished_at is not None


class TestStageLog:
    def test_create_stage_log(self):
        log = StageLog.create(stage_run_id=str(ulid.ULID()), content="Log content here")
        assert log.id is not None
        assert log.content == "Log content here"
        assert log.created_at is not None
