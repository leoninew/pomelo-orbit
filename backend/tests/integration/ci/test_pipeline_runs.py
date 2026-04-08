"""Pipeline Run API 集成测试"""

from unittest.mock import patch


class TestPipelineRunList:
    def test_returns_paginated_structure(self, auth_client):
        resp = auth_client.get("/api/ci/runs")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data

    def test_filter_by_project(self, auth_client, test_project):
        resp = auth_client.get(f"/api/ci/runs?project_id={test_project.id}")
        assert resp.status_code == 200
        assert "items" in resp.json()

    def test_list_by_project_endpoint(self, auth_client, test_project):
        resp = auth_client.get(f"/api/ci/projects/{test_project.id}/runs")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data


class TestPipelineRunGet:
    def test_not_found(self, auth_client):
        resp = auth_client.get("/api/ci/runs/nonexistent-id")
        assert resp.status_code == 404


class TestProjectTrigger:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_trigger_creates_run(self, mock_execute, auth_client, test_project, test_template):
        mock_execute.return_value = None
        resp = auth_client.post(
            f"/api/ci/projects/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert resp.status_code == 201
        data = resp.json()
        assert "id" in data
        assert data["project_id"] == test_project.id
        assert data["status"] in ("waiting_to_run", "running", "faulted", "ran_to_completion")

    def test_trigger_not_found(self, auth_client):
        resp = auth_client.post("/api/ci/projects/nonexistent/trigger", json={"template_id": "xxx"})
        assert resp.status_code == 404


class TestStageRunList:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_stage_runs_embedded_in_run(self, mock_execute, auth_client, test_project, test_template):
        mock_execute.return_value = None
        trigger_resp = auth_client.post(
            f"/api/ci/projects/{test_project.id}/trigger",
            json={"template_id": test_template.id, "trigger_ref": "main"},
        )
        assert trigger_resp.status_code == 201
        run_id = trigger_resp.json()["id"]

        # stage_runs 现在内嵌在 run 响应里，不再有独立的 /stages 端点
        resp = auth_client.get(f"/api/ci/runs/{run_id}")
        assert resp.status_code == 200
        assert "stage_runs" in resp.json()
        assert isinstance(resp.json()["stage_runs"], list)
