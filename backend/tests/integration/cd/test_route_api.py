"""
路由管理 API 集成测试
"""

from unittest.mock import patch

from pomelo_orbit.infrastructure.persistence.models import ProjectModel, RouteModel
from tests.integration.conftest import DEFAULT_CI_PROJECT_ID


class TestRouteAPI:
    """路由管理 API 测试"""

    @patch("pomelo_orbit.infrastructure.cd.traefik.manager.TraefikManager.deploy_route")
    def test_create_route(self, mock_deploy, auth_client, db_session, test_route):
        """测试创建路由"""
        response = auth_client.post(
            f"/api/cd/route?project_id={DEFAULT_CI_PROJECT_ID}",
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
        response = auth_client.get(f"/api/cd/route?project_id={DEFAULT_CI_PROJECT_ID}")

        assert response.status_code == 200
        data = response.json()
        assert data["total"] >= 1
        assert len(data["items"]) >= 1

    def test_list_routes_rejects_non_member_project(self, auth_client, db_session):
        """非项目成员不能列出路由"""
        db_session.add(ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True))
        db_session.commit()

        response = auth_client.get("/api/cd/route?project_id=other-project-id")

        assert response.status_code == 403
        assert response.json()["detail"] == "Permission denied"

    def test_get_route(self, auth_client, test_route):
        """测试获取路由详情"""
        response = auth_client.get(f"/api/cd/route/{test_route.id}")

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_route.id
        assert data["name"] == "test-route"

    def test_get_route_rejects_non_member_project(self, auth_client, db_session):
        """非项目成员不能查看其他项目路由"""
        db_session.add(ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True))
        route = RouteModel(
            project_id="other-project-id",
            name="other-route",
            domain="other.example.com",
            path_prefix="/",
            target_url="http://other:8080",
            enabled=False,
            https_enabled=False,
        )
        db_session.add(route)
        db_session.commit()
        db_session.refresh(route)

        response = auth_client.get(f"/api/cd/route/{route.id}")

        assert response.status_code == 403
        assert response.json()["detail"] == "Permission denied"

    @patch("pomelo_orbit.infrastructure.cd.traefik.manager.TraefikManager.revoke_cert")
    @patch("pomelo_orbit.infrastructure.cd.traefik.manager.TraefikManager.deploy_route")
    def test_update_route(self, mock_deploy, mock_revoke_cert, auth_client, db_session, test_route):
        """测试更新路由"""
        response = auth_client.put(
            f"/api/cd/route/{test_route.id}",
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
        response = auth_client.delete(f"/api/cd/route/{disabled_route.id}")

        assert response.status_code == 204

        db_session.expire_all()
        route = db_session.query(RouteModel).filter_by(id=disabled_route.id).first()
        assert route is None
