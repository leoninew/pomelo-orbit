"""
验证 Dishka 容器能用现有 di.py 的函数正确解析 CD 依赖
目标：不修改任何现有函数签名，通过 provider.provide(get_xxx) 注册
"""

from collections.abc import Iterator
from unittest.mock import MagicMock

from dishka import Provider, Scope, make_container
from sqlalchemy.orm import Session

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.repositories import (
    ApplicationRepository,
    ApplicationRouteRepository,
    ApplicationServiceConfigRepository,
    ConfigFileRepository,
    DeploymentRepository,
)
from pomelo_orbit.infrastructure.repositories import (
    ApplicationRepositoryImpl,
    ApplicationRouteRepositoryImpl,
    ApplicationServiceConfigRepositoryImpl,
    ConfigFileRepositoryImpl,
    DeploymentRepositoryImpl,
)

# ── 测试用 Provider：用 lambda/函数直接注册，不修改原始 di.py ──────────────────


def make_test_provider(mock_session: Session) -> Provider:
    """构建测试用 Provider，用 mock 替换外部依赖"""
    provider = Provider()

    def get_session() -> Iterator[Session]:
        yield mock_session

    provider.provide(get_session, scope=Scope.REQUEST, provides=Session)

    provider.provide(ApplicationRepositoryImpl, scope=Scope.REQUEST, provides=ApplicationRepository)
    provider.provide(ApplicationRouteRepositoryImpl, scope=Scope.REQUEST, provides=ApplicationRouteRepository)
    provider.provide(
        ApplicationServiceConfigRepositoryImpl,
        scope=Scope.REQUEST,
        provides=ApplicationServiceConfigRepository,
    )
    provider.provide(ConfigFileRepositoryImpl, scope=Scope.REQUEST, provides=ConfigFileRepository)
    provider.provide(DeploymentRepositoryImpl, scope=Scope.REQUEST, provides=DeploymentRepository)

    def get_mock_manager() -> ApplicationManager:
        return MagicMock(spec=ApplicationManager)

    provider.provide(get_mock_manager, scope=Scope.APP, provides=ApplicationManager)
    provider.provide(ApplicationService, scope=Scope.REQUEST)

    return provider


class TestDishkaCdContainer:
    """验证 Dishka 容器能正确解析 CD 依赖"""

    def test_container_resolves_application_service(self, db_session):
        """容器能解析出 ApplicationService，且依赖正确注入"""
        provider = make_test_provider(db_session)
        container = make_container(provider)

        with container() as request_container:
            svc = request_container.get(ApplicationService)

        assert isinstance(svc, ApplicationService)
        assert isinstance(svc.app_repo, ApplicationRepository)
        assert isinstance(svc.deployment_repo, DeploymentRepository)
        assert isinstance(svc.config_file_repo, ConfigFileRepository)
        assert isinstance(svc.app_route_repo, ApplicationRouteRepository)
        assert isinstance(svc.app_service_config_repo, ApplicationServiceConfigRepository)
        assert isinstance(svc.app_manager, ApplicationManager)

    def test_request_scope_creates_new_session_per_scope(self, db_session):
        """每次进入 REQUEST scope 都得到独立的 service 实例"""
        provider = make_test_provider(db_session)
        container = make_container(provider)

        with container() as req1:
            svc1 = req1.get(ApplicationService)

        with container() as req2:
            svc2 = req2.get(ApplicationService)

        # 不同 scope，不同实例
        assert svc1 is not svc2

    def test_app_scope_manager_is_singleton(self, db_session):
        """APP scope 的 ApplicationManager 在同一容器内是单例"""
        provider = make_test_provider(db_session)
        container = make_container(provider)

        with container() as req1:
            svc1 = req1.get(ApplicationService)

        with container() as req2:
            svc2 = req2.get(ApplicationService)

        # APP scope 的 manager 是同一个实例
        assert svc1.app_manager is svc2.app_manager
