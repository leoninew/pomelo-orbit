"""CI 项目 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import RepositoryModel
from pomelo_orbit.infrastructure.persistence.models import ProjectModel
from tests.unit.api.conftest import DEFAULT_CI_PROJECT_ID


class TestProjectList:
    def test_returns_paginated_structure(self, auth_client, test_project):
        resp = auth_client.get(f"/api/ci/repository?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert data["total"] >= 1

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.get("/api/ci/repository?project_id=other-project-id")
        assert resp.status_code == 403


class TestProjectCreate:
    def test_creates_project(self, auth_client, test_credential):
        resp = auth_client.post(
            f"/api/ci/repository?project_id={DEFAULT_CI_PROJECT_ID}",
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

    def test_creates_project_with_variable_overrides(self, auth_client, test_credential):
        resp = auth_client.post(
            f"/api/ci/repository?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "new-project",
                "code": "new-project-vars",
                "repository_url": "https://github.com/test/new.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {"name": "ENV", "value": "production", "description": "Environment"},
                    {"name": "DEBUG", "value": "false"},
                ],
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "new-project"
        assert "variable_declarations" in data

        # Variables is now a flat list, filter by source
        custom_vars_list = [v for v in data["variable_declarations"] if v["source"] == "repository_custom"]
        assert len(custom_vars_list) == 2
        custom_vars = {v["name"]: v["value"] for v in custom_vars_list}
        assert custom_vars == {"ENV": "production", "DEBUG": "false"}


class TestProjectGet:
    def test_returns_project(self, auth_client, test_project):
        resp = auth_client.get(f"/api/ci/repository/{test_project.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == test_project.id
        assert "variable_declarations" in data
        assert isinstance(data["variable_declarations"], list)

    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/ci/repository/nonexistent-id")
        assert resp.status_code == 404


class TestProjectUpdate:
    def test_updates_name(self, auth_client, test_project):
        resp = auth_client.put(f"/api/ci/repository/{test_project.id}", json={"name": "updated-project"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "updated-project"

    def test_preserves_variables_when_updating_basic_info(self, auth_client, test_credential):
        """修改基本信息时应保留原有变量"""
        # 1. 创建带变量的项目
        resp = auth_client.post(
            f"/api/ci/repository?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "test-preserve-vars",
                "code": "test-preserve-vars",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {"name": "VAR1", "value": "value1"},
                    {"name": "VAR2", "value": "value2"},
                ],
            },
        )
        assert resp.status_code == 201
        project_id = resp.json()["id"]

        # 2. 更新基本信息（不传 variable_overrides）
        resp = auth_client.put(
            f"/api/ci/repository/{project_id}",
            json={
                "name": "updated-name",
                "repository_url": "https://github.com/test/updated.git",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["name"] == "updated-name"
        assert data["repository_url"] == "https://github.com/test/updated.git"

        # 3. 验证变量仍然存在
        custom_vars = [v for v in data["variable_declarations"] if v["source"] == "repository_custom"]
        assert len(custom_vars) == 2
        var_dict = {v["name"]: v["value"] for v in custom_vars}
        assert var_dict == {"VAR1": "value1", "VAR2": "value2"}

    def test_can_clear_variables_explicitly(self, auth_client, test_credential):
        """明确传递空列表可以清空变量"""
        # 1. 创建带变量的项目
        resp = auth_client.post(
            f"/api/ci/repository?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "test-clear-vars",
                "code": "test-clear-vars",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {"name": "VAR1", "value": "value1"},
                ],
            },
        )
        assert resp.status_code == 201
        project_id = resp.json()["id"]

        # 2. 明确传递空列表清空变量
        resp = auth_client.put(
            f"/api/ci/repository/{project_id}",
            json={
                "variable_overrides": [],
            },
        )
        assert resp.status_code == 200
        data = resp.json()

        # 3. 验证变量已清空
        custom_vars = [v for v in data["variable_declarations"] if v["source"] == "repository_custom"]
        assert len(custom_vars) == 0


class TestProjectDelete:
    def test_deletes_project(self, auth_client, db_session, test_project):
        resp = auth_client.delete(f"/api/ci/repository/{test_project.id}")
        assert resp.status_code == 204
        assert db_session.query(RepositoryModel).filter_by(id=test_project.id).first() is None
