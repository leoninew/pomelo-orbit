from pomelo_orbit.infrastructure.persistence.models import RoleModel


def create_role(db_session, code: str, name: str, description: str | None = None) -> RoleModel:
    role = RoleModel(code=code, name=name, description=description)
    db_session.add(role)
    db_session.commit()
    db_session.refresh(role)
    return role


def test_create_and_list_roles(auth_client, db_session):
    response = auth_client.post(
        "/api/role",
        json={"code": "admin", "name": "Admin", "description": "Administrator"},
    )

    assert response.status_code == 201
    body = response.json()
    assert body["code"] == "admin"
    assert body["name"] == "Admin"
    assert body["description"] == "Administrator"
    assert body["is_active"] is True

    response = auth_client.get("/api/role", params={"search": "admin"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 1
    assert body["items"][0]["code"] == "admin"


def test_create_role_rejects_duplicate_code_and_name(auth_client, db_session):
    create_role(db_session, "admin", "Admin")

    response = auth_client.post("/api/role", json={"code": "admin", "name": "Administrator"})
    assert response.status_code == 409

    response = auth_client.post("/api/role", json={"code": "root", "name": "Admin"})
    assert response.status_code == 409


def test_update_role(auth_client, db_session):
    role = create_role(db_session, "developer", "Developer", "Old")

    response = auth_client.put(
        f"/api/role/{role.id}",
        json={"code": "dev", "name": "Dev", "description": "New"},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["code"] == "dev"
    assert body["name"] == "Dev"
    assert body["description"] == "New"


def test_delete_role(auth_client, db_session):
    role = create_role(db_session, "guest", "Guest")

    response = auth_client.delete(f"/api/role/{role.id}")
    assert response.status_code == 204
    assert db_session.get(RoleModel, role.id) is None


def test_role_code_rejects_invalid_characters(auth_client):
    response = auth_client.post("/api/role", json={"code": "bad code", "name": "Bad"})

    assert response.status_code == 422
