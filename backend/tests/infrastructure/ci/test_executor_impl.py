"""Pipeline 执行器实现测试"""

from typing import TYPE_CHECKING, cast
from unittest.mock import AsyncMock, MagicMock

import pytest

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import StageDefinition
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl

if TYPE_CHECKING:
    from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
    from pomelo_orbit.infrastructure.ci.repositories import StageRunRepositoryImpl


def make_stage(name: str, commands: list[str], depends_on: list[str] | None = None) -> StageDefinition:
    return StageDefinition(
        name=name, id=name, image="alpine:latest", script="\n".join(commands), depends_on=depends_on or []
    )


class TestPipelineExecutorImpl:
    @pytest.fixture
    def stage_run_repo(self):
        return MagicMock()

    @pytest.fixture
    def container_executor(self):
        executor = MagicMock()
        executor.run = AsyncMock(return_value=(0, "success"))
        return executor

    @pytest.fixture
    def executor(self, stage_run_repo, container_executor):
        return PipelineExecutorImpl(
            stage_run_repo=cast("StageRunRepositoryImpl", stage_run_repo),
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
            project_code="proj-code",
            repository_url="https://github.com/test/repo",
            credential_id="01HX0003",
            variables={"key": "value"},
            workspace_path="/tmp/workspace",
            artifacts_path="/tmp/artifacts",
        )

    @pytest.mark.asyncio
    async def test_execute_linear_pipeline(self, executor, context, container_executor):
        stages = [
            make_stage("a", ["echo a"]),
            make_stage("b", ["echo b"], depends_on=["a"]),
            make_stage("c", ["echo c"], depends_on=["b"]),
        ]
        result = await executor.execute(context, stages)
        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_parallel_pipeline(self, executor, context, container_executor):
        stages = [
            make_stage("a", ["echo a"]),
            make_stage("b", ["echo b"]),
            make_stage("c", ["echo c"], depends_on=["a", "b"]),
        ]
        result = await executor.execute(context, stages)
        assert result is True
        assert container_executor.run.call_count == 3

    @pytest.mark.asyncio
    async def test_execute_diamond_dependency(self, executor, context, container_executor):
        stages = [
            make_stage("a", ["echo a"]),
            make_stage("b", ["echo b"], depends_on=["a"]),
            make_stage("c", ["echo c"], depends_on=["a"]),
            make_stage("d", ["echo d"], depends_on=["b", "c"]),
        ]
        result = await executor.execute(context, stages)
        assert result is True
        assert container_executor.run.call_count == 4

    @pytest.mark.asyncio
    async def test_fail_fast_on_stage_failure(self, executor, context, container_executor, stage_run_repo):
        container_executor.run = AsyncMock(side_effect=[(0, "success"), (1, "failed")])
        stages = [
            make_stage("a", ["echo a"]),
            make_stage("b", ["echo b"]),
            make_stage("c", ["echo c"], depends_on=["a", "b"]),
        ]
        result = await executor.execute(context, stages)
        assert result is False
        assert container_executor.run.call_count == 2
        saved = [call[0][0] for call in stage_run_repo.save.call_args_list]
        canceled = [s for s in saved if s.status == TaskStatus.CANCELED]
        assert len(canceled) == 1
        assert canceled[0].stage_name == "c"

    @pytest.mark.asyncio
    async def test_cyclic_dependency_detection(self, executor, context):
        stages = [make_stage("a", ["echo a"], depends_on=["b"]), make_stage("b", ["echo b"], depends_on=["a"])]
        result = await executor.execute(context, stages)
        assert result is False

    @pytest.mark.asyncio
    async def test_stage_exception_handling(self, executor, context, container_executor):
        container_executor.run = AsyncMock(side_effect=RuntimeError("Container error"))
        result = await executor.execute(context, [make_stage("a", ["echo a"])])
        assert result is False

    @pytest.mark.asyncio
    async def test_stage_run_status_transitions(self, context):
        saved_statuses = []

        def capture(sr):
            saved_statuses.append(sr.status)

        stage_run_repo = MagicMock()
        stage_run_repo.save.side_effect = capture
        container_executor = MagicMock()
        container_executor.run = AsyncMock(return_value=(0, "success"))
        executor = PipelineExecutorImpl(
            stage_run_repo=stage_run_repo,
            container_executor=container_executor,
            artifact_repo=MagicMock(),
            credential_repo=MagicMock(),
            security_service=MagicMock(),
        )
        await executor.execute(context, [make_stage("a", ["echo a"])])
        assert len(saved_statuses) == 3
        assert saved_statuses[0] == TaskStatus.WAITING_TO_RUN
        assert saved_statuses[1] == TaskStatus.RUNNING
        assert saved_statuses[2] == TaskStatus.RAN_TO_COMPLETION

    @pytest.mark.asyncio
    async def test_complex_parallel_execution(self, executor, context, container_executor):
        stages = [
            make_stage("checkout", ["git clone"]),
            make_stage("lint", ["lint"], depends_on=["checkout"]),
            make_stage("test", ["test"], depends_on=["checkout"]),
            make_stage("build", ["build"], depends_on=["lint", "test"]),
            make_stage("deploy", ["deploy"], depends_on=["build"]),
        ]
        result = await executor.execute(context, stages)
        assert result is True
        assert container_executor.run.call_count == 5
