"""Backend CI webhook E2E tests."""

import hashlib
import hmac

from fastapi.testclient import TestClient

from tests.e2e.conftest import E2E_PROJECT_ID, auth_headers, seed_user
from tests.e2e.test_ci_pipeline_e2e import create_pipeline, wait_for_run


def github_signature(payload: bytes, secret: str) -> str:
    digest = hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()
    return f"sha256={digest}"


def test_github_webhook_triggers_run_for_matching_branch(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    pipeline = create_pipeline(e2e_client, headers, suffix="-hook")
    secret = "webhook-secret"
    webhook_resp = e2e_client.post(
        f"/api/ci/repository/{pipeline['repository_id']}/webhook?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "github", "template_id": pipeline["template_id"], "secret": secret, "branch_filter": "main"},
    )
    assert webhook_resp.status_code == 201, webhook_resp.text
    webhook_id = webhook_resp.json()["id"]

    payload = b'{"ref":"refs/heads/main","after":"abc123","pusher":{"name":"octocat"}}'
    response = e2e_client.post(
        f"/api/ci/webhook/{webhook_id}",
        content=payload,
        headers={"X-Hub-Signature-256": github_signature(payload, secret), "Content-Type": "application/json"},
    )

    assert response.status_code == 200, response.text
    assert response.json()["status"] == "triggered"
    run = wait_for_run(e2e_client, headers, response.json()["run_id"])
    assert run["trigger"] == "webhook"
    assert run["trigger_ref"] == "main"


def test_github_webhook_rejects_bad_or_missing_signature(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    pipeline = create_pipeline(e2e_client, headers, suffix="-bad-hook")
    webhook_resp = e2e_client.post(
        f"/api/ci/repository/{pipeline['repository_id']}/webhook?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "github", "template_id": pipeline["template_id"], "secret": "webhook-secret", "branch_filter": "main"},
    )
    assert webhook_resp.status_code == 201, webhook_resp.text
    webhook_id = webhook_resp.json()["id"]
    payload = b'{"ref":"refs/heads/main","after":"abc123"}'

    bad_response = e2e_client.post(
        f"/api/ci/webhook/{webhook_id}",
        content=payload,
        headers={"X-Hub-Signature-256": "sha256=bad", "Content-Type": "application/json"},
    )
    missing_response = e2e_client.post(
        f"/api/ci/webhook/{webhook_id}",
        content=payload,
        headers={"Content-Type": "application/json"},
    )

    assert bad_response.status_code == 401
    assert missing_response.status_code == 401


def test_github_webhook_ignores_non_matching_branch(e2e_client: TestClient, e2e_db) -> None:
    seed_user(e2e_db)
    headers = auth_headers(e2e_client)
    pipeline = create_pipeline(e2e_client, headers, suffix="-skip-hook")
    secret = "webhook-secret"
    webhook_resp = e2e_client.post(
        f"/api/ci/repository/{pipeline['repository_id']}/webhook?project_id={E2E_PROJECT_ID}",
        headers=headers,
        json={"name": "github", "template_id": pipeline["template_id"], "secret": secret, "branch_filter": "release/*"},
    )
    assert webhook_resp.status_code == 201, webhook_resp.text
    webhook_id = webhook_resp.json()["id"]
    payload = b'{"ref":"refs/heads/main","after":"abc123"}'

    response = e2e_client.post(
        f"/api/ci/webhook/{webhook_id}",
        content=payload,
        headers={"X-Hub-Signature-256": github_signature(payload, secret), "Content-Type": "application/json"},
    )
    runs_response = e2e_client.get(f"/api/ci/run?project_id={E2E_PROJECT_ID}", headers=headers)

    assert response.status_code == 200, response.text
    assert response.json() == {"status": "ignored", "reason": "branch filtered"}
    assert runs_response.status_code == 200, runs_response.text
    assert runs_response.json()["total"] == 0
