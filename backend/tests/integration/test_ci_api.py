"""CI 模块 API 集成测试"""

import pytest

from pomelo_orbit.infrastructure.ci.models import (  # noqa: F401 - 触发 CI 表注册
    CredentialModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)


@pytest.fixture
def test_template(db_session):
    tmpl = PipelineTemplateModel(
        name="test-template",
        description="A test template",
        content="version: v1\nsteps:\n  - name: build\n    image: alpine\n    commands:\n      - echo hello",
        variable_declarations="[]",
        is_builtin=0,
    )
    db_session.add(tmpl)
    db_session.commit()
    db_session.refresh(tmpl)
    return tmpl


@pytest.fixture
def test_credential(db_session):
    cred = CredentialModel(
        name="test-cred",
        type="git_token",
        encrypted_data="encrypted-token",
    )
    db_session.add(cred)
    db_session.commit()
    db_session.refresh(cred)
    return cred


@pytest.fixture
def test_project(db_session, test_template, test_credential):
    project = ProjectModel(
        name="test-project",
        repository_url="https://github.com/test/repo.git",
        pipeline_template_id=test_template.id,
        git_credential_id=test_credential.id,
        variable_overrides="{}",
        webhook_secret="test-secret",
    )
    db_session.add(project)
    db_session.commit()
    db_session.refresh(project)
    return project


# =============================================================================
# Templates
# =============================================================================


class TestTemplateAPI:
    def test_list_templates_returns_paginated(self, auth_client, test_template):
        """GET /templates 返回分页结构"""
        resp = auth_client.get("/api/v1/ci/templates")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert "page" in data
        assert "per_page" in data
        assert data["total"] >= 1
        assert isinstance(data["items"], list)

    def test_list_templates_item_structure(self, auth_client, test_template):
        """列表中每个模板包含必要字段"""
        resp = auth_client.get("/api/v1/ci/templates")
        assert resp.status_code == 200
        item = resp.json()["items"][0]
        assert "id" in item
        assert "name" in item
        assert "variable_declarations" in item
        assert isinstance(item["variable_declarations"], list)

    def test_list_templates_pagination(self, auth_client, db_session):
        """分页参数生效"""
        for i in range(5):
            db_session.add(
                PipelineTemplateModel(
                    name=f"tmpl-{i}",
                    description="",
                    content="version: v1\nsteps: []",
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

    def test_create_template(self, auth_client):
        """创建模板"""
        resp = auth_client.post(
            "/api/v1/ci/templates",
            json={
                "name": "new-template",
                "description": "desc",
                "content": "version: v1\nsteps: []",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-template"
        assert isinstance(data["variable_declarations"], list)

    def test_get_template(self, auth_client, test_template):
        """获取模板详情"""
        resp = auth_client.get(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == test_template.id
        assert isinstance(data["variable_declarations"], list)

    def test_get_template_not_found(self, auth_client):
        resp = auth_client.get("/api/v1/ci/templates/nonexistent-id")
        assert resp.status_code == 404

    def test_update_template(self, auth_client, test_template):
        """更新模板"""
        resp = auth_client.put(
            f"/api/v1/ci/templates/{test_template.id}",
            json={
                "name": "updated-template",
            },
        )
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-template"

    def test_delete_template(self, auth_client, db_session, test_template):
        """删除未被引用的模板"""
        resp = auth_client.delete(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 204
        assert db_session.query(PipelineTemplateModel).filter_by(id=test_template.id).first() is None

    def test_delete_template_referenced_by_project(self, auth_client, test_template, test_project):
        """被项目引用的模板不能删除"""
        resp = auth_client.delete(f"/api/v1/ci/templates/{test_template.id}")
        assert resp.status_code == 409


# =============================================================================
# Credentials
# =============================================================================


class TestCredentialAPI:
    def test_list_credentials(self, auth_client, test_credential):
        resp = auth_client.get("/api/v1/ci/credentials")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert any(c["id"] == test_credential.id for c in data["items"])

    def test_create_credential(self, auth_client):
        resp = auth_client.post(
            "/api/v1/ci/credentials",
            json={
                "name": "my-token",
                "type": "git_token",
                "data": "plaintext-token",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "my-token"
        assert data["type"] == "git_token"

    def test_delete_credential(self, auth_client, db_session, test_credential):
        resp = auth_client.delete(f"/api/v1/ci/credentials/{test_credential.id}")
        assert resp.status_code == 204
        assert db_session.query(CredentialModel).filter_by(id=test_credential.id).first() is None

    def test_delete_credential_referenced_by_project(self, auth_client, test_credential, test_project):
        """被项目引用的凭据不能删除"""
        resp = auth_client.delete(f"/api/v1/ci/credentials/{test_credential.id}")
        assert resp.status_code == 409


# =============================================================================
# Projects
# =============================================================================


class TestProjectAPI:
    def test_list_projects(self, auth_client, test_project):
        resp = auth_client.get("/api/v1/ci/projects")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert data["total"] >= 1

    def test_create_project(self, auth_client, test_template, test_credential):
        resp = auth_client.post(
            "/api/v1/ci/projects",
            json={
                "name": "new-project",
                "repository_url": "https://github.com/test/new.git",
                "pipeline_template_id": test_template.id,
                "git_credential_id": test_credential.id,
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-project"
        # webhook_secret 现在回显（前端 ProjectDetail 需要显示）
        assert "webhook_secret" in data

    def test_get_project(self, auth_client, test_project):
        resp = auth_client.get(f"/api/v1/ci/projects/{test_project.id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == test_project.id

    def test_get_project_not_found(self, auth_client):
        resp = auth_client.get("/api/v1/ci/projects/nonexistent-id")
        assert resp.status_code == 404

    def test_update_project(self, auth_client, test_project):
        resp = auth_client.put(
            f"/api/v1/ci/projects/{test_project.id}",
            json={
                "name": "updated-project",
            },
        )
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-project"

    def test_delete_project(self, auth_client, db_session, test_project):
        resp = auth_client.delete(f"/api/v1/ci/projects/{test_project.id}")
        assert resp.status_code == 204
        assert db_session.query(ProjectModel).filter_by(id=test_project.id).first() is None


# =============================================================================
# Pipeline Runs
# =============================================================================


class TestPipelineRunAPI:
    def test_list_runs_returns_paginated(self, auth_client):
        """GET /runs 返回分页结构"""
        resp = auth_client.get("/api/v1/ci/runs")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data

    def test_list_runs_filter_by_project(self, auth_client, test_project):
        """按项目过滤"""
        resp = auth_client.get(f"/api/v1/ci/runs?project_id={test_project.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data

    def test_list_project_runs(self, auth_client, test_project):
        """GET /projects/{id}/runs 返回分页结构"""
        resp = auth_client.get(f"/api/v1/ci/projects/{test_project.id}/runs")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data

    def test_get_run_not_found(self, auth_client):
        resp = auth_client.get("/api/v1/ci/runs/nonexistent-id")
        assert resp.status_code == 404

    def test_trigger_project(self, auth_client, test_project):
        """POST /projects/{id}/trigger 触发 pipeline"""
        resp = auth_client.post(f"/api/v1/ci/projects/{test_project.id}/trigger", json={})
        # 触发会尝试执行，但测试环境没有 Docker，预期 201 创建 run 记录
        assert resp.status_code == 201
        data = resp.json()
        assert "id" in data
        assert data["project_id"] == test_project.id
        assert data["status"] in ("waiting", "running", "failed")

    def test_trigger_project_not_found(self, auth_client):
        """触发不存在的项目返回 404"""
        resp = auth_client.post("/api/v1/ci/projects/nonexistent/trigger", json={})
        assert resp.status_code == 404

    def test_list_run_jobs(self, auth_client, test_project):
        """GET /runs/{id}/jobs 返回 job 列表"""
        # 先触发一个 run
        trigger_resp = auth_client.post(f"/api/v1/ci/projects/{test_project.id}/trigger", json={})
        assert trigger_resp.status_code == 201
        run_id = trigger_resp.json()["id"]

        resp = auth_client.get(f"/api/v1/ci/runs/{run_id}/jobs")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)

    def test_get_job_logs_not_found(self, auth_client):
        """GET /jobs/{id}/logs 不存在的 job 返回空"""
        resp = auth_client.get("/api/v1/ci/jobs/nonexistent-job/logs")
        assert resp.status_code == 200
        assert resp.json() is None
