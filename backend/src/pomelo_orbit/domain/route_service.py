"""Route domain service - pure business logic without infrastructure dependencies."""

from typing import Any

from pomelo_orbit.domain.entities import CertType, Route


class RouteDomainService:
    """路由领域服务 - 纯业务逻辑（不依赖基础设施）"""

    @staticmethod
    def generate_route_config(route: Route) -> dict[str, Any]:
        """为单个路由生成 Traefik 配置（纯函数）"""
        service_name = f"{route.name}-service"

        rule = f"Host(`{route.domain}`)"
        if route.path_prefix != "/":
            rule += f" && PathPrefix(`{route.path_prefix}`)"

        config: dict[str, Any] = {
            "http": {
                "services": {service_name: {"loadBalancer": {"servers": [{"url": route.target_url}]}}},
            }
        }

        # HTTPS 模式
        if route.https_enabled:
            https_router_name = f"{route.name}-route"

            if route.cert_type == CertType.LETSENCRYPT:
                # Let's Encrypt 自动证书模式
                https_router_config: dict[str, Any] = {
                    "rule": rule,
                    "service": service_name,
                    "entryPoints": ["websecure"],
                    "tls": {"certResolver": "letsencrypt"},
                }
                config["http"]["routers"] = {https_router_name: https_router_config}
                # 无需 tls.certificates，Traefik 自动从 Let's Encrypt 获取
            elif route.cert_pem:
                # 手动证书或 mkcert 证书模式（证书文件已写入 certs 目录）
                https_router_config = {
                    "rule": rule,
                    "service": service_name,
                    "entryPoints": ["websecure"],
                    "tls": {},
                }
                config["http"]["routers"] = {https_router_name: https_router_config}
                config["tls"] = {
                    "certificates": [
                        {
                            "certFile": f"/etc/traefik/certs/{route.name}.pem",
                            "keyFile": f"/etc/traefik/certs/{route.name}-key.pem",
                        }
                    ]
                }
        else:
            # HTTP 模式
            http_router_name = f"{route.name}-route"
            http_router_config: dict[str, Any] = {
                "rule": rule,
                "service": service_name,
                "entryPoints": ["web"],
            }
            config["http"]["routers"] = {http_router_name: http_router_config}

        return config
