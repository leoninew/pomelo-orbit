"""
单元测试共享夹具
"""

import pytest

from pomelo_orbit.domain.cd.entities import Application, CertType, Deployment, Route, TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.domain.shared.entities import User
from pomelo_orbit.infrastructure.time_utils import utc_now


@pytest.fixture
def create_test_application():
    """创建测试用的 Application 实体工厂"""

    def _create(
        id: str = "test-app-1", name: str = "Test Application", code: str = "test-app", **kwargs
    ) -> Application:
        defaults = {
            "image_pull_policy": "IfNotPresent",
            "status": "stopped",
            "created_at": utc_now(),
            "updated_at": utc_now(),
        }
        defaults.update(kwargs)
        return Application(id=id, name=name, code=code, **defaults)  # type: ignore[arg-type]

    return _create


@pytest.fixture
def create_test_deployment():
    """创建测试用的 Deployment 实体工厂"""

    def _create(
        id: str = "test-deploy-1",
        application_id: str = "test-app-1",
        application_name: str = "Test Application",
        **kwargs,
    ) -> Deployment:
        defaults = {
            "trigger_type": TriggerType.MANUAL,
            "status": "queued",
            "operation_type": OperationType.DEPLOY,
            "is_rollback": False,
            "started_at": utc_now(),
        }
        defaults.update(kwargs)
        return Deployment(id=id, application_id=application_id, application_name=application_name, **defaults)  # type: ignore[arg-type]

    return _create


@pytest.fixture
def create_test_user():
    """创建测试用的 User 实体工厂"""

    def _create(
        id: str = "test-user-1", username: str = "testuser", password_hash: str = "hashed_password", **kwargs
    ) -> User:
        defaults = {
            "created_at": utc_now(),
            "updated_at": utc_now(),
            "last_login_at": None,
        }
        defaults.update(kwargs)
        return User(id=id, username=username, password_hash=password_hash, **defaults)  # type: ignore[arg-type]

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
