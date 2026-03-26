"""Traefik Manager - Infrastructure service for managing Traefik route configurations."""

import logging
import platform
import subprocess
from pathlib import Path

import yaml

from pomelo_orbit.domain.certificate import Certificate
from pomelo_orbit.domain.entities import Route
from pomelo_orbit.domain.route_service import RouteDomainService

logger = logging.getLogger(__name__)


class TraefikManager:
    """Traefik 管理器 - 基础设施服务（无状态）"""

    def parse_cert(self, pem_content: bytes) -> tuple[str, str]:
        """解析合并 PEM，拆分后返回 (cert_pem, cert_key)"""
        return Certificate.parse_pem(pem_content)

    def restore_cert(self, route_name: str, cert_pem: str, cert_key: str, cert_dir: Path) -> None:
        """从数据库内容重建证书文件（用于同步）"""
        self._write_cert_files(route_name, cert_pem.encode(), cert_key.encode(), cert_dir)

    def _write_cert_files(self, route_name: str, cert_pem: bytes, cert_key: bytes, cert_dir: Path) -> None:
        # 按需创建证书目录
        cert_dir.mkdir(parents=True, exist_ok=True)
        (cert_dir / f"{route_name}.pem").write_bytes(cert_pem)
        (cert_dir / f"{route_name}-key.pem").write_bytes(cert_key)

    def revoke_cert(self, route_name: str, cert_dir: Path) -> None:
        """撤销证书文件（cert + key）"""
        for suffix in (".pem", "-key.pem"):
            path = cert_dir / f"{route_name}{suffix}"
            if path.exists():
                path.unlink()

    def _reload_traefik(self, traefik_container: str) -> None:
        """发送 SIGHUP 信号给 Traefik 容器，强制重新加载配置（仅 Windows 需要）"""
        if platform.system() != "Windows":
            return

        try:
            result = subprocess.run(
                ["docker", "ps", "-q", "-f", f"name={traefik_container}"],
                capture_output=True,
                text=True,
                check=True,
            )
            if not result.stdout.strip():
                return

            subprocess.run(
                ["docker", "kill", "--signal=HUP", traefik_container],
                capture_output=True,
                text=True,
                check=True,
            )
            logger.info(f"Traefik reloaded: container={traefik_container}")
        except subprocess.CalledProcessError as e:
            logger.warning(
                f"Traefik reload failed: container={traefik_container}, error={e.stderr.strip() if e.stderr else 'unknown error'}"
            )
        except FileNotFoundError:
            logger.warning("Traefik reload skipped: docker command not found")

    def deploy_route(self, route: Route, config_dir: Path, traefik_container: str) -> None:
        """部署路由配置文件"""
        # 按需创建配置目录
        config_dir.mkdir(parents=True, exist_ok=True)
        config_file = config_dir / f"{route.name}.yml"
        config = RouteDomainService.generate_route_config(route)
        with config_file.open("w", encoding="utf-8") as f:
            yaml.dump(config, f, default_flow_style=False, allow_unicode=True)
        logger.info(f"Route config deployed: route={route.name}, path={config_file}")
        self._reload_traefik(traefik_container)

    def deploy_cert(self, route_name: str, cert_pem: str, cert_key: str, cert_dir: Path) -> None:
        """部署证书文件"""
        self._write_cert_files(route_name, cert_pem.encode(), cert_key.encode(), cert_dir)

    def revoke_route(self, route: Route, config_dir: Path, traefik_container: str) -> None:
        """撤销路由配置文件"""
        config_file = config_dir / f"{route.name}.yml"
        if config_file.exists():
            config_file.unlink()
            logger.info(f"Route config revoked: route={route.name}, path={config_file}")
            self._reload_traefik(traefik_container)
