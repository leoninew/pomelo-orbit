"""
单元测试共享夹具
"""

from datetime import datetime

import pytest

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.entities import Application, CertType, Deployment, Route, TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.infrastructure.time_utils import utc_now


@pytest.fixture
def create_test_application():
    """创建测试用的 Application 实体工厂"""

    def _create(
        id: str = "test-app-1",
        name: str = "Test Application",
        code: str = "test-app",
        project_id: str = "test-project-1",
        image_pull_policy: str = "IfNotPresent",
        status: str = "undeployed",
        created_at: datetime | None = None,
        updated_at: datetime | None = None,
        **kwargs,
    ) -> Application:
        return Application(
            id=id,
            name=name,
            code=code,
            project_id=project_id,
            image_pull_policy=image_pull_policy,
            status=status,
            created_at=created_at or utc_now(),
            updated_at=updated_at or utc_now(),
            **kwargs,
        )

    return _create


@pytest.fixture
def create_test_deployment():
    """创建测试用的 Deployment 实体工厂"""

    def _create(
        id: str = "test-deploy-1",
        project_id: str = "test-project-1",
        application_id: str = "test-app-1",
        application_name: str = "Test Application",
        trigger_type: TriggerType = TriggerType.MANUAL,
        status: str = "queued",
        operation_type: OperationType = OperationType.DEPLOY,
        is_rollback: bool = False,
        started_at: datetime | None = None,
        **kwargs,
    ) -> Deployment:
        return Deployment(
            id=id,
            project_id=project_id,
            application_id=application_id,
            application_name=application_name,
            trigger_type=trigger_type,
            status=status,
            operation_type=operation_type,
            is_rollback=is_rollback,
            started_at=started_at or utc_now(),
            **kwargs,
        )

    return _create


@pytest.fixture
def create_test_user():
    """创建测试用的 User 实体工厂"""

    def _create(
        id: str = "test-user-1",
        username: str = "testuser",
        password_hash: str = "hashed_password",
        status: str = "enabled",
        oauth_provider: str = "",
        oauth_provider_id: str = "",
        email: str | None = None,
        auth_source: str = "password",
        created_at: datetime | None = None,
        updated_at: datetime | None = None,
        last_login_at: datetime | None = None,
    ) -> User:
        return User(
            id=id,
            username=username,
            password_hash=password_hash,
            status=status,
            oauth_provider=oauth_provider,
            oauth_provider_id=oauth_provider_id,
            email=email,
            auth_source=auth_source,
            created_at=created_at or utc_now(),
            updated_at=updated_at or utc_now(),
            last_login_at=last_login_at,
        )

    return _create


@pytest.fixture
def create_test_route():
    """创建测试用的 Route 实体工厂"""

    def _create(
        id: str = "test-route-1",
        name: str = "test-route",
        domain: str = "test.example.com",
        path_prefix: str = "/",
        target_url: str = "http://test-app:80",
        enabled: bool = True,
        https_enabled: bool = False,
        cert_pem: str | None = None,
        cert_key: str | None = None,
        cert_type: CertType = CertType.MANUAL,
        **kwargs,
    ) -> Route:
        defaults = {
            "created_at": utc_now(),
            "updated_at": utc_now(),
        }
        defaults.update(kwargs)
        return Route(
            id=id,
            name=name,
            domain=domain,
            path_prefix=path_prefix,
            target_url=target_url,
            enabled=enabled,
            https_enabled=https_enabled,
            cert_pem=cert_pem,
            cert_key=cert_key,
            cert_type=cert_type,
            **defaults,
        )

    return _create
