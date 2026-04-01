"""
路由管理 API 集成测试
"""

from unittest.mock import patch

import pytest

from pomelo_orbit.infrastructure.persistence.models import RouteModel


@pytest.fixture
def test_route(db_session):
    """创建测试路由"""
    route = RouteModel(
        name="test-route",
        domain="test.example.com",
        path_prefix="/api",
        target_url="http://backend:8000",
        enabled=True,
        https_enabled=False,
    )
    db_session.add(route)
    db_session.commit()
    db_session.refresh(route)
    return route


@pytest.fixture
def disabled_route(db_session):
    """创建禁用的测试路由"""
    route = RouteModel(
        name="disabled-route",
        domain="disabled.example.com",
        path_prefix="/",
        target_url="http://app:8080",
        enabled=False,
        https_enabled=False,
    )
    db_session.add(route)
    db_session.commit()
    db_session.refresh(route)
    return route


class TestRouteAPI:
    """路由管理 API 测试"""

    @patch("pomelo_orbit.infrastructure.traefik.manager.TraefikManager.deploy_route")
    def test_create_route(self, mock_deploy, auth_client, db_session, test_route):
        """测试创建路由"""
        response = auth_client.post(
            "/api/route",
            json={
                "name": "new-route",
                "domain": "new.example.com",
                "path_prefix": "/",
                "target_url": "http://app:3000",
                "enabled": True,
            },
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "new-route"
        assert data["domain"] == "new.example.com"
        mock_deploy.assert_called_once()

    def test_list_routes(self, auth_client, test_route):
        """测试列出路由"""
        response = auth_client.get("/api/route")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_get_route(self, auth_client, test_route):
        """测试获取路由详情"""
        response = auth_client.get(f"/api/route/{test_route.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_route.id
        assert data["name"] == "test-route"

    @patch("pomelo_orbit.infrastructure.traefik.manager.TraefikManager.revoke_cert")
    @patch("pomelo_orbit.infrastructure.traefik.manager.TraefikManager.deploy_route")
    def test_update_route(self, mock_deploy, mock_revoke_cert, auth_client, db_session, test_route):
        """测试更新路由"""
        response = auth_client.put(
            f"/api/route/{test_route.id}",
            json={"name": "updated-route", "domain": "updated.example.com"},
        )

        assert response.status_code == 200
        assert response.json()["name"] == "updated-route"

        db_session.expire_all()
        route = db_session.query(RouteModel).filter_by(id=test_route.id).first()
        assert route is not None
        assert route.name == "updated-route"

    def test_delete_route(self, auth_client, db_session, disabled_route):
        """测试删除路由"""
        response = auth_client.delete(f"/api/route/{disabled_route.id}")

        assert response.status_code == 204

        db_session.expire_all()
        route = db_session.query(RouteModel).filter_by(id=disabled_route.id).first()
        assert route is None
