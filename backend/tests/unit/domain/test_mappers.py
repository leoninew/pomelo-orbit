"""
映射器往返测试
"""

from pomelo_orbit.infrastructure.persistence.mappers import (
    ApplicationMapper,
    DeploymentMapper,
    RouteMapper,
    UserMapper,
    WebhookEventMapper,
)


class TestApplicationMapper:
    """ApplicationMapper 往返测试"""

    def test_application_round_trip(self, create_test_application):
        """验证 Application 实体的往返转换一致性"""
        original = create_test_application()

        # 往返转换
        orm_model = ApplicationMapper.to_orm(original)
        restored = ApplicationMapper.to_domain(orm_model)

        # 验证关键字段保持一致
        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.code == original.code
        assert restored.image_pull_policy == original.image_pull_policy
        assert restored.enabled == original.enabled
        assert restored.status == original.status

    def test_application_round_trip_with_custom_values(self, create_test_application):
        """验证 Application 实体使用自定义值的往返转换一致性"""
        original = create_test_application(
            id="custom-app",
            name="Custom Application",
            code="custom-code",
            enabled=False,
            status="started",
        )

        orm_model = ApplicationMapper.to_orm(original)
        restored = ApplicationMapper.to_domain(orm_model)

        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.code == original.code
        assert restored.enabled == original.enabled
        assert restored.status == original.status


class TestDeploymentMapper:
    """DeploymentMapper 往返测试"""

    def test_deployment_round_trip(self, create_test_deployment):
        """验证 Deployment 实体的往返转换一致性"""
        original = create_test_deployment()

        # 往返转换
        orm_model = DeploymentMapper.to_orm(original)
        restored = DeploymentMapper.to_domain(orm_model)

        # 验证关键字段保持一致
        assert restored.id == original.id
        assert restored.application_id == original.application_id
        assert restored.application_name == original.application_name
        assert restored.trigger_type == original.trigger_type
        assert restored.operation_type == original.operation_type
        assert restored.status == original.status

    def test_deployment_round_trip_preserves_enum_types(self, create_test_deployment):
        """验证 Deployment 实体往返转换保持枚举类型"""
        from pomelo_orbit.domain.entities import TriggerType
        from pomelo_orbit.domain.value_objects import OperationType

        original = create_test_deployment(trigger_type=TriggerType.WEBHOOK, operation_type=OperationType.RESTART)

        orm_model = DeploymentMapper.to_orm(original)
        restored = DeploymentMapper.to_domain(orm_model)

        # 验证枚举类型和值都保持不变
        assert restored.trigger_type == TriggerType.WEBHOOK
        assert restored.operation_type == OperationType.RESTART
        assert isinstance(restored.trigger_type, TriggerType)
        assert isinstance(restored.operation_type, OperationType)


class TestUserMapper:
    """UserMapper 往返测试"""

    def test_user_round_trip(self, create_test_user):
        """验证 User 实体的往返转换一致性"""
        original = create_test_user()

        # 往返转换
        orm_model = UserMapper.to_orm(original)
        restored = UserMapper.to_domain(orm_model)

        # 验证关键字段保持一致
        assert restored.id == original.id
        assert restored.username == original.username
        assert restored.password_hash == original.password_hash

    def test_user_round_trip_with_custom_values(self, create_test_user):
        """验证 User 实体使用自定义值的往返转换一致性"""
        original = create_test_user(id="custom-user", username="customuser", password_hash="custom_hash_value")

        orm_model = UserMapper.to_orm(original)
        restored = UserMapper.to_domain(orm_model)

        assert restored.id == original.id
        assert restored.username == original.username
        assert restored.password_hash == original.password_hash


class TestRouteMapper:
    """RouteMapper 往返测试"""

    def test_route_round_trip(self, create_test_route):
        """验证 Route 实体的往返转换一致性"""
        original = create_test_route()

        # 往返转换
        orm_model = RouteMapper.to_orm(original)
        restored = RouteMapper.to_domain(orm_model)

        # 验证关键字段保持一致
        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.domain == original.domain
        assert restored.path_prefix == original.path_prefix
        assert restored.target_url == original.target_url
        assert restored.enabled == original.enabled

    def test_route_round_trip_with_custom_values(self, create_test_route):
        """验证 Route 实体使用自定义值的往返转换一致性"""
        original = create_test_route(
            domain="custom.example.com", path_prefix="/api/v1", target_url="http://custom-service:9000", enabled=False
        )

        orm_model = RouteMapper.to_orm(original)
        restored = RouteMapper.to_domain(orm_model)

        assert restored.domain == original.domain
        assert restored.path_prefix == original.path_prefix
        assert restored.target_url == original.target_url
        assert restored.enabled == original.enabled

    def test_route_round_trip_with_https_enabled(self, create_test_route):
        """验证 HTTPS 启用的 Route 实体往返转换一致性"""
        from pomelo_orbit.domain.entities import CertType

        original = create_test_route(
            https_enabled=True, cert_pem="cert_content", cert_key="key_content", cert_type=CertType.MANUAL
        )

        orm_model = RouteMapper.to_orm(original)
        restored = RouteMapper.to_domain(orm_model)

        assert restored.https_enabled == original.https_enabled
        assert restored.cert_pem == original.cert_pem
        assert restored.cert_key == original.cert_key
        assert restored.cert_type == original.cert_type

    def test_route_round_trip_preserves_cert_type(self, create_test_route):
        """验证 Route 实体往返转换保持证书类型"""
        from pomelo_orbit.domain.entities import CertType

        manual_route = create_test_route(cert_type=CertType.MANUAL)
        letsencrypt_route = create_test_route(cert_type=CertType.LETSENCRYPT)
        mkcert_route = create_test_route(cert_type=CertType.MKCERT)

        # 验证 manual
        orm_model = RouteMapper.to_orm(manual_route)
        restored = RouteMapper.to_domain(orm_model)
        assert restored.cert_type == CertType.MANUAL
        assert isinstance(restored.cert_type, CertType)

        # 验证 letsencrypt
        orm_model = RouteMapper.to_orm(letsencrypt_route)
        restored = RouteMapper.to_domain(orm_model)
        assert restored.cert_type == CertType.LETSENCRYPT
        assert isinstance(restored.cert_type, CertType)

        # 验证 mkcert
        orm_model = RouteMapper.to_orm(mkcert_route)
        restored = RouteMapper.to_domain(orm_model)
        assert restored.cert_type == CertType.MKCERT
        assert isinstance(restored.cert_type, CertType)


class TestWebhookEventMapper:
    """WebhookEventMapper 往返测试"""

    def test_webhook_event_round_trip(self, create_test_webhook_event):
        """验证 WebhookEvent 实体的往返转换一致性"""
        original = create_test_webhook_event()

        # 往返转换
        orm_model = WebhookEventMapper.to_orm(original)
        restored = WebhookEventMapper.to_domain(orm_model)

        # 验证关键字段保持一致
        assert restored.id == original.id
        assert restored.source == original.source
        assert restored.event_type == original.event_type
        assert restored.status == original.status
        assert restored.repository_name == original.repository_name

    def test_webhook_event_round_trip_preserves_enum_types(self, create_test_webhook_event):
        """验证 WebhookEvent 实体往返转换保持枚举类型"""
        from pomelo_orbit.domain.entities import WebhookEventStatus, WebhookEventType, WebhookSource

        original = create_test_webhook_event(
            source=WebhookSource.GITHUB, event_type=WebhookEventType.RELEASE, status=WebhookEventStatus.MATCHED
        )

        orm_model = WebhookEventMapper.to_orm(original)
        restored = WebhookEventMapper.to_domain(orm_model)

        # 验证枚举类型和值都保持不变
        assert restored.source == WebhookSource.GITHUB
        assert restored.event_type == WebhookEventType.RELEASE
        assert restored.status == WebhookEventStatus.MATCHED
        assert isinstance(restored.source, WebhookSource)
        assert isinstance(restored.event_type, WebhookEventType)
        assert isinstance(restored.status, WebhookEventStatus)
