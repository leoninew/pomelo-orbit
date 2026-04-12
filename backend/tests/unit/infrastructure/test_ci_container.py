"""CI Container Executor 单元测试"""

from pathlib import Path
from unittest.mock import Mock, patch

import pytest
from docker.errors import ImageNotFound

from pomelo_orbit.infrastructure.ci.container import ContainerExecutionError, ContainerExecutor


def make_mock_container(log_chunks: list[bytes], exit_code: int = 0) -> Mock:
    """构造模拟容器对象（detach=True 模式）"""
    container = Mock()
    container.logs.return_value = iter(log_chunks)
    container.wait.return_value = {"StatusCode": exit_code}
    return container


class TestContainerExecutor:
    """测试 ContainerExecutor"""

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_simple_command_success(self, mock_docker):
        """测试执行简单命令成功"""
        workspace_path = Path("/tmp/workspace")
        artifacts_path = Path("/tmp/artifacts")

        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Build successful\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, logs = await executor.run(
            image="python:3.12",
            commands=["echo hello"],
            volumes=None,
            environment=None,
            workspace_path=workspace_path,
            artifacts_path=artifacts_path,
        )

        assert exit_code == 0
        assert "Build successful" in logs
        mock_client.containers.run.assert_called_once()

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_multiple_commands(self, mock_docker):
        """测试执行多条命令"""
        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Command 1\nCommand 2\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["echo hello", "echo world"],
            volumes=None,
            environment=None,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 0
        call_args = mock_client.containers.run.call_args
        assert call_args[1]["command"] == ["-c", "echo hello && echo world"]

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_with_environment_variables(self, mock_docker):
        """测试传递环境变量"""
        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"ENV_VAR=value\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()
        environment = {"ENV_VAR": "value", "DEBUG": "true"}

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["env"],
            volumes=None,
            environment=environment,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 0
        call_args = mock_client.containers.run.call_args
        assert call_args[1]["environment"] == environment

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_with_custom_volumes(self, mock_docker):
        """测试自定义卷挂载"""
        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Volume mounted\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()
        volumes = ["/host/cache:/cache", "/host/data:/data"]

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["ls /cache"],
            volumes=volumes,
            environment=None,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 0
        call_args = mock_client.containers.run.call_args
        volume_binds = call_args[1]["volumes"]
        assert "/host/cache" in volume_binds
        assert volume_binds["/host/cache"]["bind"] == "/cache"

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_image_not_found(self, mock_docker):
        """测试镜像不存在"""
        mock_client = Mock()
        mock_client.containers.run.side_effect = ImageNotFound("Image not found")
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        with pytest.raises(ContainerExecutionError, match="容器执行"):
            await executor.run(
                image="nonexistent:latest",
                commands=["echo hello"],
                volumes=None,
                environment=None,
                workspace_path=Path("/tmp/workspace"),
                artifacts_path=Path("/tmp/artifacts"),
            )

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_container_error(self, mock_docker):
        """测试容器执行失败（非零退出码）"""
        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Command failed\n"], exit_code=1)
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, logs = await executor.run(
            image="python:3.12",
            commands=["exit 1"],
            volumes=None,
            environment=None,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 1
        assert "Command failed" in logs

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_workspace_and_artifacts_mounted(self, mock_docker):
        """测试 workspace 和 artifacts 目录被正确挂载"""
        workspace_path = Path("/tmp/workspace")
        artifacts_path = Path("/tmp/artifacts")

        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Mounted\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["ls"],
            volumes=None,
            environment=None,
            workspace_path=workspace_path,
            artifacts_path=artifacts_path,
        )

        assert exit_code == 0
        call_args = mock_client.containers.run.call_args
        volume_binds = call_args[1]["volumes"]
        assert str(workspace_path) in volume_binds
        assert volume_binds[str(workspace_path)]["bind"] == "/workspace"
        assert str(artifacts_path) in volume_binds
        assert volume_binds[str(artifacts_path)]["bind"] == "/artifacts"

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_working_directory_set(self, mock_docker):
        """测试工作目录设置为 /workspace"""
        mock_client = Mock()
        mock_client.containers.run.return_value = make_mock_container([b"Working dir: /workspace\n"])
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["pwd"],
            volumes=None,
            environment=None,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 0
        call_args = mock_client.containers.run.call_args
        assert call_args[1]["working_dir"] == "/workspace"

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_container_removed_after_execution(self, mock_docker):
        """测试容器执行后被显式删除"""
        mock_client = Mock()
        mock_container = make_mock_container([b"Done\n"])
        mock_client.containers.run.return_value = mock_container
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        exit_code, _ = await executor.run(
            image="python:3.12",
            commands=["echo done"],
            volumes=None,
            environment=None,
            workspace_path=Path("/tmp/workspace"),
            artifacts_path=Path("/tmp/artifacts"),
        )

        assert exit_code == 0
        mock_container.remove.assert_called_once_with(force=True)

    @pytest.mark.asyncio
    @patch("pomelo_orbit.infrastructure.ci.container.docker")
    async def test_run_generic_exception(self, mock_docker):
        """测试通用异常处理"""
        mock_client = Mock()
        mock_client.containers.run.side_effect = RuntimeError("Unexpected error")
        mock_docker.from_env.return_value = mock_client

        executor = ContainerExecutor()

        with pytest.raises(ContainerExecutionError, match="容器执行异常"):
            await executor.run(
                image="python:3.12",
                commands=["echo hello"],
                volumes=None,
                environment=None,
                workspace_path=Path("/tmp/workspace"),
                artifacts_path=Path("/tmp/artifacts"),
            )
