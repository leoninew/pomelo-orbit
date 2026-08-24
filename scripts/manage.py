#!/usr/bin/env python3
"""
Pomelo Orbit Remote Deployment Tool

统一的远程部署和管理工具，整合以下功能：
1. 使用指定镜像更新已有环境
2. SSH 隧道管理（HTTP 转发）
3. 远程命令执行
4. Docker Compose 服务管理
5. 远程 Docker 可回收空间检查与清理

配置文件: scripts/.env
  SSH_HOST                        - 远程主机地址
  SSH_USER                        - SSH 用户名
  REMOTE_DEPLOY_DIR               - 远程部署目录（必填）
  GHCR_TOKEN                      - GitHub Container Registry 令牌（可选，拉取私有镜像需要）

用法:
  python scripts/manage.py upgrade --image IMAGE      - 更新部署镜像
  python scripts/manage.py tunnel start <remote:local> [remote:local...] - 启动 SSH 隧道
  python scripts/manage.py tunnel stop                                  - 停止 SSH 隧道
  python scripts/manage.py tunnel status                                - 查看隧道状态
  python scripts/manage.py exec <command>             - 执行远程命令
  python scripts/manage.py docker-compose <args>      - 执行 docker compose 命令
  python scripts/manage.py scp to-remote [-r] <local> <remote>    - 复制本地文件或目录到远程
  python scripts/manage.py scp from-remote [-r] <remote> <local>  - 复制远程文件或目录到本地
  python scripts/manage.py backup                     - 备份远程数据目录
  python scripts/manage.py clean                      - 分析远程数据目录并清理 7 天前的 *.log
  python scripts/manage.py docker-clean [--execute]  - 检查或清理远程 Docker 可回收空间

示例:
  python scripts/manage.py upgrade --image ghcr.io/leoninew/pomelo-orbit:v1.0
  python scripts/manage.py tunnel start 8080:8888      - 转发 8080->8888
  python scripts/manage.py tunnel start 8080:8888 9090:9999
  python scripts/manage.py exec ls -al
  python scripts/manage.py docker-compose up -d
  python scripts/manage.py docker-compose logs -f
  python scripts/manage.py scp to-remote ./local.txt /tmp/local.txt
  python scripts/manage.py scp from-remote /tmp/remote.txt ./remote.txt
  python scripts/manage.py scp to-remote -r ./dist /tmp/dist
  python scripts/manage.py clean
  python scripts/manage.py docker-clean
  python scripts/manage.py docker-clean --execute
"""

import argparse
import json
import logging
import os
import platform
import socket
import shlex
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)

# 常量
SCRIPT_DIR = Path(__file__).parent
ENV_FILE = SCRIPT_DIR / ".env"

config: "Config"


class Config:
    """配置管理"""

    def __init__(self):
        self.load_env()

    def load_env(self):
        """加载 .env 配置"""
        if not ENV_FILE.exists():
            logger.error(f"配置文件不存在: {ENV_FILE}")
            sys.exit(1)

        # 读取 .env 文件
        env_vars = {}
        with ENV_FILE.open() as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    key, value = line.split("=", 1)
                    env_vars[key.strip()] = value.strip()

        # 验证必需配置
        self.ssh_host = env_vars.get("SSH_HOST")
        self.ssh_user = env_vars.get("SSH_USER")
        self.remote_deploy_dir = env_vars.get("REMOTE_DEPLOY_DIR")

        if not all([self.ssh_host, self.ssh_user, self.remote_deploy_dir]):
            logger.error("缺少必需的配置 (SSH_HOST, SSH_USER, REMOTE_DEPLOY_DIR)")
            sys.exit(1)

        # 可选配置
        self.ghcr_token = env_vars.get("GHCR_TOKEN")

    @property
    def ssh_target(self) -> str:
        """SSH 连接目标"""
        return f"{self.ssh_user}@{self.ssh_host}"


class RemoteDockerCleanupError(RuntimeError):
    """Raised when remote Docker inspection or cleanup fails."""


def run_ssh_command(command: str, description: str = "") -> str:
    """执行远程 SSH 命令"""
    try:
        result = subprocess.run(
            ["ssh", config.ssh_target, command],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        logger.error(f"{description or command}")
        logger.error(f"输出: {e.stderr}")
        sys.exit(1)


def copy_to_remote(local_path: str, remote_path: str, recursive: bool = False) -> None:
    """复制本地文件或目录到远程服务器"""
    cmd = ["scp"]
    if recursive:
        assert Path(local_path).is_dir(), f"本地源路径不是目录: {local_path}"
        cmd.append("-r")
    cmd.extend([local_path, f"{config.ssh_target}:{remote_path}"])
    subprocess.run(cmd, check=True)


def copy_from_remote(
    remote_path: str, local_path: str, recursive: bool = False
) -> None:
    """复制远程文件或目录到本地"""
    cmd = ["scp"]
    if recursive:
        run_ssh_command(f"test -d {shlex.quote(remote_path)}", "检查远程源目录")
        cmd.append("-r")
    cmd.extend([f"{config.ssh_target}:{remote_path}", local_path])
    subprocess.run(cmd, check=True)


class SSHTunnel:
    """SSH 隧道管理"""

    STATE_FILE = SCRIPT_DIR / "tunnel.log"

    def __init__(self, config: Config):
        self.config = config

    def start(self, remote_port: int, local_port: int) -> bool:
        """启动 SSH 隧道，支持多次调用添加多个端口转发"""
        # 检查该本地端口是否已在转发
        tunnels = self._load_tunnels()
        for t in tunnels:
            if t["local_port"] == local_port and self._port_in_use(local_port):
                logger.info(
                    f"端口转发已在运行: localhost:{t['local_port']} -> {t['ssh_host']}:{t['remote_port']}"
                )
                return False

        logger.info(
            f"启动端口转发: localhost:{local_port} -> {self.config.ssh_host}:{remote_port}"
        )

        subprocess.Popen(
            [
                "ssh",
                "-fN",
                "-o",
                "ControlMaster=no",
                "-o",
                "ServerAliveInterval=60",
                "-o",
                "ExitOnForwardFailure=yes",
                "-L",
                f"{local_port}:localhost:{remote_port}",
                self.config.ssh_target,
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        # 等待端口就绪并获取 PID
        for _ in range(10):
            time.sleep(0.5)
            if self._port_in_use(local_port):
                pid = self._find_pid_by_port(local_port)
                if pid:
                    tunnels = [t for t in tunnels if t["local_port"] != local_port]
                    tunnels.append(
                        {
                            "local_port": local_port,
                            "remote_port": remote_port,
                            "ssh_host": self.config.ssh_host,
                            "ssh_user": self.config.ssh_user,
                            "pid": pid,
                        }
                    )
                    self._save_tunnels(tunnels)
                    logger.info(f"访问地址: http://localhost:{local_port}")
                    return True

        logger.error("SSH 隧道启动失败")
        return False

    def stop(self) -> bool:
        """停止所有 SSH 隧道"""
        tunnels = self._load_tunnels()
        if not tunnels:
            logger.info("隧道未运行")
            return False

        success = True
        for t in tunnels:
            pid = t.get("pid")
            if not pid:
                logger.warning(f"隧道 localhost:{t['local_port']} 缺少 PID, 跳过")
                success = False
                continue
            try:
                logger.info(
                    f"终止隧道进程: pid={pid}, local_port={t['local_port']}, "
                    f"remote={t['ssh_host']}:{t['remote_port']}"
                )
                if platform.system() == "Windows":
                    result = subprocess.run(
                        ["taskkill", "/F", "/PID", str(pid)],
                        capture_output=True,
                        text=True,
                    )
                else:
                    result = subprocess.run(
                        ["kill", "-9", str(pid)], capture_output=True, text=True
                    )
                if result.returncode == 0:
                    logger.info("  已停止")
                else:
                    logger.warning("  进程不存在或已终止")
            except Exception as e:
                logger.error(f"  停止失败: {e}")
                success = False

        self.STATE_FILE.unlink(missing_ok=True)
        return success

    def status(self) -> bool:
        """查看所有隧道状态"""
        tunnels = self._load_tunnels()
        if not tunnels:
            logger.info("未运行")
            return False

        active = []
        stale = []
        for t in tunnels:
            if self._port_in_use(t["local_port"]):
                active.append(t)
            else:
                stale.append(t)

        for t in active:
            logger.info(
                f"运行中: localhost:{t['local_port']} -> {t['ssh_host']}:{t['remote_port']} (pid={t['pid']})"
            )

        # 清理已失效的隧道记录
        if stale:
            for t in stale:
                logger.info(
                    f"已失效: localhost:{t['local_port']} -> {t['ssh_host']}:{t['remote_port']}"
                )
            if active:
                self._save_tunnels(active)
            else:
                self.STATE_FILE.unlink(missing_ok=True)

        if not active:
            logger.info("未运行")
        return bool(active)

    def is_running(self) -> bool:
        tunnels = self._load_tunnels()
        return any(self._port_in_use(t["local_port"]) for t in tunnels)

    def _port_in_use(self, port: int) -> bool:
        """检查本地端口是否被占用"""
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            return s.connect_ex(("127.0.0.1", int(port))) == 0

    def _find_pid_by_port(self, port: int) -> int | None:
        """通过端口查找进程 PID"""
        try:
            if platform.system() == "Windows":
                ps_cmd = f"Get-NetTCPConnection -LocalPort {port} -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess | Select-Object -First 1"
                result = subprocess.run(
                    ["powershell", "-Command", ps_cmd],
                    capture_output=True,
                    text=True,
                )
                pid_str = result.stdout.strip()
                return int(pid_str) if pid_str.isdigit() else None
            # macOS/Linux
            result = subprocess.run(
                ["lsof", "-ti", f":{port}"],
                capture_output=True,
                text=True,
            )
            pid_str = result.stdout.strip().split("\n")[0]
            return int(pid_str) if pid_str.isdigit() else None
        except Exception:
            return None

    def _save_tunnels(self, tunnels: list):
        """保存隧道列表"""
        self.STATE_FILE.write_text(json.dumps({"tunnels": tunnels}))

    def _load_tunnels(self) -> list:
        """加载隧道列表，兼容旧的单隧道格式"""
        if not self.STATE_FILE.exists():
            return []
        try:
            data = json.loads(self.STATE_FILE.read_text())
            # 兼容旧格式（单个隧道对象）
            if isinstance(data, dict) and "tunnels" not in data:
                data["local_port"] = int(data["local_port"])
                data["remote_port"] = int(data["remote_port"])
                return [data]
            tunnels = data.get("tunnels", [])
            for t in tunnels:
                t["local_port"] = int(t["local_port"])
                t["remote_port"] = int(t["remote_port"])
            return tunnels
        except Exception:
            return []


class RemoteExecutor:
    """远程命令执行"""

    def __init__(self, config: Config):
        self.config = config

    def exec(self, command_args: list[str], workdir: str | None = None):
        """执行远程命令"""
        if not command_args:
            logger.error("未提供命令")
            sys.exit(1)
        # 将参数列表拼接为命令字符串
        command = " ".join(command_args)
        # 如果指定了工作目录，在命令前添加 cd
        if workdir:
            command = f"cd {workdir} && {command}"
        # 使用 subprocess.run 避免本地 shell 解析问题
        subprocess.run(["ssh", self.config.ssh_target, command], check=False)

    def docker_compose(self, args: list[str]):
        """执行 docker compose 命令"""
        args_str = " ".join(args)
        # 使用双引号包裹整个命令，避免引号嵌套问题
        os.system(
            f'ssh {self.config.ssh_target} "cd {self.config.remote_deploy_dir} && docker compose --env-file .env {args_str}"'
        )


class Deployer:
    """部署管理"""

    def __init__(self, config: Config):
        self.config = config

    def upgrade(self, image: str, skip_pull: bool = False):
        """更新部署镜像并重启服务"""
        if not image:
            logger.error("upgrade 命令必须指定 --image 参数")
            sys.exit(1)

        logger.info("=== Pomelo Orbit 更新部署 ===\n")
        logger.info(f"目标服务器: {self.config.ssh_target}")
        logger.info(f"部署目录: {self.config.remote_deploy_dir}\n")

        # 检查是否已部署
        try:
            run_ssh_command(
                f"test -f {self.config.remote_deploy_dir}/docker-compose.yml",
                "检查部署状态",
            )
        except SystemExit:
            logger.error("未检测到已有部署: 缺少 docker-compose.yml")
            sys.exit(1)

        # 确保 traefik 网络存在并登录 ghcr
        self._ensure_traefik_network()
        self._docker_login()

        # 更新镜像
        self._update_image(image)

        # 检查镜像是否已存在，不存在才拉取
        if not skip_pull:
            self._pull_image_if_needed(image)

        # 重启服务
        self._restart_service()

        logger.info("\n=== 更新成功! ===\n")
        logger.info(f"部署目录: {self.config.remote_deploy_dir}\n")

    def _update_image(self, image: str):
        """后续升级：只更新镜像版本"""
        logger.info(f"更新镜像版本: {image}")
        # 使用 sed 远程修改 docker-compose.yml 的镜像行
        run_ssh_command(
            f"sed -i 's|image:.*|image: {image}|' {self.config.remote_deploy_dir}/docker-compose.yml",
            "更新镜像版本",
        )
        logger.info("镜像版本更新完成\n")

    def _ensure_traefik_network(self):
        """确保 traefik Docker 网络存在且标签正确"""
        logger.info("检查 Traefik 网络...")
        # docker-compose.yml 声明了 external: true，网络必须由我们手动创建
        # 带上 com.docker.compose.network 标签以避免 compose 的 warning
        run_ssh_command(
            "docker network inspect traefik >/dev/null || "
            "docker network create --label com.docker.compose.network=traefik traefik",
            "创建 traefik 网络",
        )
        logger.info("Traefik 网络就绪\n")

    def _docker_login(self):
        """登录 GitHub Container Registry（若配置了 GHCR_TOKEN）"""
        if not self.config.ghcr_token:
            logger.info("GHCR_TOKEN 未配置，跳过 docker login\n")
            return
        logger.info("登录 GitHub Container Registry...")
        run_ssh_command(
            f"echo '{self.config.ghcr_token}' | docker login ghcr.io -u leoninew --password-stdin",
            "ghcr.io 登录",
        )
        logger.info("登录成功\n")

    def _pull_image_if_needed(self, image: str):
        """检查镜像是否存在，不存在才拉取"""
        logger.info("检查 Docker 镜像...")
        logger.info(f"  镜像: {image}")

        # 检查镜像是否已存在
        result = subprocess.run(
            ["ssh", self.config.ssh_target, f"docker images -q {image}"],
            capture_output=True,
            text=True,
        )

        if result.stdout.strip():
            logger.info("  镜像已存在，跳过拉取\n")
        else:
            logger.info("  镜像不存在，开始拉取...")
            run_ssh_command(f"docker pull {image}", "拉取镜像")
            logger.info("镜像拉取完成\n")

    def _restart_service(self):
        """重启服务"""
        logger.info("重启服务...")
        run_ssh_command(
            f"cd {self.config.remote_deploy_dir} && docker compose --env-file .env up -d",
            "重启服务",
        )
        logger.info("服务重启完成\n")


def backup(cfg: "Config", remote_dir: str) -> None:
    """将远程目录打包压缩后下载到本地 scripts/backup/"""
    date_str = datetime.now().strftime("%Y%m%d-%H%M%S")
    remote_archive = f"/tmp/pomelo-orbit-backup-{date_str}.tar.gz"
    local_backup_dir = SCRIPT_DIR / "backup"
    local_backup_dir.mkdir(exist_ok=True)
    local_archive = local_backup_dir / f"data-{date_str}.tar.gz"

    logger.info(f"备份远程目录: {remote_dir}")
    logger.info("排除: data/pipeline；data/deployment 仅保留 */docker-compose.yml")
    run_ssh_command(
        " ".join(
            [
                f"cd {shlex.quote(remote_dir)} &&",
                "{",
                "find . -path ./data/pipeline -prune -o "
                "-path ./data/deployment -prune -o -print0;",
                "if [ -d ./data/deployment ]; then",
                "find ./data/deployment -mindepth 2 -maxdepth 2 "
                "-type f -name docker-compose.yml -print0;",
                "fi;",
                "}",
                "| tar --ignore-failed-read --null --verbatim-files-from "
                f"--no-recursion -czf {shlex.quote(remote_archive)} --files-from=-",
            ]
        ),
        "压缩远程目录",
    )
    logger.info(f"下载备份文件: {local_archive}")
    subprocess.run(
        ["scp", f"{cfg.ssh_target}:{remote_archive}", str(local_archive)],
        check=True,
    )
    run_ssh_command(f"rm -f {remote_archive}", "清理远程临时文件")
    logger.info(f"备份完成: {local_archive}")


def clean(cfg: "Config", remote_dir: str, days: int) -> None:
    """分析远程数据目录并清理指定天数前的 *.log 文件"""
    assert days >= 0, "days 必须大于等于 0"

    logger.info(f"分析远程数据目录: {remote_dir}")
    logger.info(f"清理范围: {days} 天前的 *.log 文件")

    script = f"""
set -euo pipefail
base={shlex.quote(remote_dir)}
days={days}

if [ ! -d "$base" ]; then
  printf 'data directory does not exist: %s\\n' "$base" >&2
  exit 1
fi

printf '%s\\n' '--- data directory ---'
ls -ld "$base"
printf '%s\\n' '--- total size ---'
du -sh "$base"
printf '%s\\n' '--- top level sizes ---'
du -h -d 1 "$base" 2>/dev/null | sort -hr | sed -n '1,50p'
printf '%s\\n' '--- large files >=100M ---'
find "$base" -xdev -type f -size +100M -exec ls -alh {{}} + 2>/dev/null | sort -k5 -hr | sed -n '1,50p'
printf '%s\\n' '--- top 30 files ---'
find "$base" -xdev -type f -exec ls -alh {{}} + 2>/dev/null | sort -k5 -hr | sed -n '1,30p'
printf '%s\\n' '--- *.log files older than threshold before cleanup ---'
sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -exec ls -alh {{}} + | sort -k5 -hr | sed -n '1,100p'
printf '%s\\n' '--- cleanup summary before delete ---'
old_log_count=$(sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -print | wc -l)
printf 'count=%s\\n' "$old_log_count"
if [ "$old_log_count" -gt 0 ]; then
  sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -exec du -ch {{}} + | tail -n 1
else
  printf 'total=0\\n'
fi
printf '%s\\n' '--- deleting old *.log files ---'
deleted_count=$(sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -print -delete | wc -l)
printf 'deleted_count=%s\\n' "$deleted_count"
printf '%s\\n' '--- verify remaining old *.log files ---'
remaining_count=$(sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -print | wc -l)
printf 'remaining_count=%s\\n' "$remaining_count"
if [ "$remaining_count" -gt 0 ]; then
  sudo -n find "$base" -xdev -type f -name '*.log' -mtime +"$days" -exec du -ch {{}} + | tail -n 1
else
  printf 'total=0\\n'
fi
printf '%s\\n' '--- total size after cleanup ---'
du -sh "$base"
"""

    result = subprocess.run(
        ["ssh", cfg.ssh_target, "bash -s"],
        input=script.encode("utf-8"),
    )
    if result.returncode != 0:
        logger.error("远程清理失败；请确认远程用户可执行免密 sudo")
        sys.exit(result.returncode)


def run_docker_cleanup_command(cfg: "Config", command: str, title: str) -> str:
    """Run an SSH Docker maintenance command and log its sanitized output."""
    logger.info(title)
    try:
        result = subprocess.run(
            ["ssh", cfg.ssh_target, command],
            check=True,
            text=True,
            capture_output=True,
        )
    except FileNotFoundError as error:
        raise RemoteDockerCleanupError("ssh executable was not found") from error
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip() or error.stdout.strip() or "no command output"
        raise RemoteDockerCleanupError(f"{title} failed: {detail}") from error

    output = result.stdout.strip()
    if output:
        for line in output.splitlines():
            logger.info("  %s", line)
    return output


def show_docker_cleanup_summary(cfg: "Config", phase: str) -> None:
    """Log remote filesystem, Docker, and active-container summaries."""
    logger.info("%s disk usage", phase)
    run_docker_cleanup_command(cfg, "df -hPT /", "querying root filesystem")
    logger.info("%s Docker usage", phase)
    run_docker_cleanup_command(cfg, "docker system df", "querying Docker disk usage")
    logger.info("active containers")
    run_docker_cleanup_command(
        cfg,
        "docker ps --format 'table {{.Names}}\\t{{.Image}}\\t{{.Status}}'",
        "querying active containers",
    )


def docker_clean(cfg: "Config", execute: bool) -> None:
    """Inspect remote Docker data, deleting only after explicit authorization."""
    try:
        logger.info("remote Docker cleanup target: %s", cfg.ssh_target)
        show_docker_cleanup_summary(cfg, "before")

        if not execute:
            logger.info("dry run complete; no Docker data was removed")
            logger.info(
                "rerun with docker-clean --execute to remove cache and unused images"
            )
            return

        logger.warning("removing all reclaimable BuildKit cache")
        run_docker_cleanup_command(
            cfg,
            "docker builder prune --all --force",
            "cleaning BuildKit cache",
        )
        logger.warning("removing images unused by every container")
        run_docker_cleanup_command(
            cfg,
            "docker image prune --all --force",
            "cleaning unused images",
        )
        show_docker_cleanup_summary(cfg, "after")
        logger.info("remote Docker cleanup completed")
    except RemoteDockerCleanupError as error:
        logger.error("remote Docker cleanup failed: %s", error)
        sys.exit(1)


def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="Pomelo Orbit 远程部署和管理工具")
    subparsers = parser.add_subparsers(dest="command", help="命令")

    # upgrade 命令
    upgrade_parser = subparsers.add_parser("upgrade", help="更新已有环境")
    upgrade_parser.add_argument(
        "--image", required=True, help="指定 Docker 镜像（必填）"
    )
    upgrade_parser.add_argument("--skip-pull", action="store_true", help="跳过拉取镜像")

    # tunnel 命令
    tunnel_parser = subparsers.add_parser("tunnel", help="SSH 隧道管理")
    tunnel_parser.add_argument(
        "action", choices=["start", "stop", "status"], help="操作"
    )
    tunnel_parser.add_argument(
        "ports",
        nargs="*",
        help="端口映射，格式: remote_port:local_port，如 8080:8888",
    )

    # exec 命令
    exec_parser = subparsers.add_parser("exec", help="执行远程命令")
    exec_parser.add_argument(
        "-w", "--workdir", help="工作目录（在执行命令前切换到此目录）"
    )
    exec_parser.add_argument(
        "remote_command", nargs=argparse.REMAINDER, help="要执行的命令"
    )

    # docker-compose 命令
    dc_parser = subparsers.add_parser("docker-compose", help="执行 docker compose 命令")
    dc_parser.add_argument(
        "dc_args", nargs=argparse.REMAINDER, help="docker compose 参数"
    )

    # scp 命令
    scp_parser = subparsers.add_parser("scp", help="复制本地和远程文件")
    scp_parser.add_argument(
        "direction", choices=["to-remote", "from-remote"], help="复制方向"
    )
    scp_parser.add_argument(
        "-r", "--recursive", action="store_true", help="递归复制目录"
    )
    scp_parser.add_argument("source", help="源路径")
    scp_parser.add_argument("destination", help="目标路径")

    # ssh 命令
    subparsers.add_parser("ssh", help="SSH 连接到远程服务器")

    # backup 命令
    backup_parser = subparsers.add_parser("backup", help="备份远程数据目录")
    backup_parser.add_argument(
        "--remote-dir",
        help="远程备份目录(默认: REMOTE_DEPLOY_DIR)",
    )
    # clean 命令
    clean_parser = subparsers.add_parser("clean", help="分析远程数据目录并清理旧日志")
    clean_parser.add_argument(
        "--days",
        type=int,
        default=7,
        help="清理多少天前的 *.log 文件(默认: 7)",
    )

    # docker-clean 命令
    docker_clean_parser = subparsers.add_parser(
        "docker-clean",
        help="检查或清理远程 Docker 可回收空间",
    )
    docker_clean_parser.add_argument(
        "--execute",
        action="store_true",
        help="删除全部 BuildKit 缓存和未被容器使用的镜像",
    )

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        sys.exit(1)

    global config
    config = Config()

    if args.command == "upgrade":
        deployer = Deployer(config)
        deployer.upgrade(args.image, args.skip_pull)
    elif args.command == "tunnel":
        tunnel = SSHTunnel(config)
        if args.action == "start":
            if not args.ports:
                tunnel_parser.error(
                    "tunnel start 必须指定至少一个 remote_port:local_port 端口映射"
                )
            port_mappings = []
            for p in args.ports:
                parts = p.split(":")
                if len(parts) != 2 or not parts[0].isdigit() or not parts[1].isdigit():
                    logger.error(f"无效的端口格式: {p}, 应为 remote_port:local_port")
                    sys.exit(1)
                remote, local = int(parts[0]), int(parts[1])
                port_mappings.append((remote, local))
            for remote, local in port_mappings:
                tunnel.start(remote_port=remote, local_port=local)
        elif args.action == "stop":
            tunnel.stop()
        else:
            getattr(tunnel, args.action)()
    elif args.command == "exec":
        executor = RemoteExecutor(config)
        executor.exec(args.remote_command, workdir=args.workdir)
    elif args.command == "docker-compose":
        executor = RemoteExecutor(config)
        executor.docker_compose(args.dc_args)
    elif args.command == "scp":
        if args.direction == "to-remote":
            logger.info(
                f"复制到远程: {args.source} -> {config.ssh_target}:{args.destination}"
            )
            copy_to_remote(args.source, args.destination, recursive=args.recursive)
        else:
            logger.info(
                f"复制到本地: {config.ssh_target}:{args.source} -> {args.destination}"
            )
            copy_from_remote(args.source, args.destination, recursive=args.recursive)
        logger.info("文件复制完成")
    elif args.command == "ssh":
        os.system(f"ssh {config.ssh_target}")
    elif args.command == "backup":
        backup(config, args.remote_dir or config.remote_deploy_dir)
    elif args.command == "clean":
        clean(config, f"{config.remote_deploy_dir}/data", args.days)
    elif args.command == "docker-clean":
        docker_clean(config, args.execute)


if __name__ == "__main__":
    main()
