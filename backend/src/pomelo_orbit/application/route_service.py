"""Route application service - handles route business logic."""

import logging

import ulid
from dynaconf import Dynaconf

from pomelo_orbit.domain.entities import CertType, Route
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.repositories import RouteRepository
from pomelo_orbit.infrastructure.cert.mkcert import MkcertService
from pomelo_orbit.infrastructure.config import get_project_root
from pomelo_orbit.infrastructure.time_utils import utc_now
from pomelo_orbit.infrastructure.traefik import TraefikManager

logger = logging.getLogger(__name__)


class RouteService:
    """路由应用服务 - 处理路由的业务逻辑"""

    def __init__(
        self,
        route_repo: RouteRepository,
        traefik_manager: TraefikManager,
        mkcert_service: MkcertService,
        settings: Dynaconf,
    ):
        self.route_repo = route_repo
        self.traefik_manager = traefik_manager
        self.mkcert_service = mkcert_service
        self.settings = settings

    def _get_traefik_config(self) -> tuple:
        """获取 Traefik 配置路径（config_dir, cert_dir, container_name）"""
        project_root = get_project_root()
        config_dir = project_root / self.settings.traefik.dynamic_route_dir
        cert_dir = project_root / self.settings.traefik.cert_dir
        container_name = self.settings.traefik.container_name
        return config_dir, cert_dir, container_name

    def list_routes(self, page: int, per_page: int) -> tuple[list[Route], int]:
        """列出所有路由（分页）"""
        return self.route_repo.find_paginated(page, per_page)

    def create_route(
        self,
        name: str,
        domain: str,
        path_prefix: str,
        target_url: str,
        enabled: bool,
    ) -> Route:
        """创建路由"""
        route = Route(
            id=str(ulid.ULID()),
            name=name,
            domain=domain,
            path_prefix=path_prefix,
            target_url=target_url,
            enabled=enabled,
            https_enabled=False,
            cert_pem=None,
            cert_key=None,
            created_at=utc_now(),
            updated_at=utc_now(),
        )
        self.route_repo.save(route)

        # 如果启用，写入配置文件
        if route.enabled:
            config_dir, _, container_name = self._get_traefik_config()
            self.traefik_manager.deploy_route(route, config_dir, container_name)

        return route

    def get_route(self, route_id: str) -> Route:
        """获取路由"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)
        return route

    def delete_route(self, route_id: str) -> None:
        """删除路由（要求路由已停用）"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        if route.enabled:
            raise BusinessError("Cannot delete enabled route. Please disable it first.", status_code=400)

        self.route_repo.delete(route)

    def update_route(self, route_id: str, **update_fields) -> Route:
        """更新路由"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        # 更新字段
        for key, value in update_fields.items():
            if hasattr(route, key):
                setattr(route, key, value)

        route.updated_at = utc_now()
        self.route_repo.save(route)

        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 证书
        if route.enabled and route.https_enabled and route.cert_pem and route.cert_key:
            self.traefik_manager.deploy_cert(route.name, route.cert_pem, route.cert_key, cert_dir)
        else:
            self.traefik_manager.revoke_cert(route.name, cert_dir)

        # 路由
        if route.enabled:
            self.traefik_manager.deploy_route(route, config_dir, container_name)
        else:
            self.traefik_manager.revoke_route(route, config_dir, container_name)

        return route

    def enable_route(self, route_id: str) -> None:
        """启用路由"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        route.enable()
        self.route_repo.save(route)

        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 下发证书文件（如果有）
        if route.https_enabled and route.cert_pem and route.cert_key:
            self.traefik_manager.deploy_cert(route.name, route.cert_pem, route.cert_key, cert_dir)

        # 下发路由配置
        self.traefik_manager.deploy_route(route, config_dir, container_name)

    def disable_route(self, route_id: str) -> None:
        """停用路由"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        route.disable()
        self.route_repo.save(route)

        config_dir, _, container_name = self._get_traefik_config()
        self.traefik_manager.revoke_route(route, config_dir, container_name)

    def sync_routes(self) -> None:
        """同步路由配置（含证书文件重建）"""
        routes = self.route_repo.find_all()
        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 生成集中式 TLS 配置（包含所有路由的证书）
        self.traefik_manager.generate_tls_config(routes, config_dir, container_name)

        # 同步路由配置
        for route in routes:
            # 重建证书文件
            if route.https_enabled and route.cert_pem and route.cert_key:
                self.traefik_manager.restore_cert(route.name, route.cert_pem, route.cert_key, cert_dir)

            # 部署或撤销路由配置
            if route.enabled:
                self.traefik_manager.deploy_route(route, config_dir, container_name)
            else:
                self.traefik_manager.revoke_route(route, config_dir, container_name)

    def upload_cert(self, route_id: str, cert_content: bytes) -> Route:
        """上传 SSL 证书（PEM 格式）"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        if not cert_content:
            raise BusinessError("Empty certificate file", status_code=400)

        # 解析证书
        cert_pem, cert_key = self.traefik_manager.parse_cert(cert_content)

        # 启用 HTTPS
        route.enable_https(cert_pem, cert_key)
        self.route_repo.save(route)

        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 下发证书文件
        self.traefik_manager.deploy_cert(route.name, cert_pem, cert_key, cert_dir)

        # 如果路由已启用，重新下发路由配置
        if route.enabled:
            self.traefik_manager.deploy_route(route, config_dir, container_name)

        return route

    def disable_https(self, route_id: str) -> Route:
        """禁用 HTTPS"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 删除证书文件（手动证书和 mkcert 需要，Let's Encrypt 不需要）
        if route.cert_type in (CertType.MANUAL, CertType.MKCERT):
            self.traefik_manager.revoke_cert(route.name, cert_dir)

        route.disable_https()
        self.route_repo.save(route)

        if route.enabled:
            self.traefik_manager.deploy_route(route, config_dir, container_name)

        return route

    def enable_letsencrypt(self, route_id: str) -> Route:
        """启用 Let's Encrypt 自动证书"""
        # 检查 Let's Encrypt 配置
        if not self.settings.letsencrypt.enabled:
            raise BusinessError(
                "Let's Encrypt not enabled. Please set letsencrypt.enabled=true and configure email in config",
                status_code=400,
            )
        if not self.settings.letsencrypt.email:
            raise BusinessError("请在配置中设置 letsencrypt.email", status_code=400)

        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        # 不能重复启用 letsencrypt
        if route.https_enabled and route.cert_type == CertType.LETSENCRYPT:
            raise BusinessError("Let's Encrypt 证书已启用", status_code=400)

        # 验证是否可以使用 Let's Encrypt
        can_use, reason = route.can_use_letsencrypt()
        if not can_use:
            raise BusinessError(reason, status_code=400)

        config_dir, cert_dir, container_name = self._get_traefik_config()

        # 如果之前是手动证书，先删除证书文件
        if route.cert_pem:
            self.traefik_manager.revoke_cert(route.name, cert_dir)

        route.enable_letsencrypt()
        self.route_repo.save(route)

        # 如果路由已启用，重新下发路由配置
        if route.enabled:
            self.traefik_manager.deploy_route(route, config_dir, container_name)

        return route

    def enable_mkcert(self, route_id: str) -> Route:
        """启用 mkcert 本地证书"""
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)

        # 获取证书目录
        project_root = get_project_root()
        cert_dir = project_root / self.settings.traefik.cert_dir

        # 生成证书（内部会检查 mkcert 可用性和 CA 安装状态）
        cert_pem, key_pem = self.mkcert_service.generate_cert(route.domain)

        config_dir, _, container_name = self._get_traefik_config()

        # 如果之前是手动证书，先删除证书文件
        if route.cert_pem:
            self.traefik_manager.revoke_cert(route.name, cert_dir)

        route.enable_mkcert(cert_pem, key_pem)
        self.route_repo.save(route)

        # 下发证书文件
        self.traefik_manager.deploy_cert(route.name, cert_pem, key_pem, cert_dir)

        # 如果路由已启用，重新下发路由配置
        if route.enabled:
            self.traefik_manager.deploy_route(route, config_dir, container_name)

        return route
