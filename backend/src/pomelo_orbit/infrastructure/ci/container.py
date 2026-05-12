"""容器执行器"""

import asyncio
import logging
import re
from functools import partial
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from typing import TextIO

    from pomelo_orbit.infrastructure.ci.executor_impl import VolumeMount

import docker
from docker.errors import ImageNotFound

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
        volumes: list["VolumeMount"],
        environment: dict[str, str] | None,
        entrypoint: str = "sh",
        log_file: "TextIO | None" = None,
    ) -> tuple[int, str]:
        """
        执行容器

        Args:
            image: 镜像名称
            commands: 命令列表
            volumes: 卷挂载列表
            environment: 环境变量

        Returns:
            (exit_code, logs)

        Raises:
            ContainerExecutionError: 执行失败
        """
        try:
            # 构造卷挂载：将 VolumeMount 对象转换为 Docker API 格式
            volume_binds: dict[str, dict[str, str]] = {}
            for vol in volumes:
                volume_binds.update(vol.to_docker_format())
            volume_binds["/var/run/docker.sock"] = {"bind": "/var/run/docker.sock", "mode": "rw"}

            # 构造命令：用换行符拼接，保留注释和空行语义，sh -c 按行顺序执行
            # 启用 shell trace 模式（-x 参数），输出每条执行的命令（增强可观测性）
            command = None
            if commands:
                joined = "\n".join(commands)
                # 在脚本开头添加 set -x，让 shell 输出每条执行的命令
                command = ["-x", "-c", joined]

            # 在线程池中执行 Docker 操作（避免阻塞事件循环）
            cmd_str = command[1] if command else "(none)"
            # 脱敏：隐藏 URL 中的 token（https://token@host -> https://***@host）
            safe_cmd = re.sub(r"https://[^@]+@", "https://***@", cmd_str)
            logger.info(f"Container run: image={image}, workdir=/workspace, command={safe_cmd}")
            loop = asyncio.get_running_loop()
            fn = partial(
                self._run_container_sync, image, command, volume_binds, environment or {}, entrypoint, log_file
            )
            exit_code, logs = await loop.run_in_executor(None, fn)
            return exit_code, logs

        except ImageNotFound:
            raise ContainerExecutionError(f"镜像不存在: {image}")
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
        log_file: "TextIO | None" = None,
    ) -> tuple[int, str]:
        """同步执行容器（在线程池中调用），实时流式写入日志"""
        container = None
        try:
            container = self.client.containers.run(
                image=image,
                command=command,
                volumes=volumes,
                environment=environment,
                working_dir="/workspace",
                entrypoint=entrypoint,
                remove=False,
                detach=True,
                stdout=True,
                stderr=True,
            )

            # 流式读取日志，实时写入文件
            all_logs: list[str] = []
            for chunk in container.logs(stream=True, follow=True):
                line = chunk.decode("utf-8") if isinstance(chunk, bytes) else str(chunk)
                all_logs.append(line)
                if log_file is not None:
                    log_file.write(line)
                    log_file.flush()

            result = container.wait()
            exit_code = result.get("StatusCode", 1)
            if exit_code != 0:
                error_detail = (result.get("Error") or {}).get("Message", "")
                if error_detail:
                    logger.warning(f"Container exited with error: exit_code={exit_code}, detail={error_detail}")
            return exit_code, "".join(all_logs)

        except Exception as e:
            logger.exception(f"容器执行同步异常: {e}")
            raise ContainerExecutionError(f"容器执行同步异常: {e}")
        finally:
            if container is not None:
                try:
                    container.remove(force=True)
                except Exception as e:
                    logger.warning(f"Container remove failed: {e}")
