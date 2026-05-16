class TestProjectApi:
    def test_create_and_list_projects(self, auth_client):
        create_resp = auth_client.post(
            "/api/project",
            json={"name": "Middleware", "code": "middleware"},
        )
        assert create_resp.status_code == 201
        created = create_resp.json()
        assert created["name"] == "Middleware"
        assert created["code"] == "middleware"
        assert created["owner_user_id"] == "test-user-id"

        list_resp = auth_client.get("/api/project")
        assert list_resp.status_code == 200
        projects = list_resp.json()
        assert any(project["id"] == created["id"] for project in projects)

    def test_get_project_rejects_foreign_owner(self, auth_client, db_session):
        from pomelo_orbit.domain.project.entities import Project
        from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl
        from pomelo_orbit.infrastructure.time_utils import utc_now

        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="foreign-project",
                name="Foreign Project",
                code="foreign-project",
                owner_user_id="other-user-id",
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )

        resp = auth_client.get("/api/project/foreign-project")

        assert resp.status_code == 400

    def test_update_project(self, auth_client):
        create_resp = auth_client.post(
            "/api/project",
            json={"name": "Old Name", "code": "old-name"},
        )
        project_id = create_resp.json()["id"]

        update_resp = auth_client.put(
            f"/api/project/{project_id}",
            json={"name": "New Name", "code": "new-name"},
        )

        assert update_resp.status_code == 200
        data = update_resp.json()
        assert data["name"] == "New Name"
        assert data["code"] == "new-name"
