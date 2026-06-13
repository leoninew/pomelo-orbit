"""Backend CD E2E tests."""

import httpx
from fastapi.testclient import TestClient
from sqlalchemy.orm import Session

from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
from pomelo_orbit.infrastructure.cd.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.traefik import TraefikAPIClient
from tests.e2e.conftest import E2E_PROJECT_ID, auth_headers, seed_user


def test_application_crud_import_export_and_routes(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)

    create_resp = e2e_client.post(
        f"/api/cd/application?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "e2e-app", "code": "e2e-app", "image_pull_policy": "missing"},
    )
    assert create_resp.status_code == 201, create_resp.text
    app = create_resp.json()

    list_resp = e2e_client.get(f"/api/cd/application?project_id={E2E_PROJECT_ID}", headers=headers)
    assert list_resp.status_code == 200, list_resp.text
    assert list_resp.json()["total"] == 1

    update_resp = e2e_client.put(f"/api/cd/application/{app['id']}", headers=headers, json={"route_managed": True})
    assert update_resp.status_code == 200, update_resp.text
    assert update_resp.json()["route_managed"] is True

    route_resp = e2e_client.post(
        f"/api/cd/application/{app['id']}/route",
        headers=headers,
        json={"service_name": "web", "domain": "web.example.test", "port": 8080},
    )
    assert route_resp.status_code == 201, route_resp.text
    app_route = route_resp.json()

    route_list_resp = e2e_client.get(f"/api/cd/application/{app['id']}/route", headers=headers)
    assert route_list_resp.status_code == 200, route_list_resp.text
    assert len(route_list_resp.json()) == 1

    export_resp = e2e_client.get(f"/api/cd/application/{app['id']}/export", headers=headers)
    assert export_resp.status_code == 200, export_resp.text
    exported = export_resp.json()
    assert exported["routes"] == [{"service_name": "web", "domain": "web.example.test", "port": 8080}]

    import_payload = exported | {"name": "imported-app", "code": "imported-app"}
    import_resp = e2e_client.post(
        f"/api/cd/application/import?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json=import_payload,
    )
    assert import_resp.status_code == 201, import_resp.text
    imported_app = import_resp.json()
    imported_routes_resp = e2e_client.get(f"/api/cd/application/{imported_app['id']}/route", headers=headers)
    assert imported_routes_resp.status_code == 200, imported_routes_resp.text
    assert len(imported_routes_resp.json()) == 1

    delete_route_resp = e2e_client.delete(f"/api/cd/application/{app['id']}/route/{app_route['id']}", headers=headers)
    assert delete_route_resp.status_code == 204, delete_route_resp.text

    delete_resp = e2e_client.delete(f"/api/cd/application/{app['id']}", headers=headers)
    assert delete_resp.status_code == 204, delete_resp.text


def test_route_api_side_effect_boundaries(e2e_client: TestClient, e2e_db: Session, monkeypatch) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    monkeypatch.setattr(TraefikManager, "deploy_route", lambda self, route, config_dir, container_name: None)
    monkeypatch.setattr(TraefikManager, "revoke_route", lambda self, route, config_dir, container_name: None)
    monkeypatch.setattr(TraefikManager, "restore_cert", lambda self, name, cert_pem, cert_key, cert_dir: None)
    monkeypatch.setattr(TraefikManager, "deploy_cert", lambda self, name, cert_pem, cert_key, cert_dir: None)
    monkeypatch.setattr(TraefikManager, "revoke_cert", lambda self, name, cert_dir: None)
    monkeypatch.setattr(TraefikManager, "parse_cert", lambda self, content: ("CERT", "KEY"))

    create_resp = e2e_client.post(
        f"/api/cd/route?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "e2e-route", "domain": "app.example.test", "target_url": "http://127.0.0.1:8080"},
    )
    assert create_resp.status_code == 201, create_resp.text
    route_id = create_resp.json()["id"]

    enable_resp = e2e_client.post(f"/api/cd/route/{route_id}/enable", headers=headers)
    assert enable_resp.status_code == 200, enable_resp.text
    cert_resp = e2e_client.post(
        f"/api/cd/route/{route_id}/cert",
        headers=headers,
        files={"pem": ("cert.pem", b"CERT\nKEY", "application/x-pem-file")},
    )
    assert cert_resp.status_code == 200, cert_resp.text
    assert cert_resp.json()["https_enabled"] is True

    disable_https_resp = e2e_client.delete(f"/api/cd/route/{route_id}/https", headers=headers)
    assert disable_https_resp.status_code == 200, disable_https_resp.text
    assert disable_https_resp.json()["https_enabled"] is False

    sync_resp = e2e_client.post(f"/api/cd/route/sync?project_id={E2E_PROJECT_ID}", headers=headers)
    assert sync_resp.status_code == 200, sync_resp.text

    disable_resp = e2e_client.post(f"/api/cd/route/{route_id}/disable", headers=headers)
    assert disable_resp.status_code == 200, disable_resp.text
    delete_resp = e2e_client.delete(f"/api/cd/route/{route_id}", headers=headers)
    assert delete_resp.status_code == 204, delete_resp.text


def test_deployment_stream_log(e2e_client: TestClient, e2e_db: Session, monkeypatch) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    monkeypatch.setattr(ApplicationManagerImpl, "read_deployment_log", lambda self, app_code, deployment_id, offset=0: ("deploy log", 10))

    app_resp = e2e_client.post(
        f"/api/cd/application?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "stream-app", "code": "stream-app", "image_pull_policy": "missing"},
    )
    assert app_resp.status_code == 201, app_resp.text
    app_id = app_resp.json()["id"]
    from pomelo_orbit.infrastructure.persistence.models import DeploymentModel

    deployment = DeploymentModel(
        project_id=E2E_PROJECT_ID,
        application_id=app_id,
        application_name="stream-app",
        operation_type="deploy",
        trigger_type="manual",
        status="ran_to_completion",
        is_rollback=False,
    )
    e2e_db.add(deployment)
    e2e_db.commit()
    e2e_db.refresh(deployment)

    response = e2e_client.get(f"/api/cd/deployment/{deployment.id}/stream-log", headers=headers)

    assert response.status_code == 200, response.text
    assert "deploy log" in response.text
    assert "event: complete" in response.text


def test_traefik_route_api_maps_success_and_connection_error(e2e_client: TestClient, e2e_db: Session, monkeypatch) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    monkeypatch.setattr(TraefikAPIClient, "get_routers", lambda self, api_url: [{"name": "router@docker"}])

    config_resp = e2e_client.get(f"/api/cd/traefik-route/config?project_id={E2E_PROJECT_ID}", headers=headers)
    list_resp = e2e_client.get(f"/api/cd/traefik-route?project_id={E2E_PROJECT_ID}", headers=headers)

    assert config_resp.status_code == 200, config_resp.text
    assert config_resp.json()["dashboard_domain"]
    assert list_resp.status_code == 200, list_resp.text
    assert list_resp.json() == {"items": [{"name": "router@docker"}], "total": 1}

    def raise_connect_error(self, api_url):
        raise httpx.ConnectError("offline")

    monkeypatch.setattr(TraefikAPIClient, "get_routers", raise_connect_error)
    error_resp = e2e_client.get(f"/api/cd/traefik-route?project_id={E2E_PROJECT_ID}", headers=headers)

    assert error_resp.status_code == 503
