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
                is_active=True,
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

    def test_deprecate_project(self, auth_client):
        # Create two projects
        create_resp1 = auth_client.post(
            "/api/project",
            json={"name": "Project 1", "code": "project-1"},
        )
        project_id1 = create_resp1.json()["id"]

        create_resp2 = auth_client.post(
            "/api/project",
            json={"name": "Project 2", "code": "project-2"},
        )
        project_id2 = create_resp2.json()["id"]

        # Deprecate one project should succeed
        deprecate_resp = auth_client.post(f"/api/project/{project_id1}/deprecate")
        assert deprecate_resp.status_code == 204

        # Verify project is marked inactive
        list_resp = auth_client.get("/api/project")
        projects = list_resp.json()
        project1 = next((p for p in projects if p["id"] == project_id1), None)
        project2 = next((p for p in projects if p["id"] == project_id2), None)
        assert project1 is not None
        assert project1["is_active"] is False
        assert project2 is not None
        assert project2["is_active"] is True

    def test_deprecate_last_active_project_fails(self, auth_client):
        # Get existing projects
        list_resp = auth_client.get("/api/project")
        projects = list_resp.json()

        # Deprecate all but one active project
        active_projects = [p for p in projects if p["is_active"]]
        while len(active_projects) > 1:
            auth_client.post(f"/api/project/{active_projects[0]['id']}/deprecate")
            list_resp = auth_client.get("/api/project")
            projects = list_resp.json()
            active_projects = [p for p in projects if p["is_active"]]

        # Try to deprecate the last active project
        deprecate_resp = auth_client.post(f"/api/project/{active_projects[0]['id']}/deprecate")
        assert deprecate_resp.status_code == 400
        assert "Cannot deprecate the last active project" in deprecate_resp.json()["detail"]

    def test_deprecate_project_rejects_foreign_owner(self, auth_client, db_session):
        from pomelo_orbit.domain.project.entities import Project
        from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl
        from pomelo_orbit.infrastructure.time_utils import utc_now

        repo = ProjectRepositoryImpl(db_session)
        repo.save(
            Project(
                id="foreign-project-deprecate",
                name="Foreign Project",
                code="foreign-project-deprecate",
                owner_user_id="other-user-id",
                is_active=True,
                created_at=utc_now(),
                updated_at=utc_now(),
            )
        )

        resp = auth_client.post("/api/project/foreign-project-deprecate/deprecate")
        assert resp.status_code == 400
