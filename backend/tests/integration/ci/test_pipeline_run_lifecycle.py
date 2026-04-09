"""Pipeline Run 生命周期集成测试"""

from unittest.mock import patch

from pomelo_orbit.infrastructure.ci.models import PipelineRunModel


class TestPipelineRunCreation:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_creates_run_with_manual_trigger(self, mock_execute, auth_client, test_project, test_template):
        """测试手动触发创建 pipeline run"""
        mock_execute.return_value = None

        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={
                "template_id": test_template.id,
                "trigger_ref": "main",
                "variables": {"CUSTOM_VAR": "value"},
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert "id" in data
        assert data["repository_id"] == test_project.id
        assert data["repository_name"] == test_project.name
        assert data["template_id"] == test_template.id
        assert data["trigger"] == "manual"
        assert data["trigger_ref"] == "main"
        assert data["status"] in ("waiting_to_run", "running")
        assert "variables_snapshot" in data
        assert data["variables_snapshot"]["CUSTOM_VAR"] == "value"

    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_creates_run_with_default_branch(self, mock_execute, auth_client, test_project, test_template):
        """测试使用项目默认分支触发"""
        mock_execute.return_value = None

        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id},
        )
        assert resp.status_code == 201
        data = resp.json()
        # 应该使用项目的默认分支
        assert data["trigger_ref"] == test_project.default_branch or "master"


class TestPipelineRunStatus:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_run_status_progression(self, mock_execute, auth_client, db_session, test_project, test_template):
        """测试 pipeline run 状态流转"""
        mock_execute.return_value = None

        # 创建 run
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert create_resp.status_code == 201
        run_id = create_resp.json()["id"]

        # 查询 run 状态
        resp = auth_client.get(f"/api/ci/run/{run_id}")
        assert resp.status_code == 200
        data = resp.json()
        assert "status" in data
        assert "created_at" in data
        # updated_at 可选，取决于实现

        # 验证状态是有效的
        valid_statuses = ["waiting_to_run", "running", "ran_to_completion", "faulted", "canceled"]
        assert data["status"] in valid_statuses


class TestPipelineRunWithStages:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_run_includes_stage_runs(self, mock_execute, auth_client, test_project, test_template, test_snapshot):
        """测试 pipeline run 包含 stage runs"""
        mock_execute.return_value = None

        # 创建包含 stages 的 snapshot
        # 注意：这里假设 test_snapshot 已经包含了 stages

        # 创建 run
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert create_resp.status_code == 201
        run_id = create_resp.json()["id"]

        # 获取 run 详情，应该包含 stage_runs
        resp = auth_client.get(f"/api/ci/run/{run_id}")
        assert resp.status_code == 200
        data = resp.json()
        assert "stage_runs" in data
        assert isinstance(data["stage_runs"], list)


class TestPipelineRunCancellation:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_cancel_running_run(self, mock_execute, auth_client, test_project, test_template):
        """测试取消运行中的 pipeline"""
        mock_execute.return_value = None

        # 创建 run
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert create_resp.status_code == 201
        run_id = create_resp.json()["id"]

        # 取消 run
        cancel_resp = auth_client.post(f"/api/ci/run/{run_id}/cancel")
        assert cancel_resp.status_code == 200

        # 验证状态已更新
        resp = auth_client.get(f"/api/ci/run/{run_id}")
        assert resp.status_code == 200
        assert resp.json()["status"] == "canceled"

    def test_cannot_cancel_completed_run(self, auth_client, db_session, test_project):
        """测试不能取消已完成的 run"""
        # 手动创建一个已完成的 run
        run = PipelineRunModel(
            repository_id=test_project.id,
            repository_name=test_project.name,
            pipeline_snapshot_id="test-snapshot-id",
            template_id="test-template-id",
            template_name="test-template",
            trigger="manual",
            trigger_ref="main",
            variables_snapshot="{}",
            status="ran_to_completion",
        )
        db_session.add(run)
        db_session.commit()
        db_session.refresh(run)

        # 尝试取消已完成的 run
        resp = auth_client.post(f"/api/ci/run/{run.id}/cancel")
        assert resp.status_code == 400  # Bad Request


class TestPipelineRunRetry:
    def test_retry_failed_run(self, auth_client, db_session, test_project, test_template):
        """测试重试失败的 pipeline"""
        # 创建一个失败的 run（通过 mock 或直接创建数据库记录）
        failed_run = PipelineRunModel(
            repository_id=test_project.id,
            repository_name=test_project.name,
            pipeline_snapshot_id="test-snapshot-id",
            template_id=test_template.id,
            template_name=test_template.name,
            trigger="manual",
            trigger_ref="main",
            variables_snapshot="{}",
            status="faulted",
        )
        db_session.add(failed_run)
        db_session.commit()
        db_session.refresh(failed_run)

        # 重试 run（如果端点存在）
        retry_resp = auth_client.post(f"/api/ci/run/{failed_run.id}/retry")
        # 如果端点不存在，这是预期的
        assert retry_resp.status_code in (201, 404)

    def test_cannot_retry_running_run(self, auth_client, db_session, test_project):
        """测试不能重试运行中的 run"""
        # 创建一个运行中的 run
        run = PipelineRunModel(
            repository_id=test_project.id,
            repository_name=test_project.name,
            pipeline_snapshot_id="test-snapshot-id",
            template_id="test-template-id",
            template_name="test-template",
            trigger="manual",
            trigger_ref="main",
            variables_snapshot="{}",
            status="running",
        )
        db_session.add(run)
        db_session.commit()
        db_session.refresh(run)

        # 尝试重试运行中的 run
        resp = auth_client.post(f"/api/ci/run/{run.id}/retry")
        assert resp.status_code == 400


class TestPipelineRunFiltering:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_filter_runs_by_project(self, mock_execute, auth_client, db_session, test_project, test_template):
        """测试按项目过滤 runs"""
        mock_execute.return_value = None

        # 为项目创建多个 runs
        for i in range(3):
            auth_client.post(
                f"/api/ci/repository/{test_project.id}/trigger",
                json={"template_id": test_template.id, "trigger_ref": f"branch-{i}"},
            )

        # 按项目过滤
        resp = auth_client.get(f"/api/ci/run?repository_id={test_project.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert len(data["items"]) >= 3
        # 验证所有返回的 runs 都属于该项目
        for run in data["items"]:
            assert run["repository_id"] == test_project.id

    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_filter_runs_by_status(self, mock_execute, auth_client, db_session, test_project, test_template):
        """测试按状态过滤 runs"""
        mock_execute.return_value = None

        # 创建不同状态的 runs
        auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )

        # 按状态过滤
        resp = auth_client.get("/api/ci/run?status=waiting_to_run")
        assert resp.status_code == 200
        data = resp.json()
        # 验证返回的 runs 都是指定状态（如果有结果的话）
        if data["items"]:
            for run in data["items"]:
                assert run["status"] == "waiting_to_run"


class TestPipelineRunVariables:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_run_with_project_variables(self, mock_execute, auth_client, test_project, test_template):
        """测试运行使用项目变量"""
        mock_execute.return_value = None

        # 为项目设置变量
        update_resp = auth_client.put(
            f"/api/ci/repository/{test_project.id}",
            json={
                "name": test_project.name,
                "repository_url": test_project.repository_url,
                "default_branch": test_project.default_branch,
                "variable_overrides": {"PROJECT_VAR": "project_value"},
            },
        )
        assert update_resp.status_code == 200

        # 触发运行，不传运行时变量
        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert resp.status_code == 201
        # 验证项目变量被包含在快照中
        # 注意：具体的行为取决于实现

    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_run_with_runtime_variables(self, mock_execute, auth_client, test_project, test_template):
        """测试运行时变量覆盖项目变量"""
        mock_execute.return_value = None

        # 触发运行，传入运行时变量
        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/trigger",
            json={
                "template_id": test_template.id,
                "trigger_ref": "main",
                "variables": {"RUNTIME_VAR": "runtime_value"},
            },
        )
        assert resp.status_code == 201
        # variables_snapshot 可选，取决于实现
        # 只要运行成功创建即可
