from pomelo_orbit.infrastructure.persistence.models import PermissionModel, RoleModel, RolePermissionModel


def create_role(
    db_session, code: str, name: str, description: str | None = None, permission_codes: list[str] | None = None
) -> RoleModel:
    role = RoleModel(code=code, name=name, description=description)
    db_session.add(role)
    db_session.flush()
    for permission_code in permission_codes or []:
        permission = db_session.query(PermissionModel).filter(PermissionModel.code == permission_code).one()
        db_session.add(RolePermissionModel(role_id=role.id, permission_id=permission.id))
    db_session.commit()
    db_session.refresh(role)
    return role


def test_create_and_list_roles(auth_client, db_session):
    response = auth_client.post(
        "/api/role",
        json={
            "code": "manager",
            "name": "Manager",
            "description": "Team manager",
            "permission_codes": ["user:read"],
        },
    )

    assert response.status_code == 201
    body = response.json()
    assert body["code"] == "manager"
    assert body["name"] == "Manager"
    assert body["description"] == "Team manager"
    assert "is_active" not in body
    assert body["permission_codes"] == ["user:read"]

    response = auth_client.get("/api/role", params={"search": "manager"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 1
    assert body["items"][0]["code"] == "manager"
    assert "is_active" not in body["items"][0]
    assert body["items"][0]["permission_codes"] == ["user:read"]


def test_create_role_rejects_duplicate_code_and_name(auth_client, db_session):
    create_role(db_session, "admin", "Admin")

    response = auth_client.post("/api/role", json={"code": "admin", "name": "Administrator"})
    assert response.status_code == 409

    response = auth_client.post("/api/role", json={"code": "root", "name": "Admin"})
    assert response.status_code == 409


def test_update_role(auth_client, db_session):
    role = create_role(db_session, "developer", "Developer", "Old", ["user:read"])

    response = auth_client.put(
        f"/api/role/{role.id}",
        json={"code": "dev", "name": "Dev", "description": "New", "permission_codes": ["role:read"]},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["code"] == "dev"
    assert body["name"] == "Dev"
    assert body["description"] == "New"
    assert body["permission_codes"] == ["role:read"]


def test_delete_role(auth_client, db_session):
    role = create_role(db_session, "guest", "Guest")

    response = auth_client.delete(f"/api/role/{role.id}")
    assert response.status_code == 204
    assert db_session.get(RoleModel, role.id) is None


def test_list_permissions(auth_client):
    response = auth_client.get("/api/role/permission")

    assert response.status_code == 200
    body = response.json()
    assert [item["code"] for item in body] == [
        "login:read",
        "role:read",
        "role:write",
        "setting:read",
        "setting:write",
        "user:read",
        "user:write",
    ]


def test_role_code_rejects_invalid_characters(auth_client):
    response = auth_client.post("/api/role", json={"code": "bad code", "name": "Bad"})

    assert response.status_code == 422
