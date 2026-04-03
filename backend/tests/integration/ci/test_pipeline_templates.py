"""流水线模板 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import PipelineTemplateModel


class TestPipelineTemplateList:
    def test_returns_paginated_structure(self, auth_client, test_template):
        resp = auth_client.get("/api/v1/ci/templates")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "per_page" in data

    def test_items_contain_required_fields(self, auth_client, test_template):
        resp = auth_client.get("/api/v1/ci/templates")
        item = resp.json()["items"][0]
        assert "id" in item
        assert "name" in item
        assert isinstance(item["variable_declarations"], list)

    def test_pagination_params_respected(self, auth_client, db_session):
        for i in range(5):
            db_session.add(
                PipelineTemplateModel(
                    name=f"tmpl-{i}",
                    description="",
                    stages='[{"name": "build", "type": "checkout", "config": {"ref": "master"}}]',
                    variable_declarations="[]",
                    is_builtin=0,
                )
            )
        db_session.commit()

        resp = auth_client.get("/api/v1/ci/templates?page=1&per_page=2")
        assert resp.status_code == 200
        data = resp.json()
        assert len(data["items"]) == 2
        assert data["total"] >= 5


class TestPipelineTemplateCreate:
    def test_creates_template(self, auth_client):
        resp = auth_client.post(
            "/api/v1/ci/templates",
            json={
                "name": "new-template",
                "description": "desc",
                "stages": [{"name": "build", "type": "checkout", "config": {"ref": "master"}}],
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-template"
        assert isinstance(data["variable_declarations"], list)


class TestPipelineTemplateGet:
    def test_returns_template(self, auth_client, test_template):
        resp = auth_client.get(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == test_template.id
        assert isinstance(data["variable_declarations"], list)

    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/v1/ci/templates/nonexistent-id")
        assert resp.status_code == 404


class TestPipelineTemplateUpdate:
    def test_updates_name(self, auth_client, test_template):
        resp = auth_client.put(f"/api/v1/ci/templates/{test_template.id}", json={"name": "updated-template"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-template"


class TestPipelineTemplateDelete:
    def test_deletes_unreferenced_template(self, auth_client, db_session, test_template):
        resp = auth_client.delete(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 204
        assert db_session.query(PipelineTemplateModel).filter_by(id=test_template.id).first() is None

    def test_cannot_delete_referenced_template(self, auth_client, test_template, test_project):
        resp = auth_client.delete(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 409
