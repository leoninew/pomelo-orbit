"""凭据 API 集成测试"""

from pomelo_orbit.infrastructure.ci.models import CredentialModel
from tests.integration.conftest import DEFAULT_CI_PROJECT_ID


class TestCredentialList:
    def test_returns_paginated_structure(self, auth_client, test_credential):
        resp = auth_client.get(f"/api/ci/credential?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data
        assert any(c["id"] == test_credential.id for c in data["items"])


class TestCredentialCreate:
    def test_creates_credential(self, auth_client):
        resp = auth_client.post(
            f"/api/ci/credential?project_id={DEFAULT_CI_PROJECT_ID}",
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


class TestCredentialDelete:
    def test_deletes_unreferenced_credential(self, auth_client, db_session, test_credential):
        resp = auth_client.delete(f"/api/ci/credential/{test_credential.id}")
        assert resp.status_code == 204
        assert db_session.query(CredentialModel).filter_by(id=test_credential.id).first() is None

    def test_cannot_delete_referenced_credential(self, auth_client, test_credential, test_project):
        resp = auth_client.delete(f"/api/ci/credential/{test_credential.id}")
        assert resp.status_code == 409
