"""CI Executor 单元测试"""

from typing import TYPE_CHECKING, cast
from unittest.mock import AsyncMock, Mock

import pytest

from pomelo_orbit.domain.ci.entities import Job
from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import (
    JobStatus,
    RetryPolicy,
    StageDefinition,
    StageType,
    StepDefinition,
)
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl

if TYPE_CHECKING:
    from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
    from pomelo_orbit.infrastructure.ci.repositories import JobLogRepositoryImpl, JobRepositoryImpl


def create_test_context(
    run_id: str = "run-1", retry_of: str | None = None, variables: dict | None = None
) -> ExecutionContext:
    """创建测试用的 ExecutionContext"""
    return ExecutionContext(
        run_id=run_id,
        project_id="project-1",
        repository_url="https://github.com/user/repo.git",
        credential_id="cred-1",
        workspace_path="/workspace",
        artifacts_path="/artifacts",
        variables=variables or {},
        retry_of=retry_of,
    )


def create_test_definition(steps: list[StepDefinition]) -> list[StageDefinition]:
    """创建测试用的 StageDefinition 列表"""
    result = []
    for s in steps:
        depends_on = s.depends_on or []
        # depends_on 应该在 StageDefinition 层面
        stage = StageDefinition(name=s.name, type=StageType.CUSTOM, depends_on=depends_on, steps=[s])
        result.append(stage)
    return result


class TestPipelineExecutorImpl:
    """测试 PipelineExecutorImpl"""

    @pytest.mark.asyncio
    async def test_execute_simple_pipeline_success(self):
        """测试执行简单 pipeline 成功"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()
        artifact_repo = Mock()

        container_executor.run.return_value = (0, "Success output")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=artifact_repo,
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="build", image="alpine:latest", commands=["echo 'Building'"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert job_repo.save.call_count >= 2
        assert job_log_repo.save.call_count == 1
        container_executor.run.assert_called_once()

    @pytest.mark.asyncio
    async def test_execute_pipeline_with_failure(self):
        """测试执行 pipeline 失败"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.return_value = (1, "Error output")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="build", image="alpine:latest", commands=["exit 1"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is False
        assert job_repo.save.call_count >= 2

    @pytest.mark.asyncio
    async def test_execute_parallel_jobs(self):
        """测试并行执行多个 jobs"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.return_value = (0, "Success")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="job1", image="alpine:latest", commands=["echo 1"]),
                StepDefinition(name="job2", image="alpine:latest", commands=["echo 2"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 2

    @pytest.mark.asyncio
    async def test_execute_with_dependencies(self):
        """测试执行有依赖关系的 jobs"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.return_value = (0, "Success")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="build", image="alpine:latest", commands=["echo build"]),
                StepDefinition(name="test", image="alpine:latest", commands=["echo test"], depends_on=["build"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 2

    @pytest.mark.asyncio
    async def test_execute_fail_fast(self):
        """测试 fail-fast 机制"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.side_effect = [(1, "Failed")]

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="build", image="alpine:latest", commands=["exit 1"]),
                StepDefinition(name="test", image="alpine:latest", commands=["echo test"], depends_on=["build"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is False
        assert container_executor.run.call_count == 1
        saved_jobs = [call[0][0] for call in job_repo.save.call_args_list]
        canceled_jobs = [job for job in saved_jobs if hasattr(job, "status") and job.status == JobStatus.CANCELED]
        assert len(canceled_jobs) >= 1

    @pytest.mark.asyncio
    async def test_execute_with_cyclic_dependency(self):
        """测试检测循环依赖"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(name="jobA", image="alpine:latest", commands=["echo A"], depends_on=["jobB"]),
                StepDefinition(name="jobB", image="alpine:latest", commands=["echo B"], depends_on=["jobA"]),
            ]
        )

        result = await executor.execute(context, definition)

        assert result is False
        container_executor.run.assert_not_called()

    @pytest.mark.asyncio
    async def test_execute_with_exception(self):
        """测试执行过程中抛出异常"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.side_effect = Exception("Container error")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [StepDefinition(name="build", image="alpine:latest", commands=["echo build"])]
        )

        result = await executor.execute(context, definition)

        assert result is False
        saved_jobs = [call[0][0] for call in job_repo.save.call_args_list]
        faulted_jobs = [job for job in saved_jobs if hasattr(job, "status") and job.status == JobStatus.FAULTED]
        assert len(faulted_jobs) >= 1

    @pytest.mark.asyncio
    async def test_save_artifacts(self):
        """测试保存制品"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()
        artifact_repo = Mock()

        container_executor.run.return_value = (0, "Success")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=artifact_repo,
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context()
        definition = create_test_definition(
            [
                StepDefinition(
                    name="build",
                    image="alpine:latest",
                    commands=["echo build"],
                    artifacts=[
                        {"type": "file", "name": "output.txt", "path": "output.txt"},
                        {"type": "docker_image", "name": "myapp:latest"},
                    ],
                )
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert artifact_repo.save.call_count == 2

    @pytest.mark.asyncio
    async def test_retry_skip_if_success(self):
        """测试重试时跳过成功的 job"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        original_job = Mock(spec=Job)
        original_job.name = "build"
        original_job.status = JobStatus.SUCCESS
        job_repo.find_by_run.side_effect = [[original_job], []]

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context(run_id="run-2", retry_of="run-1")
        definition = create_test_definition(
            [
                StepDefinition(
                    name="build",
                    image="alpine:latest",
                    commands=["echo build"],
                    retry_policy=RetryPolicy.SKIP_IF_SUCCESS,
                )
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        container_executor.run.assert_not_called()
        saved_jobs = [call[0][0] for call in job_repo.save.call_args_list]
        skipped_jobs = [job for job in saved_jobs if hasattr(job, "status") and job.status == JobStatus.SKIPPED]
        assert len(skipped_jobs) >= 1

    @pytest.mark.asyncio
    async def test_retry_rerun_failed_job(self):
        """测试重试时重新运行失败的 job"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        original_job = Mock(spec=Job)
        original_job.name = "build"
        original_job.status = JobStatus.FAILED
        job_repo.find_by_run.side_effect = [[original_job], []]

        container_executor.run.return_value = (0, "Success")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context(run_id="run-2", retry_of="run-1")
        definition = create_test_definition(
            [
                StepDefinition(
                    name="build",
                    image="alpine:latest",
                    commands=["echo build"],
                    retry_policy=RetryPolicy.SKIP_IF_SUCCESS,
                )
            ]
        )

        result = await executor.execute(context, definition)

        assert result is True
        container_executor.run.assert_called_once()

    @pytest.mark.asyncio
    async def test_environment_variables_passed_to_container(self):
        """测试环境变量传递给容器"""
        job_repo = Mock()
        job_log_repo = Mock()
        container_executor = AsyncMock()

        container_executor.run.return_value = (0, "Success")

        executor = PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=Mock(),
            credential_repo=Mock(),
            security_service=Mock(),
        )

        context = create_test_context(variables={"ENV_VAR": "value123", "NUMBER": 42})
        definition = create_test_definition(
            [StepDefinition(name="build", image="alpine:latest", commands=["echo $ENV_VAR"])]
        )

        result = await executor.execute(context, definition)

        assert result is True
        call_args = container_executor.run.call_args
        assert call_args[1]["environment"] == {"ENV_VAR": "value123", "NUMBER": "42"}
