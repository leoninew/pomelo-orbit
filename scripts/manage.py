#!/usr/bin/env python3
"""
Pomelo Orbit Remote Deployment Tool

统一的远程部署和管理工具，整合以下功能：
1. 使用指定镜像部署全新环境
2. 使用指定镜像更新已有环境
3. SSH 隧道管理（HTTP 转发）
4. 远程命令执行
5. Docker Compose 服务管理

配置文件: scripts/.env
  SSH_HOST                        - 远程主机地址
  SSH_USER                        - SSH 用户名
  REMOTE_PORT                     - 远程端口
  REMOTE_DEPLOY_DIR               - 远程部署目录（必填）
  POMELO_ORBIT_JWT__SECRET_KEY    - JWT 密钥（必填）
  POMELO_ORBIT_IMAGE              - Docker 镜像（可选，默认从数据库读取）
  POMELO_ORBIT_TRAEFIK__API_URL   - Traefik API 地址（可选）
  GHCR_TOKEN                      - GitHub Container Registry 令牌（可选，拉取私有镜像需要）

用法:
  python scripts/renew.py install [--image IMAGE]    - 初次部署（从数据库读取配置）
  python scripts/renew.py upgrade [--image IMAGE]    - 更新部署（只更新镜像和 .env）
  python scripts/renew.py tunnel start [ports...]    - 启动 SSH 隧道
  python scripts/renew.py tunnel stop                - 停止 SSH 隧道
  python scripts/renew.py tunnel status              - 查看隧道状态
  python scripts/renew.py exec <command>             - 执行远程命令
  python scripts/renew.py docker-compose <args>      - 执行 docker compose 命令
  python scripts/renew.py backup                     - 备份远程数据目录

示例:
  python scripts/renew.py install --image ghcr.io/leoninew/pomelo-orbit:latest
  python scripts/renew.py upgrade --image ghcr.io/leoninew/pomelo-orbit:v1.0
  python scripts/renew.py tunnel start               - 使用默认配置启动隧道
  python scripts/renew.py tunnel start 8080           - 转发 8080->8080
  python scripts/renew.py tunnel start 8080:8888      - 转发 8080->8888
  python scripts/renew.py tunnel start 8080 9090      - 转发 8080->8080 和 9090->9090
  python scripts/renew.py tunnel start 8080:8888 9090:9999
  python scripts/renew.py exec ls -al
  python scripts/renew.py docker-compose up -d
  python scripts/renew.py docker-compose logs -f
"""

import argparse
import json
import logging
import os
import platform
import re
import socket
import subprocess
import sys
import tempfile
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
PROJECT_ROOT = SCRIPT_DIR.parent
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
        self.remote_port = env_vars.get("REMOTE_PORT")
        self.jwt_secret = env_vars.get("POMELO_ORBIT_JWT__SECRET_KEY")

        self.remote_deploy_dir = env_vars.get("REMOTE_DEPLOY_DIR")

        if not all(
            [self.ssh_host, self.ssh_user, self.remote_port, self.remote_deploy_dir]
        ):
            logger.error(
                "缺少必需的配置 (SSH_HOST, SSH_USER, REMOTE_PORT, REMOTE_DEPLOY_DIR)"
            )
            sys.exit(1)

        # 可选配置
        self.local_port = env_vars.get("LOCAL_PORT", self.remote_port)
        self.image = env_vars.get("POMELO_ORBIT_IMAGE")
        self.traefik_api_url = env_vars.get("POMELO_ORBIT_TRAEFIK__API_URL")
        self.ghcr_token = env_vars.get("GHCR_TOKEN")

    @property
    def ssh_target(self) -> str:
        """SSH 连接目标"""
        return f"{self.ssh_user}@{self.ssh_host}"


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


def copy_to_remote(local_path: str, remote_path: str):
    """复制文件到远程服务器"""
    subprocess.run(
        ["scp", local_path, f"{config.ssh_target}:{remote_path}"],
        check=True,
    )


class SSHTunnel:
    """SSH 隧道管理"""

    STATE_FILE = SCRIPT_DIR / "tunnel.log"

    def __init__(self, config: Config):
        self.config = config

    def start(
        self, remote_port: int | None = None, local_port: int | None = None
    ) -> bool:
        """启动 SSH 隧道，支持多次调用添加多个端口转发"""
        actual_remote_port = int(remote_port or self.config.remote_port)
        actual_local_port = int(local_port or remote_port or self.config.local_port)

        # 检查该本地端口是否已在转发
        tunnels = self._load_tunnels()
        for t in tunnels:
            if t["local_port"] == actual_local_port and self._port_in_use(
                actual_local_port
            ):
                logger.info(
                    f"端口转发已在运行: localhost:{t['local_port']} -> {t['ssh_host']}:{t['remote_port']}"
                )
                return False

        logger.info(
            f"启动端口转发: localhost:{actual_local_port} -> {self.config.ssh_host}:{actual_remote_port}"
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
                f"{actual_local_port}:localhost:{actual_remote_port}",
                self.config.ssh_target,
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        # 等待端口就绪并获取 PID
        for _ in range(10):
            time.sleep(0.5)
            if self._port_in_use(actual_local_port):
                pid = self._find_pid_by_port(actual_local_port)
                if pid:
                    tunnels = [
                        t for t in tunnels if t["local_port"] != actual_local_port
                    ]
                    tunnels.append(
                        {
                            "local_port": actual_local_port,
                            "remote_port": actual_remote_port,
                            "ssh_host": self.config.ssh_host,
                            "ssh_user": self.config.ssh_user,
                            "pid": pid,
                        }
                    )
                    self._save_tunnels(tunnels)
                    logger.info(f"访问地址: http://localhost:{actual_local_port}")
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
                    f"终止进程 {pid} (localhost:{t['local_port']} -> {t['ssh_host']}:{t['remote_port']})..."
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

    def exec(self, command_args: list[str]):
        """执行远程命令"""
        if not command_args:
            logger.error("未提供命令")
            sys.exit(1)
        # 将参数列表拼接为命令字符串
        command = " ".join(command_args)
        os.system(f"ssh {self.config.ssh_target} {command}")

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

    def install(self, image: str | None = None):
        """初次部署：从数据库读取配置并部署"""
        logger.info("=== Pomelo Orbit 初次部署 ===\n")
        logger.info(f"目标服务器: {self.config.ssh_target}")
        logger.info(f"部署目录: {self.config.remote_deploy_dir}\n")

        # 检查环境
        self._check_environment()

        # 创建部署目录
        self._create_deploy_dir()

        # 从数据库读取并渲染配置
        self._deploy_config_first(image or self.config.image)

        # 传输 .env
        self._deploy_env()

        # 传输管理脚本
        self._deploy_boot_script()

        # 确保 traefik 网络存在并登录 ghcr
        self._ensure_traefik_network()
        self._docker_login()

        # 拉取镜像
        self._pull_image()

        # 启动服务
        self._start_service()

        logger.info("\n=== 部署成功! ===\n")
        logger.info(f"部署目录: {self.config.remote_deploy_dir}\n")

    def upgrade(self, image: str, skip_pull: bool = False):
        """更新部署：只更新镜像和 .env"""
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
            logger.error("未检测到已有部署, 请先执行 install 命令")
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

    def _check_environment(self):
        """检查本地和远程环境"""
        logger.info("检查本地环境...")
        if subprocess.run(["which", "pomelo-db"], capture_output=True).returncode != 0:
            logger.error("需要 pomelo-db")
            sys.exit(1)
        if subprocess.run(["which", "jq"], capture_output=True).returncode != 0:
            logger.error("需要 jq")
            sys.exit(1)
        logger.info("本地环境检查通过\n")

        logger.info("检查远程环境...")
        run_ssh_command("command -v docker", "检查 docker")
        run_ssh_command("docker compose version", "检查 docker compose")
        logger.info("远程环境检查通过\n")

    def _create_deploy_dir(self):
        """创建远程部署目录"""
        logger.info("创建远程部署目录...")
        run_ssh_command(f"mkdir -p {self.config.remote_deploy_dir}", "创建目录")
        logger.info("目录创建完成\n")

    def _deploy_config_first(self, image: str | None):
        """初次部署：从数据库读取并渲染配置文件"""
        logger.info("从数据库获取配置文件...")
        os.chdir(PROJECT_ROOT)

        # 查询 pomelo-orbit 应用 ID
        result = subprocess.run(
            [
                "pomelo-db",
                "-d",
                "local",
                "-e",
                "SELECT id FROM application WHERE code='pomelo-orbit'",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        data = json.loads(result.stdout)
        if not data.get("data"):
            logger.error("未找到 pomelo-orbit 应用")
            sys.exit(1)
        app_id = data["data"][0]["id"]
        logger.info(f"  - 应用 ID: {app_id}")

        # 查询 docker-compose.yml.jinja 模板
        result = subprocess.run(
            [
                "pomelo-db",
                "-d",
                "local",
                "-e",
                f"SELECT content FROM application_config_file WHERE application_id='{app_id}' AND path='docker-compose.yml.jinja'",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        data = json.loads(result.stdout)
        if not data.get("data"):
            logger.error("未找到 docker-compose.yml.jinja 模板")
            sys.exit(1)
        config_content = data["data"][0]["content"]

        # 渲染 Jinja 变量
        # 固定路径
        config_content = re.sub(
            r"\{\{\s*app\.physical_data_dir\s*\}\}",
            f"{self.config.remote_deploy_dir}/data",
            config_content,
        )
        # pomelo-orbit 固定使用 lvh.me 域名
        config_content = re.sub(
            r"\{\{\s*config\.domain_suffix\s*\}\}", "lvh.me", config_content
        )

        # 替换镜像
        if image:
            config_content = re.sub(r"image:\s*\S+", f"image: {image}", config_content)
            logger.info(f"  - 使用镜像: {image}")

        # 检查是否有未渲染的 Jinja 变量
        if re.search(r"\{\{.*?\}\}", config_content):
            logger.warning("模板中仍有未渲染的 Jinja 变量, 请检查")

        # 写入临时文件并传输
        with tempfile.NamedTemporaryFile(
            mode="w", delete=False, suffix=".yml", encoding="utf-8"
        ) as f:
            f.write(config_content)
            temp_file = f.name

        copy_to_remote(temp_file, f"{self.config.remote_deploy_dir}/docker-compose.yml")
        Path(temp_file).unlink()
        logger.info("配置文件传输完成\n")

    def _update_image(self, image: str):
        """后续升级：只更新镜像版本"""
        logger.info(f"更新镜像版本: {image}")
        # 使用 sed 远程修改 docker-compose.yml 的镜像行
        run_ssh_command(
            f"sed -i 's|image:.*|image: {image}|' {self.config.remote_deploy_dir}/docker-compose.yml",
            "更新镜像版本",
        )
        logger.info("镜像版本更新完成\n")

    def _deploy_env(self):
        """传输 .env 配置"""
        logger.info("传输 .env 配置...")
        if not self.config.jwt_secret:
            logger.error("scripts/.env 中缺少 POMELO_ORBIT_JWT__SECRET_KEY")
            sys.exit(1)

        env_content = f"POMELO_ORBIT_JWT__SECRET_KEY={self.config.jwt_secret}\n"
        if self.config.traefik_api_url:
            env_content += (
                f"POMELO_ORBIT_TRAEFIK__API_URL={self.config.traefik_api_url}\n"
            )

        with tempfile.NamedTemporaryFile(mode="w", delete=False, encoding="utf-8") as f:
            f.write(env_content)
            temp_file = f.name

        copy_to_remote(temp_file, f"{self.config.remote_deploy_dir}/.env")
        os.unlink(temp_file)
        logger.info(".env 传输完成\n")

    def _deploy_boot_script(self):
        """传输管理脚本"""
        logger.info("传输管理脚本...")
        boot_script = SCRIPT_DIR / "boot.sh"
        copy_to_remote(str(boot_script), f"{self.config.remote_deploy_dir}/boot.sh")
        run_ssh_command(
            f"chmod +x {self.config.remote_deploy_dir}/boot.sh", "设置执行权限"
        )
        logger.info("管理脚本传输完成\n")

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

    def _pull_image(self):
        """拉取 Docker 镜像"""
        logger.info("拉取 Docker 镜像...")
        image = run_ssh_command(
            f"grep 'image:' {self.config.remote_deploy_dir}/docker-compose.yml | awk '{{print $2}}'",
            "获取镜像名称",
        )
        logger.info(f"  镜像: {image}")
        run_ssh_command(f"docker pull {image}", "拉取镜像")
        logger.info("镜像拉取完成\n")

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

    def _start_service(self):
        """启动服务"""
        logger.info("启动服务...")
        run_ssh_command(
            f"cd {self.config.remote_deploy_dir} && ./boot.sh up", "启动服务"
        )
        logger.info("服务启动命令已执行\n")

        logger.info("等待服务启动...")
        time.sleep(5)

        logger.info("检查服务状态...")
        try:
            run_ssh_command("docker ps | grep -q pomelo-orbit", "检查容器")
        except SystemExit:
            logger.error("部署失败")
            logger.error("请查看日志检查问题")
            sys.exit(1)

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
    run_ssh_command(
        f"tar -czf {remote_archive} -C {remote_dir} .",
        "压缩远程目录",
    )
    logger.info(f"下载备份文件: {local_archive}")
    subprocess.run(
        ["scp", f"{cfg.ssh_target}:{remote_archive}", str(local_archive)],
        check=True,
    )
    run_ssh_command(f"rm -f {remote_archive}", "清理远程临时文件")
    logger.info(f"备份完成: {local_archive}")


def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="Pomelo Orbit 远程部署和管理工具")
    subparsers = parser.add_subparsers(dest="command", help="命令")

    # install 命令
    install_parser = subparsers.add_parser("install", help="初次部署环境")
    install_parser.add_argument("--image", help="指定 Docker 镜像")

    # upgrade 命令
    upgrade_parser = subparsers.add_parser("upgrade", help="更新已有环境")
    upgrade_parser.add_argument("--image", required=True, help="指定 Docker 镜像（必填）")
    upgrade_parser.add_argument(
        "--skip-pull", action="store_true", help="跳过拉取镜像"
    )

    # tunnel 命令
    tunnel_parser = subparsers.add_parser("tunnel", help="SSH 隧道管理")
    tunnel_parser.add_argument(
        "action", choices=["start", "stop", "status"], help="操作"
    )
    tunnel_parser.add_argument(
        "ports", nargs="*", help="端口映射, 格式: remote_port[:local_port], 如 8080 或 8080:8888"
    )

    # exec 命令
    exec_parser = subparsers.add_parser("exec", help="执行远程命令")
    exec_parser.add_argument(
        "remote_command", nargs=argparse.REMAINDER, help="要执行的命令"
    )

    # docker-compose 命令
    dc_parser = subparsers.add_parser("docker-compose", help="执行 docker compose 命令")
    dc_parser.add_argument(
        "dc_args", nargs=argparse.REMAINDER, help="docker compose 参数"
    )

    # ssh 命令
    subparsers.add_parser("ssh", help="SSH 连接到远程服务器")

    # backup 命令
    backup_parser = subparsers.add_parser("backup", help="备份远程数据目录")
    backup_parser.add_argument(
        "--remote-dir",
        help="远程备份目录(默认: REMOTE_DEPLOY_DIR)",
    )

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        sys.exit(1)

    global config
    config = Config()

    if args.command == "install":
        deployer = Deployer(config)
        deployer.install(args.image)
    elif args.command == "upgrade":
        deployer = Deployer(config)
        deployer.upgrade(args.image, args.skip_pull)
    elif args.command == "tunnel":
        tunnel = SSHTunnel(config)
        if args.action == "start":
            # 解析端口映射: remote_port[:local_port], 如 8080 或 8080:8888
            port_mappings = []
            for p in args.ports:
                if ":" in p:
                    parts = p.split(":")
                    if len(parts) != 2 or not parts[0].isdigit() or not parts[1].isdigit():
                        logger.error(f"无效的端口格式: {p}, 应为 remote_port[:local_port]")
                        sys.exit(1)
                    remote, local = int(parts[0]), int(parts[1])
                else:
                    if not p.isdigit():
                        logger.error(f"无效的端口: {p}")
                        sys.exit(1)
                    remote = local = int(p)
                port_mappings.append((remote, local))
            if not port_mappings:
                # 无参数时显示帮助
                tunnel_parser.print_help()
                sys.exit(0)
            else:
                for remote, local in port_mappings:
                    tunnel.start(remote_port=remote, local_port=local)
        elif args.action == "stop":
            tunnel.stop()
        else:
            getattr(tunnel, args.action)()
    elif args.command == "exec":
        executor = RemoteExecutor(config)
        executor.exec(args.remote_command)
    elif args.command == "docker-compose":
        executor = RemoteExecutor(config)
        executor.docker_compose(args.dc_args)
    elif args.command == "ssh":
        os.system(f"ssh {config.ssh_target}")
    elif args.command == "backup":
        backup(config, args.remote_dir or config.remote_deploy_dir)


if __name__ == "__main__":
    main()
