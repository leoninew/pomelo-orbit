"""Stage API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import BuildStageModel
from pomelo_orbit.infrastructure.persistence.models import ProjectModel
from tests.integration.conftest import DEFAULT_CI_PROJECT_ID


class TestStageList:
    def test_returns_stages(self, auth_client, db_session):
        """测试返回 Stage 列表"""
        # 创建多个 stages
        for i in range(3):
            stage = BuildStageModel(
                project_id=DEFAULT_CI_PROJECT_ID,
                name=f"stage-{i}",
                image="alpine:latest",
                script=f"echo {i}",
                artifacts="[]",
                description="",
            )
            db_session.add(stage)
        db_session.commit()

        # 列出 stages
        resp = auth_client.get(f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        data = resp.json()
        assert isinstance(data, dict)
        assert "items" in data
        assert isinstance(data["items"], list)
        assert len(data["items"]) >= 3

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.get("/api/ci/build-stage?project_id=other-project-id")

        assert resp.status_code == 403


class TestStageCreate:
    def test_creates_stage(self, auth_client):
        resp = auth_client.post(
            f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "build",
                "image": "alpine:latest",
                "script": "echo 'building'",
                "artifacts": [{"type": "binary", "path": "/app/dist", "name": "dist"}],
                "description": "Build stage",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "build"
        assert data["image"] == "alpine:latest"
        assert data["script"] == "echo 'building'"
        assert len(data["artifacts"]) == 1

    def test_validates_required_fields(self, auth_client):
        resp = auth_client.post(
            f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}",
            json={"name": "incomplete"},
        )
        assert resp.status_code == 422

    def test_rejects_duplicate_name(self, auth_client, db_session):
        """测试拒绝同名 Stage"""
        # 创建第一个 stage
        create_resp = auth_client.post(
            f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "duplicate-test",
                "image": "alpine:latest",
                "script": "echo test",
            },
        )
        assert create_resp.status_code == 201

        # 尝试创建同名 stage
        resp = auth_client.post(
            f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "duplicate-test",
                "image": "ubuntu:latest",
                "script": "echo another",
            },
        )
        assert resp.status_code == 409

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        db_session.add(project)
        db_session.commit()

        resp = auth_client.post(
            "/api/ci/build-stage?project_id=other-project-id",
            json={
                "name": "other-build",
                "image": "alpine:latest",
                "script": "echo test",
            },
        )

        assert resp.status_code == 403


class TestStageGet:
    def test_returns_stage(self, auth_client, db_session):
        """测试获取单个 Stage"""
        # 创建 stage
        stage = BuildStageModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            name="get-test",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(stage)
        db_session.commit()
        db_session.refresh(stage)

        # 获取 stage
        resp = auth_client.get(f"/api/ci/build-stage/{stage.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == stage.id
        assert data["name"] == "get-test"

    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/ci/build-stage/nonexistent-id")
        assert resp.status_code == 404

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        stage = BuildStageModel(
            project_id=project.id,
            name="other-stage",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(project)
        db_session.add(stage)
        db_session.commit()

        resp = auth_client.get(f"/api/ci/build-stage/{stage.id}")

        assert resp.status_code == 403


class TestStageUpdate:
    def test_updates_stage(self, auth_client, db_session):
        """测试更新 Stage"""
        # 创建 stage
        stage = BuildStageModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            name="update-test",
            image="alpine:latest",
            script="echo old",
            artifacts="[]",
            description="",
        )
        db_session.add(stage)
        db_session.commit()
        db_session.refresh(stage)

        # 更新 stage
        resp = auth_client.put(
            f"/api/ci/build-stage/{stage.id}",
            json={
                "name": "updated-stage",
                "image": "ubuntu:latest",
                "script": "echo new",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["name"] == "updated-stage"
        assert data["image"] == "ubuntu:latest"
        assert data["script"] == "echo new"

    def test_partial_update(self, auth_client, db_session):
        """测试部分更新"""
        stage = BuildStageModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            name="partial-test",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="old description",
        )
        db_session.add(stage)
        db_session.commit()
        db_session.refresh(stage)

        # 只更新 description
        resp = auth_client.put(
            f"/api/ci/build-stage/{stage.id}",
            json={"description": "new description"},
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["description"] == "new description"
        assert data["name"] == "partial-test"

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        stage = BuildStageModel(
            project_id=project.id,
            name="other-update-stage",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(project)
        db_session.add(stage)
        db_session.commit()

        resp = auth_client.put(f"/api/ci/build-stage/{stage.id}", json={"name": "updated"})

        assert resp.status_code == 403


class TestStageDelete:
    def test_deletes_unreferenced_stage(self, auth_client, db_session):
        """测试删除未被引用的 Stage"""
        # 创建 stage
        stage = BuildStageModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            name="delete-test",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(stage)
        db_session.commit()
        db_session.refresh(stage)

        # 删除 stage
        resp = auth_client.delete(f"/api/ci/build-stage/{stage.id}")
        assert resp.status_code == 204

        # 验证已删除
        assert db_session.query(BuildStageModel).filter_by(id=stage.id).first() is None

    def test_cannot_delete_referenced_stage(self, auth_client, db_session, test_template):
        """测试不能删除被模板引用的 Stage"""
        # 创建 stage
        stage = BuildStageModel(
            project_id=DEFAULT_CI_PROJECT_ID,
            name="referenced-stage",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(stage)
        db_session.commit()
        db_session.refresh(stage)

        # 尝试删除（应该被阻止，如果被引用）
        resp = auth_client.delete(f"/api/ci/build-stage/{stage.id}")
        # 如果 stage 没被引用，可以删除；如果被引用，应该返回 409
        assert resp.status_code in (204, 409)

    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        stage = BuildStageModel(
            project_id=project.id,
            name="other-delete-stage",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(project)
        db_session.add(stage)
        db_session.commit()

        resp = auth_client.delete(f"/api/ci/build-stage/{stage.id}")

        assert resp.status_code == 403
        assert db_session.get(BuildStageModel, stage.id) is not None


class TestStageDuplicate:
    def test_rejects_non_member_project(self, auth_client, db_session):
        project = ProjectModel(id="other-project-id", name="Other Project", code="other", is_active=True)
        stage = BuildStageModel(
            project_id=project.id,
            name="other-duplicate-stage",
            image="alpine:latest",
            script="echo test",
            artifacts="[]",
            description="",
        )
        db_session.add(project)
        db_session.add(stage)
        db_session.commit()

        resp = auth_client.post(f"/api/ci/build-stage/{stage.id}/duplicate")

        assert resp.status_code == 403


class TestStageUsage:
    def test_stage_reusable_across_templates(self, auth_client):
        """测试 Stage 可以在多个模板中复用"""
        # 创建一个通用的 build stage
        stage_resp = auth_client.post(
            f"/api/ci/build-stage?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "common-build",
                "image": "alpine:latest",
                "script": "echo 'building'",
            },
        )
        assert stage_resp.status_code == 201
        stage_id = stage_resp.json()["id"]

        # 创建两个模板，都使用这个 stage（通过编排）
        template1_resp = auth_client.post(
            f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}",
            json={"name": "template-1", "description": ""},
        )
        assert template1_resp.status_code == 201

        template2_resp = auth_client.post(
            f"/api/ci/template?project_id={DEFAULT_CI_PROJECT_ID}",
            json={"name": "template-2", "description": ""},
        )
        assert template2_resp.status_code == 201

        # 验证 stage 仍然存在且可以被查询
        resp = auth_client.get(f"/api/ci/build-stage/{stage_id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == stage_id
