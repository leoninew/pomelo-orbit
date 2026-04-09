"""流水线模板 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import PipelineTemplateModel

STAGE_JSON = '[{"name": "build", "image": "alpine:latest", "script": "echo build"}]'
STAGE_PAYLOAD = [{"name": "build", "image": "alpine:latest", "script": "echo build"}]


class TestPipelineTemplateList:
    def test_returns_paginated_structure(self, auth_client, test_template):
        resp = auth_client.get("/api/ci/template")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "per_page" in data

    def test_items_contain_required_fields(self, auth_client, test_template):
        resp = auth_client.get("/api/ci/template")
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
                    variable_declarations="[]",
                )
            )
        db_session.commit()

        resp = auth_client.get("/api/ci/template?page=1&per_page=2")
        assert resp.status_code == 200
        data = resp.json()
        assert len(data["items"]) == 2
        assert data["total"] >= 5


class TestPipelineTemplateCreate:
    def test_creates_template(self, auth_client):
        resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "new-template",
                "description": "desc",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-template"
        assert isinstance(data["variable_declarations"], list)


class TestPipelineTemplateGet:
    def test_returns_template(self, auth_client, test_template):
        resp = auth_client.get(f"/api/ci/template/{test_template.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == test_template.id
        assert isinstance(data["variable_declarations"], list)

    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/ci/template/nonexistent-id")
        assert resp.status_code == 404


class TestPipelineTemplateUpdate:
    def test_updates_name(self, auth_client, test_template):
        resp = auth_client.put(f"/api/ci/template/{test_template.id}", json={"name": "updated-template"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-template"


class TestPipelineTemplateDelete:
    def test_deletes_unreferenced_template(self, auth_client, db_session, test_template):
        resp = auth_client.delete(f"/api/ci/template/{test_template.id}")
        assert resp.status_code == 204
        assert db_session.query(PipelineTemplateModel).filter_by(id=test_template.id).first() is None

    def test_cannot_delete_referenced_template(self, auth_client, test_template, test_project):
        # 创建一个 webhook 引用该模板
        webhook_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook",
            json={
                "name": "test-webhook",
                "template_id": test_template.id,
                "secret": "test-secret",
            },
        )
        assert webhook_resp.status_code == 201

        # 现在模板被 webhook 引用，不能删除
        resp = auth_client.delete(f"/api/ci/template/{test_template.id}")
        assert resp.status_code == 409
