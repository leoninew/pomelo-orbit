"""Pipeline 执行器实现测试"""

from typing import TYPE_CHECKING, cast
from unittest.mock import AsyncMock, MagicMock

import pytest

from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import JobStatus, StageDefinition, StageType, StepDefinition
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl

if TYPE_CHECKING:
    from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
    from pomelo_orbit.infrastructure.ci.repositories import JobLogRepositoryImpl, JobRepositoryImpl


def custom_stage(name: str, commands: list[str], depends_on: list[str] | None = None) -> StageDefinition:
    """创建 custom stage 的辅助函数"""
    return StageDefinition(
        name=name,
        type=StageType.CUSTOM,
        depends_on=depends_on or [],
        steps=[StepDefinition(name=name, image="alpine", commands=commands)],
    )


class TestPipelineExecutorImpl:
    """Pipeline 执行器实现测试"""

    @pytest.fixture
    def job_repo(self):
        return MagicMock()

    @pytest.fixture
    def job_log_repo(self):
        return MagicMock()

    @pytest.fixture
    def container_executor(self):
        executor = MagicMock()
        executor.run = AsyncMock(return_value=(0, "success"))
        return executor

    @pytest.fixture
    def executor(self, job_repo, job_log_repo, container_executor):
        return PipelineExecutorImpl(
            job_repo=cast("JobRepositoryImpl", job_repo),
            job_log_repo=cast("JobLogRepositoryImpl", job_log_repo),
            container_executor=cast("ContainerExecutor", container_executor),
            artifact_repo=MagicMock(),
            credential_repo=MagicMock(),
            security_service=MagicMock(),
        )

    @pytest.fixture
    def context(self):
        return ExecutionContext(
            run_id="01HX0001",
            project_id="01HX0002",
            repository_url="https://github.com/test/repo",
            credential_id="01HX0003",
            variables={"key": "value"},
            workspace_path="/tmp/workspace",
            artifacts_path="/tmp/artifacts",
        )

    @pytest.mark.asyncio
    async def test_execute_linear_pipeline(self, executor, context, container_executor):
        """测试执行线性 pipeline"""
        stages = [
            custom_stage("a", ["echo a"]),
            custom_stage("b", ["echo b"], depends_on=["a"]),
            custom_stage("c", ["echo c"], depends_on=["b"]),
        ]

        result = await executor.execute(context, stages)

        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_parallel_pipeline(self, executor, context, container_executor):
        """测试执行并行 pipeline"""
        stages = [
            custom_stage("a", ["echo a"]),
            custom_stage("b", ["echo b"]),
            custom_stage("c", ["echo c"], depends_on=["a", "b"]),
        ]

        result = await executor.execute(context, stages)

        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_diamond_dependency(self, executor, context, container_executor):
        """测试执行菱形依赖"""
        stages = [
            custom_stage("a", ["echo a"]),
            custom_stage("b", ["echo b"], depends_on=["a"]),
            custom_stage("c", ["echo c"], depends_on=["a"]),
            custom_stage("d", ["echo d"], depends_on=["b", "c"]),
        ]

        result = await executor.execute(context, stages)

        assert result is True
        assert container_executor.run.call_count == 4

    @pytest.mark.asyncio
    async def test_fail_fast_on_stage_failure(self, executor, context, container_executor, job_repo):
        """测试 Stage 失败时的 fail-fast"""
        container_executor.run = AsyncMock(
            side_effect=[
                (0, "success"),  # a
                (1, "failed"),  # b
            ]
        )

        stages = [
            custom_stage("a", ["echo a"]),
            custom_stage("b", ["echo b"]),
            custom_stage("c", ["echo c"], depends_on=["a", "b"]),
        ]

        result = await executor.execute(context, stages)

        assert result is False
        assert container_executor.run.call_count == 2

        saved_jobs = [call[0][0] for call in job_repo.save.call_args_list]
        canceled_jobs = [job for job in saved_jobs if job.status == JobStatus.CANCELED]
        assert len(canceled_jobs) == 1
        assert canceled_jobs[0].name == "c"

    @pytest.mark.asyncio
    async def test_cyclic_dependency_detection(self, executor, context):
        """测试循环依赖检测"""
        stages = [
            custom_stage("a", ["echo a"], depends_on=["b"]),
            custom_stage("b", ["echo b"], depends_on=["a"]),
        ]

        result = await executor.execute(context, stages)

        assert result is False

    @pytest.mark.asyncio
    async def test_job_exception_handling(self, executor, context, container_executor):
        """测试 Job 执行异常处理"""
        container_executor.run = AsyncMock(side_effect=RuntimeError("Container error"))

        stages = [custom_stage("a", ["echo a"])]

        result = await executor.execute(context, stages)

        assert result is False

    @pytest.mark.asyncio
    async def test_job_log_recording(self, executor, context, container_executor, job_log_repo):
        """测试 Job 日志记录"""
        container_executor.run = AsyncMock(return_value=(0, "test output"))

        stages = [custom_stage("a", ["echo a"])]

        await executor.execute(context, stages)

        assert job_log_repo.save.called
        saved_log = job_log_repo.save.call_args[0][0]
        assert saved_log.content == "test output"

    @pytest.mark.asyncio
    async def test_job_status_transitions(self, job_log_repo, context):
        """测试 Job 状态转换"""
        saved_statuses = []

        def capture_status(job):
            saved_statuses.append(job.status)

        job_repo = MagicMock()
        job_repo.save.side_effect = capture_status

        container_executor = MagicMock()
        container_executor.run = AsyncMock(return_value=(0, "success"))
        executor = PipelineExecutorImpl(
            job_repo=job_repo,
            job_log_repo=job_log_repo,
            container_executor=container_executor,
            artifact_repo=MagicMock(),
            credential_repo=MagicMock(),
            security_service=MagicMock(),
        )

        stages = [custom_stage("a", ["echo a"])]

        await executor.execute(context, stages)

        assert len(saved_statuses) == 3
        assert saved_statuses[0] == JobStatus.WAITING
        assert saved_statuses[1] == JobStatus.RUNNING
        assert saved_statuses[2] == JobStatus.SUCCESS

    @pytest.mark.asyncio
    async def test_complex_parallel_execution(self, executor, context, container_executor):
        """测试复杂并行执行场景"""
        stages = [
            custom_stage("checkout", ["git clone"]),
            custom_stage("lint", ["lint"], depends_on=["checkout"]),
            custom_stage("test", ["test"], depends_on=["checkout"]),
            custom_stage("build", ["build"], depends_on=["lint", "test"]),
            custom_stage("deploy", ["deploy"], depends_on=["build"]),
        ]

        result = await executor.execute(context, stages)

        assert result is True
        assert container_executor.run.call_count == 5
