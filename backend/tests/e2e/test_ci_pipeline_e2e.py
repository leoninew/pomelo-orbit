"""Backend CI pipeline E2E tests."""

import time
from typing import Any, cast

from fastapi.testclient import TestClient

from tests.e2e.conftest import E2E_PROJECT_ID, auth_headers, seed_user


def create_pipeline(client: TestClient, headers: dict[str, str], suffix: str = "") -> dict[str, str]:
    credential_resp = client.post(
        f"/api/ci/credential?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": f"e2e-cred{suffix}", "type": "github_token", "data": "ghp_test"},
    )
    assert credential_resp.status_code == 201, credential_resp.text

    repository_resp = client.post(
        f"/api/ci/repository?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": f"e2e-repo{suffix}",
            "code": f"e2e-repo{suffix}".strip("-"),
            "repository_url": "https://github.com/example/repo.git",
            "git_credential_id": None,
            "default_branch": "main",
            "variable_overrides": [{"name": "CUSTOM_VAR", "value": "repo-value", "source": "repository_custom"}],
        },
    )
    assert repository_resp.status_code == 201, repository_resp.text
    repository_id = repository_resp.json()["id"]

    stage_resp = client.post(
        f"/api/ci/build-stage?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": f"build{suffix}",
            "image": "alpine:latest",
            "script": "echo {{ repository_url }} && echo {{ CUSTOM_VAR }}",
            "artifacts": [{"type": "binary", "path": "dist/app.tar.gz", "name": "app"}],
            "description": "E2E build stage",
        },
    )
    assert stage_resp.status_code == 201, stage_resp.text
    stage = stage_resp.json()

    template_resp = client.post(
        f"/api/ci/template?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": f"e2e-template{suffix}",
            "description": "E2E template",
            "variable_declarations": [
                {"name": "CUSTOM_VAR", "value": "template-value", "source": "template_custom"},
            ],
        },
    )
    assert template_resp.status_code == 201, template_resp.text
    template_id = template_resp.json()["id"]

    update_resp = client.put(
        f"/api/ci/template/{template_id}",
        headers=headers,
        json={
            "orchestration": [
                {
                    "stage_id": stage["id"],
                    "stage_name": "build",
                    "stage_version": stage["version"],
                    "depends_on": [],
                    "sort_order": 0,
                }
            ],
            "variable_declarations": [
                {"name": "CUSTOM_VAR", "value": "template-value", "source": "template_custom"},
            ],
        },
    )
    assert update_resp.status_code == 200, update_resp.text

    return {"repository_id": repository_id, "template_id": template_id, "stage_id": stage["id"]}


def trigger_pipeline(
    client: TestClient,
    headers: dict[str, str],
    repository_id: str,
    template_id: str,
    variables: dict[str, str] | None = None,
) -> dict[str, Any]:
    payload: dict[str, object] = {"template_id": template_id, "trigger_ref": "main"}
    if variables is not None:
        payload["variables"] = variables
    trigger_resp = client.post(
        f"/api/ci/repository/{repository_id}/trigger",
        headers=headers,
        json=payload,
    )
    assert trigger_resp.status_code == 201, trigger_resp.text
    result = trigger_resp.json()
    assert isinstance(result, dict)
    return cast("dict[str, Any]", result)


def wait_for_run(client: TestClient, headers: dict[str, str], run_id: str) -> dict[str, Any]:
    run: dict[str, Any] = {}
    for _ in range(20):
        response = client.get(f"/api/ci/run/{run_id}", headers=headers)
        assert response.status_code == 200, response.text
        result = response.json()
        assert isinstance(result, dict)
        run = cast("dict[str, Any]", result)
        if run["status"] in {"ran_to_completion", "faulted", "canceled"}:
            return run
        time.sleep(0.05)
    return run


def test_pipeline_trigger_creates_run_snapshot_artifacts_and_stage_log(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    pipeline = create_pipeline(e2e_client, headers)

    created_run = trigger_pipeline(e2e_client, headers, pipeline["repository_id"], pipeline["template_id"])
    completed_run = wait_for_run(e2e_client, headers, created_run["id"])

    assert completed_run["status"] == "ran_to_completion"
    assert completed_run["snapshot_id"]
    assert completed_run["stage_runs"]

    snapshot_resp = e2e_client.get(f"/api/ci/snapshot/{completed_run['snapshot_id']}", headers=headers)
    assert snapshot_resp.status_code == 200, snapshot_resp.text
    snapshot = snapshot_resp.json()
    assert snapshot["stages_snapshot"][0]["name"] == "build"
    snapshot_vars = {item["name"] for item in snapshot["variables_snapshot"]}
    assert "repository_url" in snapshot_vars
    assert "CUSTOM_VAR" in snapshot_vars

    artifacts_resp = e2e_client.get(f"/api/ci/run/{completed_run['id']}/artifacts", headers=headers)
    assert artifacts_resp.status_code == 200, artifacts_resp.text
    artifacts = artifacts_resp.json()
    assert len(artifacts) == 1
    assert artifacts[0]["name"] == "app"

    artifact_list_resp = e2e_client.get(f"/api/ci/artifact?project_id={E2E_PROJECT_ID}", headers=headers)
    assert artifact_list_resp.status_code == 200, artifact_list_resp.text
    assert artifact_list_resp.json()["total"] == 1

    stage_run_id = completed_run["stage_runs"][0]["id"]
    log_resp = e2e_client.get(f"/api/ci/run/{completed_run['id']}/stages/{stage_run_id}/log", headers=headers)
    assert log_resp.status_code == 200, log_resp.text
    assert "https://github.com/example/repo.git" in log_resp.json()["logs"]
    assert "repo-value" in log_resp.json()["logs"]
    assert log_resp.json()["is_complete"] is True


def test_pipeline_reuses_snapshot_until_template_changes(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    pipeline = create_pipeline(e2e_client, headers, suffix="-reuse")

    first_run = wait_for_run(
        e2e_client,
        headers,
        trigger_pipeline(e2e_client, headers, pipeline["repository_id"], pipeline["template_id"])["id"],
    )
    second_run = wait_for_run(
        e2e_client,
        headers,
        trigger_pipeline(e2e_client, headers, pipeline["repository_id"], pipeline["template_id"])["id"],
    )

    assert second_run["snapshot_id"] == first_run["snapshot_id"]

    update_resp = e2e_client.put(
        f"/api/ci/build-stage/{pipeline['stage_id']}",
        headers=headers,
        json={"script": "echo {{ repository_url }} && echo {{ template_id }}"},
    )
    assert update_resp.status_code == 200, update_resp.text
    stage = update_resp.json()
    template_resp = e2e_client.get(f"/api/ci/template/{pipeline['template_id']}", headers=headers)
    assert template_resp.status_code == 200, template_resp.text
    template = template_resp.json()
    orchestration = template["orchestration"]
    orchestration[0]["stage_version"] = stage["version"]
    refresh_resp = e2e_client.put(
        f"/api/ci/template/{pipeline['template_id']}",
        headers=headers,
        json={"orchestration": orchestration, "variable_declarations": template["variable_declarations"]},
    )
    assert refresh_resp.status_code == 200, refresh_resp.text

    third_run = wait_for_run(
        e2e_client,
        headers,
        trigger_pipeline(e2e_client, headers, pipeline["repository_id"], pipeline["template_id"])["id"],
    )

    assert third_run["snapshot_id"] != first_run["snapshot_id"]


def test_pipeline_runtime_variables_apply_expected_precedence(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)

    credential_resp = e2e_client.post(
        f"/api/ci/credential?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "e2e-runtime-cred", "type": "github_token", "data": "ghp_test"},
    )
    assert credential_resp.status_code == 201, credential_resp.text

    repository_resp = e2e_client.post(
        f"/api/ci/repository?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": "e2e-runtime-repo",
            "code": "e2e-runtime-repo",
            "repository_url": "https://github.com/example/runtime.git",
            "git_credential_id": credential_resp.json()["id"],
            "default_branch": "main",
            "variable_overrides": [
                {"name": "REPO_VAR", "value": "repo-value"},
                {"name": "SHARED_VAR", "value": "from-repo"},
            ],
        },
    )
    assert repository_resp.status_code == 201, repository_resp.text
    repository_id = repository_resp.json()["id"]

    stage_resp = e2e_client.post(
        f"/api/ci/build-stage?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": "runtime-build",
            "image": "alpine:latest",
            "script": "echo {{ repository_id }} {{ template_id }} {{ REPO_VAR }} {{ TEMPLATE_VAR }} {{ SHARED_VAR }} {{ RUNTIME_VAR }}",
            "description": "Runtime variable precedence stage",
        },
    )
    assert stage_resp.status_code == 201, stage_resp.text
    stage = stage_resp.json()

    template_resp = e2e_client.post(
        f"/api/ci/template?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={
            "name": "e2e-runtime-template",
            "description": "Runtime variable precedence template",
            "variable_declarations": [
                {"name": "TEMPLATE_VAR", "value": "template-value", "source": "template_custom"},
                {"name": "SHARED_VAR", "value": "from-template", "source": "template_custom"},
            ],
        },
    )
    assert template_resp.status_code == 201, template_resp.text
    template_id = template_resp.json()["id"]

    update_resp = e2e_client.put(
        f"/api/ci/template/{template_id}",
        headers=headers,
        json={
            "orchestration": [
                {
                    "stage_id": stage["id"],
                    "stage_name": "runtime-build",
                    "stage_version": stage["version"],
                    "depends_on": [],
                    "sort_order": 0,
                }
            ],
            "variable_declarations": [
                {"name": "TEMPLATE_VAR", "value": "template-value", "source": "template_custom"},
                {"name": "SHARED_VAR", "value": "from-template", "source": "template_custom"},
            ],
        },
    )
    assert update_resp.status_code == 200, update_resp.text

    run = wait_for_run(
        e2e_client,
        headers,
        trigger_pipeline(
            e2e_client,
            headers,
            repository_id,
            template_id,
            variables={"RUNTIME_VAR": "runtime-value", "SHARED_VAR": "from-runtime", "repository_id": "fake-id"},
        )["id"],
    )
    assert run["status"] == "ran_to_completion"

    variables = {item["name"]: item for item in run["variables_snapshot"]}
    assert variables["repository_id"]["value"] == repository_id
    assert variables["template_id"]["value"] == template_id
    assert variables["REPO_VAR"]["value"] == "repo-value"
    assert variables["TEMPLATE_VAR"]["value"] == "template-value"
    assert variables["RUNTIME_VAR"]["value"] == "runtime-value"
    assert variables["SHARED_VAR"]["value"] == "from-runtime"
