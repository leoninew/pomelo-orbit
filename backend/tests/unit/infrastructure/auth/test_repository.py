from pomelo_orbit.domain.auth.entities import Role
from pomelo_orbit.infrastructure.auth.repositories import PermissionRepositoryImpl, RoleRepositoryImpl
from pomelo_orbit.infrastructure.persistence.models import PermissionModel, RolePermissionModel, UserRoleModel


def test_save_and_find_role_by_id(db_session):
    repo = RoleRepositoryImpl(db_session)
    role = Role(id="role-1", code="admin", name="Admin", description="Administrator")

    repo.save(role)

    found = repo.find_by_id("role-1")
    assert found is not None
    assert found.code == "admin"
    assert found.name == "Admin"
    assert found.description == "Administrator"


def test_find_role_by_code_and_name(db_session):
    repo = RoleRepositoryImpl(db_session)
    repo.save(Role(id="role-2", code="developer", name="Developer", description=None))

    by_code = repo.find_by_code("developer")
    by_name = repo.find_by_name("Developer")

    assert by_code is not None
    assert by_code.id == "role-2"
    assert by_name is not None
    assert by_name.id == "role-2"


def test_find_paginated_searches_role_fields(db_session):
    repo = RoleRepositoryImpl(db_session)
    repo.save(Role(id="role-3", code="qa", name="QA", description="Quality"))
    repo.save(Role(id="role-4", code="ops", name="Operations", description="Deploy"))

    roles, total = repo.find_paginated(page=1, per_page=10, search="Deploy")

    assert total == 1
    assert roles[0].code == "ops"


def test_delete_role(db_session):
    repo = RoleRepositoryImpl(db_session)
    role = Role(id="role-5", code="guest", name="Guest", description=None)
    repo.save(role)

    repo.delete(role)

    assert repo.find_by_id("role-5") is None


def test_list_permissions(db_session):
    repo = PermissionRepositoryImpl(db_session)
    db_session.add(PermissionModel(id="perm-1", code="role:read", name="View Roles"))
    db_session.add(PermissionModel(id="perm-2", code="user:read", name="View Users"))
    db_session.commit()

    permissions = repo.list_all()

    assert [permission.code for permission in permissions] == ["role:read", "user:read"]


def test_set_and_find_role_permissions(db_session):
    role_repo = RoleRepositoryImpl(db_session)
    repo = PermissionRepositoryImpl(db_session)
    role_repo.save(Role(id="role-6", code="viewer", name="Viewer", description=None))
    db_session.add(PermissionModel(id="perm-3", code="user:read", name="View Users"))
    db_session.add(PermissionModel(id="perm-4", code="user:write", name="Manage Users"))
    db_session.commit()

    repo.set_role_permissions("role-6", ["user:read", "user:write"])
    db_session.commit()

    permissions = repo.find_by_role_id("role-6")
    assert [permission.code for permission in permissions] == ["user:read", "user:write"]

    repo.set_role_permissions("role-6", ["user:read"])
    db_session.commit()

    permissions = repo.find_by_role_id("role-6")
    assert [permission.code for permission in permissions] == ["user:read"]


def test_find_permissions_by_user_id(db_session):
    role_repo = RoleRepositoryImpl(db_session)
    repo = PermissionRepositoryImpl(db_session)
    role_repo.save(Role(id="role-7", code="viewer", name="Viewer", description=None))
    role_repo.save(Role(id="role-8", code="auditor", name="Auditor", description=None))
    db_session.add(PermissionModel(id="perm-5", code="user:read", name="View Users"))
    db_session.add(RolePermissionModel(role_id="role-7", permission_id="perm-5"))
    db_session.add(RolePermissionModel(role_id="role-8", permission_id="perm-5"))
    db_session.add(UserRoleModel(user_id="user-1", role_id="role-7"))
    db_session.add(UserRoleModel(user_id="user-1", role_id="role-8"))
    db_session.commit()

    permissions = repo.find_by_user_id("user-1")

    assert [permission.code for permission in permissions] == ["user:read"]
