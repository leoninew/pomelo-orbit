"""CI 项目 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import ProjectModel


class TestProjectList:
    def test_returns_paginated_structure(self, auth_client, test_project):
        resp = auth_client.get("/api/ci/projects")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert data["total"] >= 1


class TestProjectCreate:
    def test_creates_project(self, auth_client, test_credential):
        resp = auth_client.post(
            "/api/ci/projects",
            json={
                "name": "new-project",
                "code": "new-project",
                "repository_url": "https://github.com/test/new.git",
                "git_credential_id": test_credential.id,
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-project"
        assert data["code"] == "new-project"
        assert data["repository_url"] == "https://github.com/test/new.git"


class TestProjectGet:
    def test_returns_project(self, auth_client, test_project):
        resp = auth_client.get(f"/api/ci/projects/{test_project.id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == test_project.id

    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/ci/projects/nonexistent-id")
        assert resp.status_code == 404


class TestProjectUpdate:
    def test_updates_name(self, auth_client, test_project):
        resp = auth_client.put(f"/api/ci/projects/{test_project.id}", json={"name": "updated-project"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-project"


class TestProjectDelete:
    def test_deletes_project(self, auth_client, db_session, test_project):
        resp = auth_client.delete(f"/api/ci/projects/{test_project.id}")
        assert resp.status_code == 204
        assert db_session.query(ProjectModel).filter_by(id=test_project.id).first() is None
