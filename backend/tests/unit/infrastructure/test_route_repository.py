"""RouteRepository 单元测试"""

from pomelo_orbit.domain.entities import Route
from pomelo_orbit.infrastructure.repositories.route import RouteRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestRouteRepository:
    """路由仓储测试"""

    def test_save_and_find_by_id(self, db_session):
        """测试保存路由并通过 ID 查找"""
        repo = RouteRepositoryImpl(db_session)
        route = Route(
            id="route-1",
            name="Route 1",
            domain="app-1.example.com",
            path_prefix="/",
            target_url="http://localhost:3000",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )

        repo.save(route)
        found = repo.find_by_id("route-1")

        assert found is not None
        assert found.id == "route-1"
        assert found.domain == "app-1.example.com"
        assert found.target_url == "http://localhost:3000"

    def test_find_by_id_not_found(self, db_session):
        """测试查找不存在的路由"""
        repo = RouteRepositoryImpl(db_session)
        found = repo.find_by_id("non-existent")
        assert found is None

    def test_find_by_application(self, db_session):
        """测试查询应用的路由"""
        repo = RouteRepositoryImpl(db_session)
        route1 = Route(
            id="route-app1-1",
            name="Route 1",
            domain="app-1.example.com",
            path_prefix="/",
            target_url="http://localhost:3000",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )
        route2 = Route(
            id="route-app1-2",
            name="Route 2",
            domain="api.app-1.example.com",
            path_prefix="/api",
            target_url="http://localhost:3001",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )
        route3 = Route(
            id="route-app2-1",
            name="Route 3",
            domain="app-2.example.com",
            path_prefix="/",
            target_url="http://localhost:4000",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )

        repo.save(route1)
        repo.save(route2)
        repo.save(route3)

        routes = repo.find_by_application("app-1")
        assert len(routes) == 2
        assert all("app-1" in r.domain for r in routes)

    def test_update_route(self, db_session):
        """测试更新路由"""
        repo = RouteRepositoryImpl(db_session)
        route = Route(
            id="route-update",
            name="Old Route",
            domain="old.example.com",
            path_prefix="/",
            target_url="http://localhost:3000",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )
        repo.save(route)

        route.domain = "new.example.com"
        route.target_url = "http://localhost:4000"
        repo.save(route)

        found = repo.find_by_id("route-update")
        assert found is not None
        assert found.domain == "new.example.com"
        assert found.target_url == "http://localhost:4000"

    def test_delete_route(self, db_session):
        """测试删除路由"""
        repo = RouteRepositoryImpl(db_session)
        route = Route(
            id="route-delete",
            name="Delete Route",
            domain="delete.example.com",
            path_prefix="/",
            target_url="http://localhost:3000",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
        )
        repo.save(route)

        repo.delete(route)
        found = repo.find_by_id("route-delete")
        assert found is None

    def test_multiple_routes_for_application(self, db_session):
        """测试一个应用有多个路由"""
        repo = RouteRepositoryImpl(db_session)
        domains = [
            "app-1.example.com",
            "www.app-1.example.com",
            "api.app-1.example.com",
        ]

        for i, domain in enumerate(domains):
            route = Route(
                id=f"route-multi-{i}",
                name=f"Route {i}",
                domain=domain,
                path_prefix="/",
                target_url=f"http://localhost:{3000 + i}",
                enabled=True,
                https_enabled=False,
                created_at=utc_now(),
            )
            repo.save(route)

        routes = repo.find_by_application("app-1")
        assert len(routes) == 3
        found_domains = {r.domain for r in routes}
        assert found_domains == set(domains)
