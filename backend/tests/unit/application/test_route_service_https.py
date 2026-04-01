"""Route Service HTTPS 功能单元测试"""

from unittest.mock import Mock

import pytest
from ulid import ULID

from pomelo_orbit.application.route_service import RouteService
from pomelo_orbit.domain.entities import CertType, Route
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.time_utils import utc_now


def create_route(name="test-route", domain="test.example.com", enabled=True, **kwargs):
    """辅助函数：创建 Route 实例"""
    return Route(
        id=str(ULID()),
        name=name,
        domain=domain,
        path_prefix=kwargs.get("path_prefix", "/"),
        target_url=kwargs.get("target_url", "http://app:8000"),
        enabled=enabled,
        https_enabled=kwargs.get("https_enabled", False),
        cert_pem=kwargs.get("cert_pem"),
        cert_key=kwargs.get("cert_key"),
        cert_type=kwargs.get("cert_type", CertType.MANUAL),
        created_at=utc_now(),
        updated_at=utc_now(),
    )


class TestUploadCert:
    """测试 upload_cert"""

    def test_upload_cert_route_not_found(self):
        """测试上传证书时路由不存在"""
        route_repo = Mock()
        route_repo.find_by_id.return_value = None

        service = RouteService(route_repo, Mock(), Mock(), Mock())

        with pytest.raises(BusinessError, match="not found"):
            service.upload_cert("route-1", b"cert_content")

    def test_upload_cert_empty_content(self):
        """测试上传空证书"""
        route_repo = Mock()
        route_repo.find_by_id.return_value = create_route()

        service = RouteService(route_repo, Mock(), Mock(), Mock())

        with pytest.raises(BusinessError, match="Empty certificate"):
            service.upload_cert("route-1", b"")


class TestDisableHttps:
    """测试 disable_https"""

    def test_disable_https_route_not_found(self):
        """测试禁用 HTTPS 时路由不存在"""
        route_repo = Mock()
        route_repo.find_by_id.return_value = None

        service = RouteService(route_repo, Mock(), Mock(), Mock())

        with pytest.raises(BusinessError, match="not found"):
            service.disable_https("route-1")


class TestEnableLetsencrypt:
    """测试 enable_letsencrypt"""

    def test_enable_letsencrypt_not_enabled_in_config(self):
        """测试 Let's Encrypt 未在配置中启用"""
        settings = Mock()
        settings.letsencrypt.enabled = False

        service = RouteService(Mock(), Mock(), Mock(), settings)

        with pytest.raises(BusinessError, match="not enabled"):
            service.enable_letsencrypt("route-1")

    def test_enable_letsencrypt_no_email(self):
        """测试 Let's Encrypt 未配置邮箱"""
        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = ""

        service = RouteService(Mock(), Mock(), Mock(), settings)

        with pytest.raises(BusinessError, match="email"):
            service.enable_letsencrypt("route-1")

    def test_enable_letsencrypt_route_not_found(self):
        """测试启用 Let's Encrypt 时路由不存在"""
        route_repo = Mock()
        route_repo.find_by_id.return_value = None
        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = "admin@example.com"

        service = RouteService(route_repo, Mock(), Mock(), settings)

        with pytest.raises(BusinessError, match="not found"):
            service.enable_letsencrypt("route-1")


class TestEnableMkcert:
    """测试 enable_mkcert"""

    def test_enable_mkcert_route_not_found(self):
        """测试启用 mkcert 时路由不存在"""
        route_repo = Mock()
        route_repo.find_by_id.return_value = None

        service = RouteService(route_repo, Mock(), Mock(), Mock())

        with pytest.raises(BusinessError, match="not found"):
            service.enable_mkcert("route-1")

    def test_enable_mkcert_success_enabled_route(self):
        """测试成功启用 mkcert（路由已启用）"""
        route = create_route(enabled=True)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        mkcert_service = Mock()
        mkcert_service.generate_cert.return_value = ("cert_pem", "key_pem")

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, mkcert_service, settings)
        result = service.enable_mkcert(route.id)

        # 验证证书生成
        mkcert_service.generate_cert.assert_called_once_with(route.domain)

        # 验证路由更新
        assert result.https_enabled is True
        assert result.cert_type == CertType.MKCERT
        assert result.cert_pem == "cert_pem"
        assert result.cert_key == "key_pem"
        route_repo.save.assert_called_once()

        # 验证证书部署
        traefik_manager.deploy_cert.assert_called_once()

        # 验证路由配置重新部署
        traefik_manager.deploy_route.assert_called_once()

    def test_enable_mkcert_success_disabled_route(self):
        """测试成功启用 mkcert（路由未启用）"""
        route = create_route(enabled=False)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        mkcert_service = Mock()
        mkcert_service.generate_cert.return_value = ("cert_pem", "key_pem")

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, mkcert_service, settings)
        result = service.enable_mkcert(route.id)

        # 验证证书生成
        mkcert_service.generate_cert.assert_called_once()

        # 验证路由更新
        assert result.https_enabled is True
        route_repo.save.assert_called_once()

        # 验证证书部署
        traefik_manager.deploy_cert.assert_called_once()

        # 路由未启用，不应重新部署路由配置
        traefik_manager.deploy_route.assert_not_called()

    def test_enable_mkcert_replaces_existing_cert(self):
        """测试启用 mkcert 替换现有证书"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_pem="old_cert",
            cert_key="old_key",
            cert_type=CertType.MANUAL,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        mkcert_service = Mock()
        mkcert_service.generate_cert.return_value = ("new_cert", "new_key")

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, mkcert_service, settings)
        result = service.enable_mkcert(route.id)

        # 验证旧证书被删除
        traefik_manager.revoke_cert.assert_called_once()

        # 验证新证书部署
        traefik_manager.deploy_cert.assert_called_once()

        # 验证路由更新
        assert result.cert_pem == "new_cert"
        assert result.cert_key == "new_key"
        assert result.cert_type == CertType.MKCERT


class TestUploadCertSuccess:
    """测试 upload_cert 成功场景"""

    def test_upload_cert_success_enabled_route(self):
        """测试成功上传证书（路由已启用）"""
        route = create_route(enabled=True)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        traefik_manager.parse_cert.return_value = ("cert_pem", "key_pem")

        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.upload_cert(route.id, b"cert_content")

        # 验证证书解析
        traefik_manager.parse_cert.assert_called_once_with(b"cert_content")

        # 验证路由更新
        assert result.https_enabled is True
        assert result.cert_pem == "cert_pem"
        assert result.cert_key == "key_pem"
        route_repo.save.assert_called_once()

        # 验证证书部署
        traefik_manager.deploy_cert.assert_called_once()

        # 验证路由配置重新部署
        traefik_manager.deploy_route.assert_called_once()

    def test_upload_cert_success_disabled_route(self):
        """测试成功上传证书（路由未启用）"""
        route = create_route(enabled=False)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        traefik_manager.parse_cert.return_value = ("cert_pem", "key_pem")

        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        service.upload_cert(route.id, b"cert_content")

        # 验证证书部署
        traefik_manager.deploy_cert.assert_called_once()

        # 路由未启用，不应重新部署路由配置
        traefik_manager.deploy_route.assert_not_called()


class TestDisableHttpsSuccess:
    """测试 disable_https 成功场景"""

    def test_disable_https_manual_cert_enabled_route(self):
        """测试禁用 HTTPS（手动证书，路由已启用）"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_pem="cert",
            cert_key="key",
            cert_type=CertType.MANUAL,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.disable_https(route.id)

        # 验证证书文件被删除
        traefik_manager.revoke_cert.assert_called_once()

        # 验证路由更新
        assert result.https_enabled is False
        route_repo.save.assert_called_once()

        # 验证路由配置重新部署
        traefik_manager.deploy_route.assert_called_once()

    def test_disable_https_mkcert_enabled_route(self):
        """测试禁用 HTTPS（mkcert 证书，路由已启用）"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_pem="cert",
            cert_key="key",
            cert_type=CertType.MKCERT,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.disable_https(route.id)

        # 验证证书文件被删除
        traefik_manager.revoke_cert.assert_called_once()

        # 验证路由更新
        assert result.https_enabled is False

    def test_disable_https_letsencrypt_enabled_route(self):
        """测试禁用 HTTPS（Let's Encrypt 证书，路由已启用）"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_type=CertType.LETSENCRYPT,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.disable_https(route.id)

        # Let's Encrypt 证书不需要删除文件
        traefik_manager.revoke_cert.assert_not_called()

        # 验证路由更新
        assert result.https_enabled is False

        # 验证路由配置重新部署
        traefik_manager.deploy_route.assert_called_once()

    def test_disable_https_disabled_route(self):
        """测试禁用 HTTPS（路由未启用）"""
        route = create_route(
            enabled=False,
            https_enabled=True,
            cert_pem="cert",
            cert_key="key",
            cert_type=CertType.MANUAL,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        service.disable_https(route.id)

        # 验证证书文件被删除
        traefik_manager.revoke_cert.assert_called_once()

        # 路由未启用，不应重新部署路由配置
        traefik_manager.deploy_route.assert_not_called()


class TestEnableLetsencryptSuccess:
    """测试 enable_letsencrypt 成功场景"""

    def test_enable_letsencrypt_success_enabled_route(self):
        """测试成功启用 Let's Encrypt（路由已启用）"""
        route = create_route(enabled=True)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = "admin@example.com"
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.enable_letsencrypt(route.id)

        # 验证路由更新
        assert result.https_enabled is True
        assert result.cert_type == CertType.LETSENCRYPT
        route_repo.save.assert_called_once()

        # 验证路由配置重新部署
        traefik_manager.deploy_route.assert_called_once()

    def test_enable_letsencrypt_success_disabled_route(self):
        """测试成功启用 Let's Encrypt（路由未启用）"""
        route = create_route(enabled=False)
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = "admin@example.com"
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.enable_letsencrypt(route.id)

        # 验证路由更新
        assert result.https_enabled is True
        route_repo.save.assert_called_once()

        # 路由未启用，不应重新部署路由配置
        traefik_manager.deploy_route.assert_not_called()

    def test_enable_letsencrypt_replaces_manual_cert(self):
        """测试启用 Let's Encrypt 替换手动证书"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_pem="old_cert",
            cert_key="old_key",
            cert_type=CertType.MANUAL,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        traefik_manager = Mock()
        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = "admin@example.com"
        settings.traefik.dynamic_route_dir = "config"
        settings.traefik.cert_dir = "certs"
        settings.traefik.container_name = "traefik"

        service = RouteService(route_repo, traefik_manager, Mock(), settings)
        result = service.enable_letsencrypt(route.id)

        # 验证旧证书被删除
        traefik_manager.revoke_cert.assert_called_once()

        # 验证路由更新
        assert result.cert_type == CertType.LETSENCRYPT
        assert result.cert_pem is None
        assert result.cert_key is None

    def test_enable_letsencrypt_already_enabled(self):
        """测试重复启用 Let's Encrypt"""
        route = create_route(
            enabled=True,
            https_enabled=True,
            cert_type=CertType.LETSENCRYPT,
        )
        route_repo = Mock()
        route_repo.find_by_id.return_value = route

        settings = Mock()
        settings.letsencrypt.enabled = True
        settings.letsencrypt.email = "admin@example.com"

        service = RouteService(route_repo, Mock(), Mock(), settings)

        with pytest.raises(BusinessError, match="已启用"):
            service.enable_letsencrypt(route.id)
