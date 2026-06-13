"""流水线模板 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import PipelineTemplateModel
from pomelo_orbit.infrastructure.persistence.models import ProjectModel
from tests.unit.api.conftest import DEFAULT_CI_PROJECT_ID

STAGE_JSON = '[{"name": "build", "image": "alpine:latest", "script": "echo build"}]'
STAGE_PAYLOAD = [{"name": "build", "image": "alpine:latest", "script": "echo build"}]


class TestPipelineTemplateList:
    def test_returns_paginated_structure(self, auth_client, test_template):
        resp = auth_client.get(f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "per_page" in data

    def test_items_contain_required_fields(self, auth_client, test_template):
        resp = auth_client.get(f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}")
        item = resp.json()["items"][0]
        assert "id" in item
        assert "name" in item
        assert isinstance(item["variable_declarations"], list)

    def test_pagination_params_respected(self, auth_client, db_session):
        for i in range(5):
            db_session.add(
                PipelineTemplateModel(
                    project_id=DEFAULT_CI_PROJECT_ID,
                    name=f"tmpl-{i}",
                    description="",
                    variable_declarations="[]",
                )
            )
        db_session.commit()

        resp = auth_client.get(f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}&page=1&per_page=2")
        assert resp.status_code == 200
        data = resp.json()
        assert len(data["items"]) == 2
        assert data["total"] >= 5

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.get("/api/ci/template?project_id=other-project-id")

        assert resp.status_code == 403


class TestPipelineTemplateCreate:
    def test_creates_template(self, auth_client):
        resp = auth_client.post(
            f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "new-template",
                "description": "desc",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-template"
        assert isinstance(data["variable_declarations"], list)

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.post(
            "/api/ci/template?project_id=other-project-id",
            json={
                "name": "new-template",
                "description": "desc",
            },
        )

        assert resp.status_code == 403


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

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        template = PipelineTemplateModel(
            project_id=project.id,
            name="other-template",
            description="",
            variable_declarations="[]",
        )
        db_session.add(template)
        db_session.commit()

        resp = auth_client.get(f"/api/ci/template/{template.id}")

        assert resp.status_code == 403


class TestPipelineTemplateResolveVariables:
    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.post(
            "/api/ci/template/resolve-variables?project_id=other-project-id",
            json={"orchestration": [], "variable_declarations": []},
        )

        assert resp.status_code == 403


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
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
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
