"""
Docker 应用管理器
整合 Docker Compose 文件管理和命令执行
"""

import asyncio
import json
import logging
import shutil
import subprocess
import sys
from pathlib import Path
from typing import TextIO, cast

import yaml
from dynaconf import Dynaconf
from jinja2 import BaseLoader, Environment, StrictUndefined, TemplateError, UndefinedError

from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.entities import ApplicationRoute
from pomelo_orbit.infrastructure.config import get_project_root

logger = logging.getLogger(__name__)


class ApplicationManagerImpl(ApplicationManager):
    """应用管理器 - 整合文件管理和 Docker 操作"""

    def __init__(self, settings: Dynaconf):
        self.settings = settings

    # ==================== 路径获取 ====================
    def _detect_container_id(self) -> str | None:
        # cgroup v1: /proc/self/cgroup 每行格式如 "12:devices:/docker/<64-hex-id>"
        try:
            cgroup = Path("/proc/self/cgroup")
            if cgroup.is_file():
                for line in cgroup.read_text().splitlines():
                    if "docker" in line:
                        for part in reversed(line.split("/")):
                            if len(part) == 64 and all(c in "0123456789abcdef" for c in part):
                                return part
        except Exception:
            pass

        # cgroup v2: /proc/self/cgroup 只有 "0::/"，改从 /proc/self/mountinfo 提取
        # 其中 /etc/hostname 挂载行包含 /data/docker/containers/<64-hex-id>/hostname
        try:
            mountinfo = Path("/proc/self/mountinfo")
            if mountinfo.is_file():
                for line in mountinfo.read_text().splitlines():
                    if "/etc/hostname" in line:
                        for part in line.split("/"):
                            if len(part) == 64 and all(c in "0123456789abcdef" for c in part):
                                return part
        except Exception:
            pass

        logger.warning("Container ID auto-detection failed, falling back to local path")
        return None

    def _get_container_mount(self, container_id: str) -> str:
        result = subprocess.run(
            ["docker", "inspect", "--format", "{{json .Mounts}}", container_id],
            capture_output=True,
            text=True,
            timeout=5,
        )
        assert result.returncode == 0, "Auto-detect: docker inspect failed"
        mounts = json.loads(result.stdout)
        for mount in mounts:
            if mount.get("Type") == "bind" and mount.get("Destination", "") == "/app/data":
                source = mount.get("Source")
                return cast("str", source)
        raise RuntimeError("get container mount bind failed")

    def get_app_working_dir(self, application_code: str) -> Path:
        """获取运行时目录"""
        return get_project_root() / "data" / "cd" / application_code

    # ==================== 模板渲染 ====================
    def _render_template(self, application_code: str, content: str) -> str:
        # physical_dir: {root} 的宿主机路径（data 目录的上级）
        # physical_app_dir: {root}/data/cd/{app_code} 的宿主机路径
        container_id = self._detect_container_id()
        logger.debug(f"Rendering template, app={application_code}, container_id={container_id}")

        if container_id:
            # 容器内：挂载点是 {root}/data，往上一级得到 physical_dir
            mount = Path(self._get_container_mount(container_id))
            # pomelo-orbit 自举部署需要完整的宿主机路径配置
            physical_dir = str(mount.parent).replace("\\", "/")
            physical_app_dir = str(mount / "cd" / application_code).replace("\\", "/")
        else:
            # pomelo-orbit 自举部署需要完整的宿主机路径配置
            physical_dir = str(get_project_root()).replace("\\", "/")
            # 其他应用在 data/cd/{app_code}/ 下执行，./data 即为应用数据目录
            physical_app_dir = "."

        domain_suffix = self.settings.traefik.domain_suffix
        context = {
            "app": {
                "code": application_code,
                "physical_dir": physical_dir,
                "physical_app_dir": physical_app_dir,
            },
            "config": {
                "domain_suffix": domain_suffix,
            },
            "cert": {
                "letsencrypt": {
                    "enabled": self.settings.cert.letsencrypt.enabled,
                    "email": self.settings.cert.letsencrypt.email,
                    "challenge": self.settings.cert.letsencrypt.challenge,
                    "dns_provider": self.settings.cert.letsencrypt.dns_provider,
                }
            },
        }

        try:
            env = Environment(loader=BaseLoader(), undefined=StrictUndefined)
            template = env.from_string(content)
            rendered = template.render(context)
        except (TemplateError, UndefinedError) as e:
            raise ValueError(f"Template rendering failed: {e}") from e

        return rendered

    # ==================== 文件操作 ====================

    def _write_file_raw(self, application_code: str, filename: str, content: str, newline: str = "") -> None:
        """写文件（不渲染，content 已是最终内容）"""
        working_dir = self.get_app_working_dir(application_code)
        filepath = working_dir / filename
        filepath.parent.mkdir(parents=True, exist_ok=True)
        filepath.write_text(content, encoding="utf-8", newline=newline)
        if filepath.name == "init.sh":
            filepath.chmod(0o755)

    def _write_file(self, application_code: str, filename: str, content: str, newline: str = "") -> None:
        """写文件（.jinja 自动渲染）"""
        if filename.endswith(".jinja"):
            filename = filename.removesuffix(".jinja")
            content = self._render_template(application_code, content)
        self._write_file_raw(application_code, filename, content, newline)

    def _read_file(self, application_code: str, filename: str) -> str | None:
        working_dir = self.get_app_working_dir(application_code)
        path = working_dir / filename
        if path.exists():
            return path.read_text(encoding="utf-8")
        return None

    def _inject_route_labels(self, compose_yaml: str, routes: list[ApplicationRoute], letsencrypt: bool) -> str:
        """将路由配置注入 docker-compose YAML 的 labels，返回新的 YAML 字符串。
        route_managed=True 时必须调用，即使 routes 为空也会清除所有 service 的 traefik labels。
        """
        data = yaml.safe_load(compose_yaml)
        services: dict = data.get("services", {})

        # 先清除所有 service 的 labels（托管模式下模板里的 labels 不生效）
        # TODO: 只清除 traefik. 开头的 label，保留其他自定义 label
        for service in services.values():
            if service and "labels" in service:
                del service["labels"]

        # 再按路由配置注入
        for route in routes:
            service = services.get(route.service_name)
            if not service:
                raise ValueError(f"Service '{route.service_name}' not found in docker-compose.yml")
            router_name = route.service_name
            if letsencrypt:
                labels = [
                    "traefik.enable=true",
                    f"traefik.http.routers.{router_name}.rule=Host(`{route.domain}`)",
                    f"traefik.http.routers.{router_name}.entrypoints=websecure",
                    f"traefik.http.routers.{router_name}.tls=true",
                    f"traefik.http.routers.{router_name}.tls.certresolver=letsencrypt",
                    f"traefik.http.services.{router_name}.loadbalancer.server.port={route.port}",
                ]
            else:
                labels = [
                    "traefik.enable=true",
                    f"traefik.http.routers.{router_name}.rule=Host(`{route.domain}`)",
                    f"traefik.http.routers.{router_name}.entrypoints=web",
                    f"traefik.http.services.{router_name}.loadbalancer.server.port={route.port}",
                ]
            service["labels"] = labels

        return yaml.dump(data, allow_unicode=True, default_flow_style=False)

    # ==================== Docker 命令 ====================

    async def _run_command_win32(self, cmd: list[str], cwd: Path) -> str:
        def _run_sync() -> str:
            result = subprocess.run(
                cmd,
                cwd=str(cwd),
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            )
            std_result = result.stdout + result.stderr
            if result.returncode != 0:
                raise subprocess.CalledProcessError(result.returncode, cmd, std_result)
            return std_result

        return await asyncio.to_thread(_run_sync)

    async def _run_command_unix(self, cmd: list[str], cwd: Path) -> str:
        process = await asyncio.create_subprocess_exec(
            *cmd,
            cwd=str(cwd),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
        )
        stdout, _ = await process.communicate()
        output = stdout.decode("utf-8", errors="replace")
        if process.returncode != 0:
            raise subprocess.CalledProcessError(process.returncode or 1, cmd, output)
        return output

    async def _run_command(self, cmd: list[str], cwd: Path, log_file: TextIO | None = None) -> str:
        cmd_str = f"$ {' '.join(cmd)}"
        logger.info(f"{cmd_str}  (cwd={cwd})")
        if log_file:
            self._write_log(log_file, cmd_str)
        if sys.platform == "win32":
            return await self._run_command_win32(cmd, cwd)
        else:  # noqa: RET505
            return await self._run_command_unix(cmd, cwd)

    async def _compose_pull(self, app_dir: Path, log_file: TextIO | None = None) -> str:
        return await self._run_command(
            ["docker", "compose", "-f", "docker-compose.yml", "pull"],
            cwd=app_dir,
            log_file=log_file,
        )

    async def _compose_up(
        self,
        app_dir: Path,
        pull_policy: str = "missing",
        env_file: str | None = None,
        log_file: TextIO | None = None,
    ) -> str:
        cmd = ["docker", "compose", "-f", "docker-compose.yml"]
        if env_file:
            env_path = app_dir / env_file
            if not env_path.exists():
                raise FileNotFoundError(f"Environment file not found: {env_path}")
            cmd.extend(["--env-file", env_file])
        cmd.extend(["up", "-d", "--remove-orphans", "--pull", pull_policy])
        return await self._run_command(cmd, cwd=app_dir, log_file=log_file)

    async def _compose_down(
        self,
        app_dir: Path,
        remove_volumes: bool = False,
        env_file: str | None = None,
        log_file: TextIO | None = None,
    ) -> str:
        cmd = ["docker", "compose", "-f", "docker-compose.yml"]
        if env_file:
            cmd.extend(["--env-file", env_file])
        cmd.append("down")
        if remove_volumes:
            cmd.append("-v")
        return await self._run_command(cmd, cwd=app_dir, log_file=log_file)

    async def _compose_restart(self, app_dir: Path, env_file: str | None = None, log_file: TextIO | None = None) -> str:
        cmd = ["docker", "compose", "-f", "docker-compose.yml"]
        if env_file:
            cmd.extend(["--env-file", env_file])
        cmd.append("restart")
        return await self._run_command(cmd, cwd=app_dir, log_file=log_file)

    async def _compose_logs(self, app_dir: Path, tail: int = 100) -> str:
        return await self._run_command(
            ["docker", "compose", "-f", "docker-compose.yml", "logs", "--tail", str(tail)],
            cwd=app_dir,
        )

    async def _run_init_script(self, app_dir: Path, log_file: TextIO | None = None) -> str:
        script_path = app_dir / "init.sh"
        if not script_path.exists():
            raise FileNotFoundError(f"Init script not found: {script_path}")

        if sys.platform != "win32":
            return await self._run_command(["bash", "init.sh"], cwd=app_dir, log_file=log_file)
        else:  # noqa: RET505
            # sys.platform == "win32"
            bash_path = shutil.which("bash")
            if not bash_path:
                raise RuntimeError("bash not found in PATH. Please install Git Bash or Cygwin.")

            cmd_str = f"$ {bash_path} init.sh"
            logger.info(f"{cmd_str}  (cwd={app_dir})")
            if log_file:
                self._write_log(log_file, cmd_str)

            def _run_sync() -> str:
                result = subprocess.run(
                    [bash_path, "init.sh"],
                    cwd=str(app_dir),
                    capture_output=True,
                    text=True,
                    encoding="utf-8",
                    errors="replace",
                )
                if result.returncode != 0:
                    error_msg = result.stdout + result.stderr
                    raise subprocess.CalledProcessError(
                        result.returncode, [bash_path, "init.sh"], error_msg or "(no output)"
                    )
                return result.stdout + result.stderr

            return await asyncio.to_thread(_run_sync)

    async def _compose_ps(self, app_dir: Path) -> str:
        return await self._run_command(
            ["docker", "compose", "-f", "docker-compose.yml", "ps", "--format", "json"],
            cwd=app_dir,
        )

    async def _image_prune(self, app_dir: Path) -> str:
        return await self._run_command(
            ["docker", "image", "prune", "-f"],
            cwd=app_dir,
        )

    # ==================== 对外 API ====================

    def _get_deployment_log_path(self, application_code: str, deployment_id: str) -> Path:
        runtime_dir = self.get_app_working_dir(application_code)
        deployments_dir = runtime_dir / "deployments"
        deployments_dir.mkdir(parents=True, exist_ok=True)
        return deployments_dir / f"{deployment_id}.log"

    def _write_log(self, log_file: TextIO, message: str) -> None:
        log_file.write(message)
        if not message.endswith("\n"):
            log_file.write("\n")
        log_file.flush()

    async def deploy(
        self,
        application_code: str,
        config_files: list,
        pull_policy: str,
        deployment_id: str,
        env_file: str | None = None,
        routes: list[ApplicationRoute] | None = None,
    ) -> None:
        working_dir = self.get_app_working_dir(application_code)
        log_path = self._get_deployment_log_path(application_code, deployment_id)
        log_file = log_path.open("w", encoding="utf-8")
        letsencrypt_enabled: bool = self.settings.cert.letsencrypt.enabled

        try:
            logger.info(f"Deploy executing: app={application_code}, deployment={deployment_id}")
            self._write_log(log_file, f"Working directory: {working_dir}")

            init_script_file = None
            for config_file in config_files:
                content = config_file.content
                filename = config_file.path
                self._write_log(log_file, f"Writing configuration file: {filename}")

                # 渲染 jinja 模板
                if filename.endswith(".jinja"):
                    filename = filename.removesuffix(".jinja")
                    content = self._render_template(application_code, content)

                # 注入路由 labels（仅 docker-compose.yml，且启用了路由托管）
                if filename == "docker-compose.yml" and routes is not None:
                    content = self._inject_route_labels(content, routes, letsencrypt_enabled)

                self._write_file_raw(application_code, filename, content)

                if config_file.path == "init.sh":
                    init_script_file = config_file

            self._write_log(log_file, "Configuration files written")

            if init_script_file:
                self._write_log(log_file, "Running init script...")
                init_output = await self._run_init_script(working_dir, log_file=log_file)
                self._write_log(log_file, f"Init script output:\n{init_output}")

            self._write_log(log_file, f"Starting services (pull policy: {pull_policy})...")
            up_output = await self._compose_up(
                working_dir, pull_policy=pull_policy, env_file=env_file, log_file=log_file
            )
            self._write_log(log_file, f"Start output:\n{up_output}")
            logger.info(f"Deploy completed: app={application_code}, deployment={deployment_id}")
        finally:
            log_file.close()

    async def start(self, application_code: str) -> str:
        runtime_dir = self.get_app_working_dir(application_code)
        if not runtime_dir.exists():
            raise FileNotFoundError("应用目录不存在, 请先部署应用")
        return await self._compose_up(runtime_dir)

    async def stop(self, application_code: str, remove_volumes: bool = False, env_file: str | None = None) -> str:
        working_dir = self.get_app_working_dir(application_code)
        if not working_dir.exists():
            raise FileNotFoundError("应用目录不存在")
        return await self._compose_down(working_dir, remove_volumes=remove_volumes, env_file=env_file)

    async def restart(self, application_code: str, env_file: str | None = None) -> str:
        working_dir = self.get_app_working_dir(application_code)
        if not working_dir.exists():
            raise FileNotFoundError("应用目录不存在")
        return await self._compose_restart(working_dir, env_file=env_file)

    async def status(self, application_code: str) -> str:
        working_dir = self.get_app_working_dir(application_code)
        return await self._compose_ps(working_dir)

    async def logs(self, application_code: str, tail: int = 100) -> str:
        working_dir = self.get_app_working_dir(application_code)
        return await self._compose_logs(working_dir, tail=tail)

    async def pull(self, application_code: str) -> str:
        working_dir = self.get_app_working_dir(application_code)
        return await self._compose_pull(working_dir)

    async def cleanup(self, application_code: str) -> str:
        working_dir = self.get_app_working_dir(application_code)
        return await self._image_prune(working_dir)

    def render_compose(self, application_code: str, content: str, filename: str) -> str:
        """渲染 docker-compose 模板（不写文件），供 UI 解析 service 列表使用"""
        if filename.endswith(".jinja"):
            return self._render_template(application_code, content)
        return content

    def purge(self, application_code: str) -> None:
        working_dir = self.get_app_working_dir(application_code)
        if working_dir.exists():
            shutil.rmtree(working_dir)

    def read_deployment_log(self, application_code: str, deployment_id: str, offset: int = 0) -> tuple[str, int]:
        log_path = self._get_deployment_log_path(application_code, deployment_id)
        if not log_path.exists():
            return "", offset
        try:
            with log_path.open(encoding="utf-8") as f:
                f.seek(offset)
                content = f.read()
                return content, f.tell()
        except Exception as e:
            logger.error(
                f"Deployment log read failed: app={application_code}, deployment={deployment_id}, error={e}",
                exc_info=True,
            )
            return "", offset


__all__ = ["ApplicationManagerImpl"]
