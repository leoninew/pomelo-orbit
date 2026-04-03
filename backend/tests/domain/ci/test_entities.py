"""测试 CI 实体"""

import ulid

from pomelo_orbit.domain.ci.entities import (
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


class TestProject:
    """测试 Project 实体"""

    def test_create_project(self):
        """测试创建项目"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_snapshot_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
        )

        assert project.id is not None
        assert project.name == "test-project"
        assert project.variable_overrides == {"KEY": "value"}
        assert project.created_at is not None
        assert project.updated_at is not None

    def test_update_project(self):
        """测试更新项目"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_snapshot_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
        )

        original_updated_at = project.updated_at

        # 添加微小延迟确保时间戳不同
        import time

        time.sleep(0.001)

        project.update(name="new-name", variable_overrides={"NEW_KEY": "new_value"})

        assert project.name == "new-name"
        assert project.variable_overrides == {"NEW_KEY": "new_value"}
        assert project.updated_at >= original_updated_at


class TestCredential:
    """测试 Credential 实体"""

    def test_create_credential(self):
        """测试创建凭据"""
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
    """测试 PipelineTemplate 实体"""

    def test_create_template(self):
        """测试创建模板"""
        var_decl = VariableDeclaration(name="IMAGE_NAME", required=True)
        template = PipelineTemplate.create(
            name="test-template",
            stages=[],
            variable_declarations=[var_decl],
            description="Test template",
        )

        assert template.id is not None
        assert template.name == "test-template"
        assert len(template.variable_declarations) == 1
        assert template.is_builtin is False

    def test_update_template(self):
        """测试更新模板"""
        template = PipelineTemplate.create(
            name="test-template",
            stages=[],
            variable_declarations=[],
        )

        original_updated_at = template.updated_at

        # 添加微小延迟确保时间戳不同
        import time

        time.sleep(0.001)

        template.update(name="new-template", description="Updated")

        assert template.name == "new-template"
        assert template.description == "Updated"
        assert template.updated_at >= original_updated_at


class TestPipelineRun:
    """测试 PipelineRun 实体"""

    def test_create_pipeline_run(self):
        """测试创建 pipeline run"""
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={"KEY": "value"},
        )

        assert run.id is not None
        assert run.status.value == PipelineRunStatus.WAITING.value
        assert run.started_at is None

    def test_pipeline_run_lifecycle(self):
        """测试 pipeline run 生命周期"""
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )

        # 开始执行
        run.start()
        assert run.status.value == PipelineRunStatus.RUNNING.value
        assert run.started_at is not None

        # 执行成功
        run.complete_success()
        assert run.status.value == PipelineRunStatus.SUCCESS.value
        assert run.finished_at is not None

    def test_pipeline_run_failure(self):
        """测试 pipeline run 失败"""
        run = PipelineRun.create(
            project_id=str(ulid.ULID()),
            pipeline_snapshot_id=str(ulid.ULID()),
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )

        run.start()
        run.complete_failed()

        assert run.status.value == PipelineRunStatus.FAILED.value
        assert run.finished_at is not None

    def test_create_pipeline_run_with_retry_of(self):
        """测试创建重试的 pipeline run"""
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
        assert retry_run.status.value == PipelineRunStatus.WAITING.value


class TestJob:
    """测试 Job 实体"""

    def test_create_job(self):
        """测试创建 job"""
        job = Job.create(
            pipeline_run_id=str(ulid.ULID()),
            name="test-job",
        )

        assert job.id is not None
        assert job.name == "test-job"
        assert job.status.value == JobStatus.WAITING.value
        assert job.parent_job_id is None

    def test_job_lifecycle_success(self):
        """测试 job 成功生命周期"""
        job = Job.create(
            pipeline_run_id=str(ulid.ULID()),
            name="test-job",
        )

        job.start()
        assert job.status.value == JobStatus.RUNNING.value
        assert job.started_at is not None

        job.complete_success(exit_code=0)
        assert job.status.value == JobStatus.SUCCESS.value
        assert job.exit_code == 0
        assert job.finished_at is not None

    def test_job_lifecycle_failed(self):
        """测试 job 失败生命周期"""
        job = Job.create(
            pipeline_run_id=str(ulid.ULID()),
            name="test-job",
        )

        job.start()
        job.complete_failed(exit_code=1, error_message="Command failed")

        assert job.status.value == JobStatus.FAILED.value
        assert job.exit_code == 1
        assert job.error_message == "Command failed"
        assert job.finished_at is not None

    def test_job_lifecycle_faulted(self):
        """测试 job 故障生命周期"""
        job = Job.create(
            pipeline_run_id=str(ulid.ULID()),
            name="test-job",
        )

        job.start()
        job.complete_faulted(error_message="Container timeout")

        assert job.status.value == JobStatus.FAULTED.value
        assert job.error_message == "Container timeout"
        assert job.finished_at is not None


class TestJobLog:
    """测试 JobLog 实体"""

    def test_create_job_log(self):
        """测试创建 job 日志"""
        log = JobLog.create(
            job_id=str(ulid.ULID()),
            content="Log content here",
        )

        assert log.id is not None
        assert log.content == "Log content here"
        assert log.created_at is not None
