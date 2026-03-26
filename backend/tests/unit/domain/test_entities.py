"""
领域实体测试
"""

from pomelo_orbit.domain.entities import TriggerType, WebhookEventStatus, WebhookEventType, WebhookSource
from pomelo_orbit.domain.value_objects import OperationType


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
        assert app.enabled is True
        assert app.status == "stopped"

    def test_application_with_git_source(self, create_test_application):
        """测试 Application 实体可以关联 Git 源"""
        from pomelo_orbit.domain.entities import GitSource

        git_source = GitSource(
            id="git-1",
            application_id="test-app-1",
            repository_url="https://github.com/test/repo",
            deploy_branches="main,master",
            auto_deploy=True,
        )
        app = create_test_application(git_source=git_source)

        assert app.git_source is not None
        assert app.git_source.repository_url == "https://github.com/test/repo"

    def test_application_with_image_source(self, create_test_application):
        """测试 Application 实体可以关联镜像源"""
        from pomelo_orbit.domain.entities import ImageSource

        image_source = ImageSource(id="img-1", application_id="test-app-1", image_name="nginx:latest")
        app = create_test_application(image_source=image_source)

        assert app.image_source is not None
        assert app.image_source.image_name == "nginx:latest"

    def test_application_with_config_files(self, create_test_application):
        """测试 Application 实体可以关联配置文件"""
        from pomelo_orbit.domain.entities import ApplicationConfigFile

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
        deployment = create_test_deployment(trigger_type=TriggerType.WEBHOOK, operation_type=OperationType.RESTART)

        assert deployment.trigger_type == TriggerType.WEBHOOK
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

    def test_deployment_with_webhook_trigger(self, create_test_deployment):
        """测试 Webhook 触发的部署"""
        deployment = create_test_deployment(
            trigger_type=TriggerType.WEBHOOK, webhook_event_id="webhook-1", trigger_ref="refs/heads/main"
        )

        assert deployment.trigger_type == TriggerType.WEBHOOK
        assert deployment.webhook_event_id == "webhook-1"
        assert deployment.trigger_ref == "refs/heads/main"

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
        from pomelo_orbit.domain.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_https("cert_pem_content", "cert_key_content")

        assert route.https_enabled is True
        assert route.cert_pem == "cert_pem_content"
        assert route.cert_key == "cert_key_content"
        assert route.cert_type == CertType.MANUAL

    def test_route_enable_letsencrypt(self, create_test_route):
        """测试启用 Let's Encrypt 自动证书"""
        from pomelo_orbit.domain.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_letsencrypt()

        assert route.https_enabled is True
        assert route.cert_pem is None
        assert route.cert_key is None
        assert route.cert_type == CertType.LETSENCRYPT

    def test_route_enable_mkcert(self, create_test_route):
        """测试启用 mkcert 本地证书"""
        from pomelo_orbit.domain.entities import CertType

        route = create_test_route(https_enabled=False)
        route.enable_mkcert("mkcert_pem", "mkcert_key")

        assert route.https_enabled is True
        assert route.cert_pem == "mkcert_pem"
        assert route.cert_key == "mkcert_key"
        assert route.cert_type == CertType.MKCERT

    def test_route_disable_https(self, create_test_route):
        """测试禁用 HTTPS"""
        from pomelo_orbit.domain.entities import CertType

        route = create_test_route(https_enabled=True, cert_pem="cert", cert_key="key", cert_type=CertType.MANUAL)
        route.disable_https()

        assert route.https_enabled is False
        assert route.cert_pem is None
        assert route.cert_key is None
        assert route.cert_type == CertType.MANUAL


class TestWebhookEventEntity:
    """WebhookEvent 实体测试"""

    def test_create_webhook_event_with_required_fields(self, create_test_webhook_event):
        """测试创建 WebhookEvent 实体时所有必需字段已正确设置"""
        event = create_test_webhook_event()

        assert event.id == "test-webhook-1"
        assert event.source == WebhookSource.GITHUB
        assert event.event_type == WebhookEventType.PUSH

    def test_create_webhook_event_with_valid_enums(self, create_test_webhook_event):
        """测试创建 WebhookEvent 实体时枚举字段为有效值"""
        event = create_test_webhook_event(
            source=WebhookSource.GITHUB, event_type=WebhookEventType.RELEASE, status=WebhookEventStatus.MATCHED
        )

        assert event.source == WebhookSource.GITHUB
        assert event.event_type == WebhookEventType.RELEASE
        assert event.status == WebhookEventStatus.MATCHED
        assert isinstance(event.source, WebhookSource)
        assert isinstance(event.event_type, WebhookEventType)
        assert isinstance(event.status, WebhookEventStatus)

    def test_webhook_event_default_status(self, create_test_webhook_event):
        """测试 WebhookEvent 默认状态为 RECEIVED"""
        event = create_test_webhook_event()
        assert event.status == WebhookEventStatus.RECEIVED

    def test_webhook_event_with_repository_info(self, create_test_webhook_event):
        """测试 WebhookEvent 包含仓库信息"""
        event = create_test_webhook_event(
            repository_name="owner/repo",
            repository_url="https://github.com/owner/repo",
            branch="develop",
            sender="developer",
        )

        assert event.repository_name == "owner/repo"
        assert event.repository_url == "https://github.com/owner/repo"
        assert event.branch == "develop"
        assert event.sender == "developer"

    def test_webhook_event_with_matched_application(self, create_test_webhook_event):
        """测试 WebhookEvent 匹配到应用"""
        event = create_test_webhook_event(
            status=WebhookEventStatus.MATCHED,
            matched_application_id="app-1",
            triggered_deployment_id="deploy-1",
        )

        assert event.status == WebhookEventStatus.MATCHED
        assert event.matched_application_id == "app-1"
        assert event.triggered_deployment_id == "deploy-1"

    def test_webhook_event_with_error(self, create_test_webhook_event):
        """测试 WebhookEvent 处理错误"""
        event = create_test_webhook_event(status=WebhookEventStatus.ERROR, error_message="Invalid signature")

        assert event.status == WebhookEventStatus.ERROR
        assert event.error_message == "Invalid signature"

    def test_webhook_event_signature_validation(self, create_test_webhook_event):
        """测试 WebhookEvent 签名验证"""
        valid_event = create_test_webhook_event(signature_valid=True)
        invalid_event = create_test_webhook_event(signature_valid=False)

        assert valid_event.signature_valid is True
        assert invalid_event.signature_valid is False

    def test_webhook_event_all_sources(self, create_test_webhook_event):
        """测试所有 Webhook 来源"""
        github_event = create_test_webhook_event(source=WebhookSource.GITHUB)

        assert github_event.source == WebhookSource.GITHUB

    def test_webhook_event_all_types(self, create_test_webhook_event):
        """测试所有事件类型"""
        push_event = create_test_webhook_event(event_type=WebhookEventType.PUSH)
        release_event = create_test_webhook_event(event_type=WebhookEventType.RELEASE)
        ping_event = create_test_webhook_event(event_type=WebhookEventType.PING)

        assert push_event.event_type == WebhookEventType.PUSH
        assert release_event.event_type == WebhookEventType.RELEASE
        assert ping_event.event_type == WebhookEventType.PING


class TestGitSourceEntity:
    """GitSource 实体测试"""

    def test_create_git_source_with_required_fields(self):
        """测试创建 GitSource 实体"""
        from pomelo_orbit.domain.entities import GitSource

        git_source = GitSource(
            id="git-1",
            application_id="app-1",
            repository_url="https://github.com/test/repo",
            deploy_branches="main,master",
            auto_deploy=True,
        )

        assert git_source.id == "git-1"
        assert git_source.application_id == "app-1"
        assert git_source.repository_url == "https://github.com/test/repo"

    def test_git_source_with_custom_values(self):
        """测试 GitSource 自定义值"""
        from pomelo_orbit.domain.entities import GitSource

        git_source = GitSource(
            id="git-1",
            application_id="app-1",
            repository_url="https://github.com/test/repo",
            deploy_branches="develop,staging",
            auto_deploy=False,
        )

        assert git_source.deploy_branches == "develop,staging"
        assert git_source.auto_deploy is False

    def test_git_source_with_custom_branches(self):
        """测试 GitSource 自定义分支"""
        from pomelo_orbit.domain.entities import GitSource

        git_source = GitSource(
            id="git-1",
            application_id="app-1",
            repository_url="https://github.com/test/repo",
            deploy_branches="develop,staging",
            auto_deploy=False,
        )

        assert git_source.deploy_branches == "develop,staging"
        assert git_source.auto_deploy is False


class TestImageSourceEntity:
    """ImageSource 实体测试"""

    def test_create_image_source_with_required_fields(self):
        """测试创建 ImageSource 实体"""
        from pomelo_orbit.domain.entities import ImageSource

        image_source = ImageSource(id="img-1", application_id="app-1", image_name="nginx:latest")

        assert image_source.id == "img-1"
        assert image_source.application_id == "app-1"
        assert image_source.image_name == "nginx:latest"

    def test_image_source_with_registry(self):
        """测试 ImageSource 包含镜像仓库"""
        from pomelo_orbit.domain.entities import ImageSource

        image_source = ImageSource(
            id="img-1",
            application_id="app-1",
            image_name="myapp:v1.0",
            registry_url="https://registry.example.com",
        )

        assert image_source.registry_url == "https://registry.example.com"


class TestApplicationConfigFileEntity:
    """ApplicationConfigFile 实体测试"""

    def test_create_config_file_with_required_fields(self):
        """测试创建 ApplicationConfigFile 实体"""
        from pomelo_orbit.domain.entities import ApplicationConfigFile

        config_file = ApplicationConfigFile(
            id="cfg-1", application_id="app-1", path=".env", content="DATABASE_URL=postgres://localhost"
        )

        assert config_file.id == "cfg-1"
        assert config_file.application_id == "app-1"
        assert config_file.path == ".env"
        assert config_file.content == "DATABASE_URL=postgres://localhost"

    def test_config_file_with_yaml_content(self):
        """测试配置文件包含 YAML 内容"""
        from pomelo_orbit.domain.entities import ApplicationConfigFile

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
        from pomelo_orbit.domain.entities import LoginHistory

        login = LoginHistory(id="login-1", user_id="user-1", username="testuser", success=True)

        assert login.id == "login-1"
        assert login.user_id == "user-1"
        assert login.username == "testuser"
        assert login.success is True

    def test_login_history_with_client_info(self):
        """测试 LoginHistory 包含客户端信息"""
        from pomelo_orbit.domain.entities import LoginHistory

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
        from pomelo_orbit.domain.entities import LoginHistory

        login = LoginHistory(id="login-1", user_id="user-1", username="testuser", success=False)

        assert login.success is False


class TestEnumValues:
    """枚举值测试"""

    def test_source_type_enum(self):
        """测试 SourceType 枚举"""
        from pomelo_orbit.domain.entities import SourceType

        assert SourceType.GIT.value == "git"
        assert SourceType.IMAGE.value == "image"

    def test_trigger_type_enum(self):
        """测试 TriggerType 枚举"""
        from pomelo_orbit.domain.entities import TriggerType

        assert TriggerType.WEBHOOK.value == "webhook"
        assert TriggerType.MANUAL.value == "manual"

    def test_webhook_source_enum(self):
        """测试 WebhookSource 枚举"""
        from pomelo_orbit.domain.entities import WebhookSource

        assert WebhookSource.GITHUB.value == "github"

    def test_webhook_event_type_enum(self):
        """测试 WebhookEventType 枚举"""
        from pomelo_orbit.domain.entities import WebhookEventType

        assert WebhookEventType.PUSH.value == "push"
        assert WebhookEventType.RELEASE.value == "release"
        assert WebhookEventType.PING.value == "ping"

    def test_webhook_event_status_enum(self):
        """测试 WebhookEventStatus 枚举"""
        from pomelo_orbit.domain.entities import WebhookEventStatus

        assert WebhookEventStatus.RECEIVED.value == "received"
        assert WebhookEventStatus.MATCHED.value == "matched"
        assert WebhookEventStatus.IGNORED.value == "ignored"
        assert WebhookEventStatus.ERROR.value == "error"

    def test_cert_type_enum(self):
        """测试 CertType 枚举"""
        from pomelo_orbit.domain.entities import CertType

        assert CertType.MANUAL.value == "manual"
        assert CertType.LETSENCRYPT.value == "letsencrypt"
        assert CertType.MKCERT.value == "mkcert"
