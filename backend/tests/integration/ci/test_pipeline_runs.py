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
    def test_trigger_creates_run(self, mock_execute, auth_client, test_project):
        mock_execute.return_value = None
        resp = auth_client.post(f"/api/ci/projects/{test_project.id}/trigger", json={})
        assert resp.status_code == 201
        data = resp.json()
        assert "id" in data
        assert data["project_id"] == test_project.id
        assert data["status"] in ("waiting", "running", "failed", "success")

    def test_trigger_not_found(self, auth_client):
        resp = auth_client.post("/api/ci/projects/nonexistent/trigger", json={})
        assert resp.status_code == 404


class TestJobList:
    @patch("pomelo_orbit.application.ci.pipeline_service.PipelineService.execute_run")
    def test_list_jobs_for_run(self, mock_execute, auth_client, test_project):
        mock_execute.return_value = None
        trigger_resp = auth_client.post(f"/api/ci/projects/{test_project.id}/trigger", json={})
        assert trigger_resp.status_code == 201
        run_id = trigger_resp.json()["id"]

        resp = auth_client.get(f"/api/ci/runs/{run_id}/jobs")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)


class TestJobLogs:
    def test_not_found_returns_empty(self, auth_client):
        resp = auth_client.get("/api/ci/jobs/nonexistent-job/logs")
        assert resp.status_code == 200
        assert resp.json() is None
