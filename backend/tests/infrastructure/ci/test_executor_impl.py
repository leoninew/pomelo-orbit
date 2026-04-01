"""Pipeline 执行器实现测试"""

from typing import TYPE_CHECKING, cast
from unittest.mock import AsyncMock, MagicMock

import pytest

from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import JobStatus, PipelineDefinition, StepDefinition
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl

if TYPE_CHECKING:
    from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
    from pomelo_orbit.infrastructure.ci.repositories import JobLogRepositoryImpl, JobRepositoryImpl


class TestPipelineExecutorImpl:
    """Pipeline 执行器实现测试"""

    @pytest.fixture
    def job_repo(self):
        """Job 仓库 mock"""
        return MagicMock()

    @pytest.fixture
    def job_log_repo(self):
        """Job 日志仓库 mock"""
        return MagicMock()

    @pytest.fixture
    def container_executor(self):
        """容器执行器 mock"""
        executor = MagicMock()
        executor.run = AsyncMock(return_value=(0, "success"))
        return executor

    @pytest.fixture
    def executor(self, job_repo, job_log_repo, container_executor):
        """执行器实例"""
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
        """执行上下文"""
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
        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
                StepDefinition(name="b", image="alpine", commands=["echo b"], depends_on=["a"]),
                StepDefinition(name="c", image="alpine", commands=["echo c"], depends_on=["b"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_parallel_pipeline(self, executor, context, container_executor):
        """测试执行并行 pipeline"""
        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
                StepDefinition(name="b", image="alpine", commands=["echo b"]),
                StepDefinition(name="c", image="alpine", commands=["echo c"], depends_on=["a", "b"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_diamond_dependency(self, executor, context, container_executor):
        """测试执行菱形依赖"""
        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
                StepDefinition(name="b", image="alpine", commands=["echo b"], depends_on=["a"]),
                StepDefinition(name="c", image="alpine", commands=["echo c"], depends_on=["a"]),
                StepDefinition(name="d", image="alpine", commands=["echo d"], depends_on=["b", "c"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 4

    @pytest.mark.asyncio
    async def test_fail_fast_on_job_failure(self, executor, context, container_executor, job_repo):
        """测试 Job 失败时的 fail-fast"""
        # 第二个 Job 失败
        container_executor.run = AsyncMock(
            side_effect=[
                (0, "success"),  # a
                (1, "failed"),  # b
            ]
        )

        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
                StepDefinition(name="b", image="alpine", commands=["echo b"]),
                StepDefinition(name="c", image="alpine", commands=["echo c"], depends_on=["a", "b"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is False
        # 只执行了 a 和 b，c 被取消
        assert container_executor.run.call_count == 2

        # 验证 c 被标记为 CANCELED
        saved_jobs = [call[0][0] for call in job_repo.save.call_args_list]
        canceled_jobs = [job for job in saved_jobs if job.status == JobStatus.CANCELED]
        assert len(canceled_jobs) == 1
        assert canceled_jobs[0].name == "c"

    @pytest.mark.asyncio
    async def test_cyclic_dependency_detection(self, executor, context):
        """测试循环依赖检测"""
        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"], depends_on=["b"]),
                StepDefinition(name="b", image="alpine", commands=["echo b"], depends_on=["a"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is False

    @pytest.mark.asyncio
    async def test_job_exception_handling(self, executor, context, container_executor):
        """测试 Job 执行异常处理"""
        container_executor.run = AsyncMock(side_effect=RuntimeError("Container error"))

        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is False

    @pytest.mark.asyncio
    async def test_job_log_recording(self, executor, context, container_executor, job_log_repo):
        """测试 Job 日志记录"""
        container_executor.run = AsyncMock(return_value=(0, "test output"))

        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
            ],
        )

        await executor.execute(context, definition)

        # 验证日志被保存
        assert job_log_repo.save.called
        saved_log = job_log_repo.save.call_args[0][0]
        assert saved_log.content == "test output"

    @pytest.mark.asyncio
    async def test_job_status_transitions(self, job_log_repo, context):
        """测试 Job 状态转换"""
        # 创建一个列表来捕获每次 save 时的状态
        saved_statuses = []

        def capture_status(job):
            saved_statuses.append(job.status)

        # 创建新的 job_repo mock
        job_repo = MagicMock()
        job_repo.save.side_effect = capture_status

        # 创建新的 container_executor 和 executor 实例
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

        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="a", image="alpine", commands=["echo a"]),
            ],
        )

        await executor.execute(context, definition)

        # 验证状态转换：创建(WAITING) -> 开始(RUNNING) -> 完成(SUCCESS)
        assert len(saved_statuses) == 3
        assert saved_statuses[0] == JobStatus.WAITING
        assert saved_statuses[1] == JobStatus.RUNNING
        assert saved_statuses[2] == JobStatus.SUCCESS

    @pytest.mark.asyncio
    async def test_complex_parallel_execution(self, executor, context, container_executor):
        """测试复杂并行执行场景"""
        definition = PipelineDefinition(
            version="1.0",
            steps=[
                StepDefinition(name="checkout", image="alpine", commands=["git clone"]),
                StepDefinition(name="lint", image="alpine", commands=["lint"], depends_on=["checkout"]),
                StepDefinition(name="test", image="alpine", commands=["test"], depends_on=["checkout"]),
                StepDefinition(name="build", image="alpine", commands=["build"], depends_on=["lint", "test"]),
                StepDefinition(name="deploy", image="alpine", commands=["deploy"], depends_on=["build"]),
            ],
        )

        result = await executor.execute(context, definition)

        assert result is True
        assert container_executor.run.call_count == 5
