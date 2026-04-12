"""CI Executor 单元测试"""

from typing import TYPE_CHECKING, cast
from unittest.mock import AsyncMock, Mock

import pytest

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import ArtifactConfig, StageDefinition
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl

if TYPE_CHECKING:
    from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
    from pomelo_orbit.infrastructure.ci.repositories import StageRunRepositoryImpl


def make_executor(container_executor=None, stage_run_repo=None, artifact_repo=None):
    return PipelineExecutorImpl(
        stage_run_repo=cast("StageRunRepositoryImpl", stage_run_repo or Mock()),
        container_executor=cast("ContainerExecutor", container_executor or AsyncMock()),
        artifact_repo=artifact_repo or Mock(),
        credential_repo=Mock(),
        security_service=Mock(),
    )


def make_context(run_id: str = "run-1", retry_of: str | None = None, variables: dict | None = None) -> ExecutionContext:
    return ExecutionContext(
        run_id=run_id,
        repository_id="project-1",
        project_code="project-1",
        repository_url="https://github.com/user/repo.git",
        credential_id="cred-1",
        workspace_path="/workspace",
        artifacts_path="/artifacts",
        variables=variables or {},
        retry_of=retry_of,
    )


def stage(
    name: str, image: str = "alpine:latest", commands: list[str] | None = None, depends_on: list[str] | None = None
) -> StageDefinition:
    return StageDefinition(
        name=name,
        id=name,
        image=image,
        script="\n".join(commands or ["echo ok"]),
        depends_on=depends_on or [],
        version=1,
    )


class TestPipelineExecutorImpl:
    @pytest.mark.asyncio
    async def test_execute_simple_pipeline_success(self):
        stage_run_repo = Mock()
        container_executor = AsyncMock()
        container_executor.run.return_value = (0, "Success output")
        executor = make_executor(container_executor=container_executor, stage_run_repo=stage_run_repo)
        result = await executor.execute(make_context(), [stage("build")])
        assert result is True
        assert stage_run_repo.save.call_count >= 2
        container_executor.run.assert_called_once()

    @pytest.mark.asyncio
    async def test_execute_pipeline_with_failure(self):
        stage_run_repo = Mock()
        container_executor = AsyncMock()
        container_executor.run.return_value = (1, "Error output")
        executor = make_executor(container_executor=container_executor, stage_run_repo=stage_run_repo)
        result = await executor.execute(make_context(), [stage("build")])
        assert result is False
        assert stage_run_repo.save.call_count >= 2

    @pytest.mark.asyncio
    async def test_execute_parallel_jobs(self):
        container_executor = AsyncMock()
        container_executor.run.return_value = (0, "Success")
        executor = make_executor(container_executor=container_executor)
        result = await executor.execute(make_context(), [stage("job1"), stage("job2")])
        assert result is True
        assert container_executor.run.call_count == 2

    @pytest.mark.asyncio
    async def test_execute_with_dependencies(self):
        container_executor = AsyncMock()
        container_executor.run.return_value = (0, "Success")
        executor = make_executor(container_executor=container_executor)
        result = await executor.execute(make_context(), [stage("build"), stage("test", depends_on=["build"])])
        assert result is True
        assert container_executor.run.call_count == 2

    @pytest.mark.asyncio
    async def test_execute_fail_fast(self):
        stage_run_repo = Mock()
        container_executor = AsyncMock()
        container_executor.run.side_effect = [(1, "Failed")]
        executor = make_executor(container_executor=container_executor, stage_run_repo=stage_run_repo)
        result = await executor.execute(make_context(), [stage("build"), stage("test", depends_on=["build"])])
        assert result is False
        assert container_executor.run.call_count == 1
        saved = [call[0][0] for call in stage_run_repo.save.call_args_list]
        canceled = [s for s in saved if hasattr(s, "status") and s.status == TaskStatus.CANCELED]
        assert len(canceled) >= 1

    @pytest.mark.asyncio
    async def test_execute_with_cyclic_dependency(self):
        container_executor = AsyncMock()
        executor = make_executor(container_executor=container_executor)
        stages = [stage("jobA", depends_on=["jobB"]), stage("jobB", depends_on=["jobA"])]
        result = await executor.execute(make_context(), stages)
        assert result is False
        container_executor.run.assert_not_called()

    @pytest.mark.asyncio
    async def test_execute_with_exception(self):
        stage_run_repo = Mock()
        container_executor = AsyncMock()
        container_executor.run.side_effect = Exception("Container error")
        executor = make_executor(container_executor=container_executor, stage_run_repo=stage_run_repo)
        result = await executor.execute(make_context(), [stage("build")])
        assert result is False
        saved = [call[0][0] for call in stage_run_repo.save.call_args_list]
        faulted = [s for s in saved if hasattr(s, "status") and s.status == TaskStatus.FAULTED]
        assert len(faulted) >= 1

    @pytest.mark.asyncio
    async def test_save_artifacts(self):
        artifact_repo = Mock()
        container_executor = AsyncMock()
        container_executor.run.return_value = (0, "Success")
        s = StageDefinition(
            name="build",
            id="build",
            image="alpine:latest",
            script="echo build",
            artifacts=[ArtifactConfig(name="output.txt", path="output.txt")],
            version=1,
        )
        executor = make_executor(container_executor=container_executor, artifact_repo=artifact_repo)
        result = await executor.execute(make_context(), [s])
        assert result is True
        assert artifact_repo.save.call_count == 1

    @pytest.mark.asyncio
    async def test_environment_variables_passed_to_container(self):
        container_executor = AsyncMock()
        container_executor.run.return_value = (0, "Success")
        executor = make_executor(container_executor=container_executor)
        result = await executor.execute(make_context(variables={"ENV_VAR": "value123", "NUMBER": 42}), [stage("build")])
        assert result is True
        call_args = container_executor.run.call_args
        assert call_args[1]["environment"] == {"ENV_VAR": "value123", "NUMBER": "42"}
