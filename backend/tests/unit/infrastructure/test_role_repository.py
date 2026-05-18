from pomelo_orbit.domain.auth.entities import Role
from pomelo_orbit.infrastructure.auth.repositories import RoleRepositoryImpl


def test_save_and_find_role_by_id(db_session):
    repo = RoleRepositoryImpl(db_session)
    role = Role(id="role-1", code="admin", name="Admin", description="Administrator", is_active=True)

    repo.save(role)

    found = repo.find_by_id("role-1")
    assert found is not None
    assert found.code == "admin"
    assert found.name == "Admin"
    assert found.description == "Administrator"


def test_find_role_by_code_and_name(db_session):
    repo = RoleRepositoryImpl(db_session)
    repo.save(Role(id="role-2", code="developer", name="Developer", description=None, is_active=True))

    by_code = repo.find_by_code("developer")
    by_name = repo.find_by_name("Developer")

    assert by_code is not None
    assert by_code.id == "role-2"
    assert by_name is not None
    assert by_name.id == "role-2"


def test_find_paginated_searches_role_fields(db_session):
    repo = RoleRepositoryImpl(db_session)
    repo.save(Role(id="role-3", code="qa", name="QA", description="Quality", is_active=True))
    repo.save(Role(id="role-4", code="ops", name="Operations", description="Deploy", is_active=True))

    roles, total = repo.find_paginated(page=1, per_page=10, search="Deploy")

    assert total == 1
    assert roles[0].code == "ops"


def test_delete_role(db_session):
    repo = RoleRepositoryImpl(db_session)
    role = Role(id="role-5", code="guest", name="Guest", description=None, is_active=True)
    repo.save(role)

    repo.delete(role)

    assert repo.find_by_id("role-5") is None
