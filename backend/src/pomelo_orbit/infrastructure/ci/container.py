"""容器执行器"""

import asyncio
import logging
import re
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from typing import TextIO

import docker
from docker.errors import ContainerError, ImageNotFound

logger = logging.getLogger(__name__)


class ContainerExecutionError(Exception):
    """容器执行错误"""


class ContainerExecutor:
    """容器执行器"""

    def __init__(self):
        self._client = None

    @property
    def client(self):
        if self._client is None:
            self._client = docker.from_env()
        return self._client

    async def run(
        self,
        image: str,
        commands: list[str] | None,
        volumes: list[str] | None,
        environment: dict[str, str] | None,
        workspace_path: Path,
        artifacts_path: Path,
        entrypoint: str = "sh",
        extra_binds: dict[str, dict[str, str]] | None = None,
        log_file: "TextIO | None" = None,
    ) -> tuple[int, str]:
        """
        执行容器

        Args:
            image: 镜像名称
            commands: 命令列表
            volumes: 卷挂载列表（格式：host_path:container_path）
            environment: 环境变量
            workspace_path: workspace 路径
            artifacts_path: artifacts 路径

        Returns:
            (exit_code, logs)

        Raises:
            ContainerExecutionError: 执行失败
        """
        try:
            # 构造卷挂载（使用 dict 格式，Docker SDK 正确处理 Windows 盘符路径）
            volume_binds = {
                str(workspace_path): {"bind": "/workspace", "mode": "rw"},
                str(artifacts_path): {"bind": "/artifacts", "mode": "rw"},
            }

            # 添加用户指定的卷
            if volumes:
                for vol in volumes:
                    if ":" in vol:
                        host_path, container_path = vol.split(":", 1)
                        volume_binds[host_path] = {"bind": container_path, "mode": "rw"}

            # 添加额外的 bind mounts（已是 dict 格式，避免 Windows 路径解析问题）
            if extra_binds:
                volume_binds.update(extra_binds)

            # 构造命令
            command = None
            if commands:
                joined = " && ".join(commands)
                command = ["-c", joined]

            # 在线程池中执行 Docker 操作（避免阻塞事件循环）
            cmd_str = command[1] if command else "(none)"
            # 脱敏：隐藏 URL 中的 token（https://token@host -> https://***@host）
            safe_cmd = re.sub(r"https://[^@]+@", "https://***@", cmd_str)
            logger.info(f"Container run: image={image}, workdir=/workspace, command={safe_cmd}")
            loop = asyncio.get_running_loop()
            exit_code, logs = await loop.run_in_executor(
                None,
                self._run_container_sync,
                image,
                command,
                volume_binds,
                environment or {},
                entrypoint,
            )
            if log_file is not None and logs:
                log_file.write(logs)
                log_file.flush()
            return exit_code, logs

        except ImageNotFound:
            raise ContainerExecutionError(f"镜像不存在: {image}")
        except ContainerError as e:
            logger.error(f"容器执行失败: {e}")
            return e.exit_status, e.stderr.decode("utf-8") if e.stderr else str(e)
        except Exception as e:
            logger.exception(f"容器执行异常: {e}")
            raise ContainerExecutionError(f"容器执行异常: {e}")

    def _run_container_sync(
        self,
        image: str,
        command: list[str] | None,
        volumes: dict[str, Any],
        environment: dict[str, str],
        entrypoint: str = "sh",
    ) -> tuple[int, str]:
        """同步执行容器（在线程池中调用）"""
        try:
            container = self.client.containers.run(
                image=image,
                command=command,
                volumes=volumes,
                environment=environment,
                working_dir="/workspace",
                entrypoint=entrypoint,
                remove=True,
                detach=False,
                stdout=True,
                stderr=True,
            )

            # container.run() 返回的是日志输出（bytes）
            logs = container.decode("utf-8") if isinstance(container, bytes) else str(container)
            return 0, logs

        except ContainerError as e:
            # 容器执行失败（非零退出码）
            raise e
        except Exception as e:
            logger.exception(f"容器执行同步异常: {e}")
            raise ContainerExecutionError(f"容器执行同步异常: {e}")
