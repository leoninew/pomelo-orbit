"""
领域实体测试
"""

from pomelo_orbit.domain.cd.entities import TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType


class TestApplicationEntity:
    """Application 实体测试"""

    def test_create_application_with_required_fields(self, create_test_application):
        """测试创建 Application 实体时所有必需字段已正确设置"""
        app = create_test_application()

        assert app.id == "test-app-1"
        assert app.name == "Test Application"
        assert app.code == "test-app"

    def test_create_application_with_custom_fields(self, create_test_application):
        """测试创建 Application 实体时可以自定义字段"""
        app = create_test_application(id="custom-id", name="Custom App", code="custom-code")

        assert app.id == "custom-id"
        assert app.name == "Custom App"
        assert app.code == "custom-code"

    def test_application_default_values(self, create_test_application):
        """测试 Application 实体的默认值"""
        app = create_test_application()

        assert app.image_pull_policy == "IfNotPresent"
        assert app.status == "undeployed"

    def test_application_with_config_files(self, create_test_application):
        """测试 Application 实体可以关联配置文件"""
        from pomelo_orbit.domain.cd.entities import ApplicationConfigFile

        config_files = [
            ApplicationConfigFile(id="cfg-1", application_id="test-app-1", path=".env", content="KEY=value"),
            ApplicationConfigFile(id="cfg-2", application_id="test-app-1", path="config.yml", content="port: 8080"),
        ]
        app = create_test_application(config_files=config_files)

        assert len(app.config_files) == 2
        assert app.config_files[0].path == ".env"
        assert app.config_files[1].path == "config.yml"


class TestDeploymentEntity:
    """Deployment 实体测试"""

    def test_create_deployment_with_required_fields(self, create_test_deployment):
        """测试创建 Deployment 实体时所有必需字段已正确设置"""
        deployment = create_test_deployment()

        assert deployment.id == "test-deploy-1"
        assert deployment.application_id == "test-app-1"
        assert deployment.application_name == "Test Application"

    def test_create_deployment_with_valid_enums(self, create_test_deployment):
        """测试创建 Deployment 实体时枚举字段为有效值"""
        deployment = create_test_deployment(trigger_type=TriggerType.MANUAL, operation_type=OperationType.RESTART)

        assert deployment.trigger_type == TriggerType.MANUAL
        assert deployment.operation_type == OperationType.RESTART
        assert isinstance(deployment.trigger_type, TriggerType)
        assert isinstance(deployment.operation_type, OperationType)

    def test_deployment_default_values(self, create_test_deployment):
        """测试 Deployment 实体的默认值"""
        deployment = create_test_deployment()

        assert deployment.trigger_type == TriggerType.MANUAL
        assert deployment.status == "queued"
        assert deployment.operation_type == OperationType.DEPLOY
        assert deployment.finished_at is None
        assert deployment.duration_ms is None
        assert deployment.is_rollback is False

    def test_deployment_with_rollback_info(self, create_test_deployment):
        """测试回滚部署"""
        deployment = create_test_deployment(is_rollback=True, rollback_from_deployment_id="deploy-old")

        assert deployment.is_rollback is True
        assert deployment.rollback_from_deployment_id == "deploy-old"

    def test_deployment_operation_types(self, create_test_deployment):
        """测试不同的部署操作类型"""
        deploy = create_test_deployment(operation_type=OperationType.DEPLOY)
        stop = create_test_deployment(operation_type=OperationType.STOP)
        restart = create_test_deployment(operation_type=OperationType.RESTART)

        assert deploy.operation_type == OperationType.DEPLOY
        assert stop.operation_type == OperationType.STOP
        assert restart.operation_type == OperationType.RESTART


class TestUserEntity:
    """User 实体测试"""

    def test_create_user_with_required_fields(self, create_test_user):
        """测试创建 User 实体时必需字段非空"""
        user = create_test_user()

        assert user.id == "test-user-1"
        assert user.username
        assert user.password_hash
        assert user.username == "testuser"
        assert user.password_hash == "hashed_password"

    def test_create_user_with_custom_fields(self, create_test_user):
        """测试创建 User 实体时可以自定义字段"""
        user = create_test_user(id="custom-user", username="customuser", password_hash="custom_hash")

        assert user.id == "custom-user"
        assert user.username == "customuser"
        assert user.password_hash == "custom_hash"


class TestRouteEntity:
    """Route 实体测试"""

    def test_create_route_with_required_fields(self, create_test_route):
        """测试创建 Route 实体时所有必需字段已正确设置"""
        route = create_test_route()

        assert route.id == "test-route-1"
        assert route.name == "test-route"
        assert route.domain == "test.example.com"
        assert route.path_prefix == "/"
        assert route.target_url == "http://test-app:80"

    def test_create_route_with_custom_fields(self, create_test_route):
        """测试创建 Route 实体时可以自定义字段"""
        route = create_test_route(domain="custom.example.com", path_prefix="/api", target_url="http://custom-app:8080")

        assert route.domain == "custom.example.com"
        assert route.path_prefix == "/api"
        assert route.target_url == "http://custom-app:8080"

    def test_route_enable(self, create_test_route):
        """测试启用路由"""
        route = create_test_route(enabled=False)
        assert route.enabled is False

        route.enable()
        assert route.enabled is True

    def test_route_disable(self, create_test_route):
        """测试停用路由"""
        route = create_test_route(enabled=True)
        assert route.enabled is True

        route.disable()
        assert route.enabled is False

    def test_route_enable_https_manual(self, create_test_route):
        """测试启用 HTTPS（手动证书）"""
        from pomelo_orbit.domain.cd.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_https("cert_pem_content", "cert_key_content")

        assert route.https_enabled is True
        assert route.cert_pem == "cert_pem_content"
        assert route.cert_key == "cert_key_content"
        assert route.cert_type == CertType.MANUAL

    def test_route_enable_letsencrypt(self, create_test_route):
        """测试启用 Let's Encrypt 自动证书"""
        from pomelo_orbit.domain.cd.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_letsencrypt()

        assert route.https_enabled is True
        assert route.cert_pem is None
        assert route.cert_key is None
        assert route.cert_type == CertType.LETSENCRYPT

    def test_route_enable_mkcert(self, create_test_route):
        """测试启用 mkcert 本地证书"""
        from pomelo_orbit.domain.cd.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_mkcert("mkcert_pem", "mkcert_key")

        assert route.https_enabled is True
        assert route.cert_pem == "mkcert_pem"
        assert route.cert_key == "mkcert_key"
        assert route.cert_type == CertType.MKCERT

    def test_route_disable_https(self, create_test_route):
        """测试禁用 HTTPS"""
        from pomelo_orbit.domain.cd.entities import CertType

        route = create_test_route(https_enabled=True, cert_pem="cert", cert_key="key", cert_type=CertType.MANUAL)
        route.disable_https()

        assert route.https_enabled is False
        assert route.cert_pem is None
        assert route.cert_key is None
        assert route.cert_type == CertType.MANUAL


class TestApplicationConfigFileEntity:
    """ApplicationConfigFile 实体测试"""

    def test_create_config_file_with_required_fields(self):
        """测试创建 ApplicationConfigFile 实体"""
        from pomelo_orbit.domain.cd.entities import ApplicationConfigFile

        config_file = ApplicationConfigFile(
            id="cfg-1", application_id="app-1", path=".env", content="DATABASE_URL=postgres://localhost"
        )

        assert config_file.id == "cfg-1"
        assert config_file.application_id == "app-1"
        assert config_file.path == ".env"
        assert config_file.content == "DATABASE_URL=postgres://localhost"

    def test_config_file_with_yaml_content(self):
        """测试配置文件包含 YAML 内容"""
        from pomelo_orbit.domain.cd.entities import ApplicationConfigFile

        yaml_content = """
server:
  port: 8080
  host: 0.0.0.0
"""
        config_file = ApplicationConfigFile(id="cfg-1", application_id="app-1", path="config.yml", content=yaml_content)

        assert "port: 8080" in config_file.content


class TestLoginHistoryEntity:
    """LoginHistory 实体测试"""

    def test_create_login_history_with_required_fields(self):
        """测试创建 LoginHistory 实体"""
        from pomelo_orbit.domain.auth.entities import LoginHistory

        login = LoginHistory(id="login-1", user_id="user-1", username="testuser", success=True)

        assert login.id == "login-1"
        assert login.user_id == "user-1"
        assert login.username == "testuser"
        assert login.success is True

    def test_login_history_with_client_info(self):
        """测试 LoginHistory 包含客户端信息"""
        from pomelo_orbit.domain.auth.entities import LoginHistory

        login = LoginHistory(
            id="login-1",
            user_id="user-1",
            username="testuser",
            success=True,
            ip_address="192.168.1.100",
            user_agent="Mozilla/5.0",
        )

        assert login.ip_address == "192.168.1.100"
        assert login.user_agent == "Mozilla/5.0"

    def test_login_history_failed_login(self):
        """测试失败的登录记录"""
        from pomelo_orbit.domain.auth.entities import LoginHistory

        login = LoginHistory(id="login-1", user_id="user-1", username="testuser", success=False)

        assert login.success is False


class TestEnumValues:
    """枚举值测试"""

    def test_trigger_type_enum(self):
        """测试 TriggerType 枚举"""
        from pomelo_orbit.domain.cd.entities import TriggerType

        assert TriggerType.MANUAL.value == "manual"

    def test_cert_type_enum(self):
        """测试 CertType 枚举"""
        from pomelo_orbit.domain.cd.entities import CertType

        assert CertType.MANUAL.value == "manual"
        assert CertType.LETSENCRYPT.value == "letsencrypt"
        assert CertType.MKCERT.value == "mkcert"
