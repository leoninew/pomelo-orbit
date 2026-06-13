"""Representative backend authorization E2E tests."""

from fastapi.testclient import TestClient
from sqlalchemy.orm import Session

from pomelo_orbit.infrastructure.persistence.models import ProjectModel
from tests.e2e.conftest import E2E_PROJECT_ID, auth_headers, seed_user
from tests.e2e.test_ci_pipeline_e2e import create_pipeline

OTHER_PROJECT_ID = "01OTHER3NDEKTSV4RRFFQ69G5FAV"


def seed_other_project(session: Session) -> None:
    if session.get(ProjectModel, OTHER_PROJECT_ID) is None:
        session.add(ProjectModel(id=OTHER_PROJECT_ID, name="Other Project", code="other", is_active=True))
        session.commit()


def test_unauthenticated_requests_are_rejected(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db)

    ci_response = e2e_client.get(f"/api/ci/template?project_id={E2E_PROJECT_ID}")
    cd_response = e2e_client.get(f"/api/cd/application?project_id={E2E_PROJECT_ID}")

    assert ci_response.status_code in (401, 403)
    assert cd_response.status_code in (401, 403)


def test_non_member_cannot_access_project_lists(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db)
    seed_other_project(e2e_db)
    headers = auth_headers(e2e_client)

    ci_response = e2e_client.get(f"/api/ci/template?project_id={OTHER_PROJECT_ID}", headers=headers)
    cd_response = e2e_client.get(f"/api/cd/application?project_id={OTHER_PROJECT_ID}", headers=headers)

    assert ci_response.status_code == 403
    assert cd_response.status_code == 403


def test_non_member_cannot_access_ci_repository_detail(e2e_client: TestClient, e2e_db: Session) -> None:
    seed_user(e2e_db)
    member_headers = auth_headers(e2e_client)
    repository_id = create_pipeline(e2e_client, member_headers, suffix="-authz")["repository_id"]

    seed_user(e2e_db, user_id="other-user-id", username="otheruser", project_id=OTHER_PROJECT_ID)
    other_headers = auth_headers(e2e_client, username="otheruser")

    response = e2e_client.get(f"/api/ci/repository/{repository_id}", headers=other_headers)

    assert response.status_code == 403
