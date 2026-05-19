"""UserRepository 单元测试"""

from pomelo_orbit.domain.auth.entities import LoginHistory, User
from pomelo_orbit.infrastructure.cd.repositories.user import UserRepositoryImpl
from pomelo_orbit.infrastructure.persistence.models import (
    PermissionModel,
    RoleModel,
    RolePermissionModel,
    UserRoleModel,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestUserRepository:
    """用户仓储测试"""

    def test_save_and_find_by_id(self, db_session):
        """测试保存用户并通过 ID 查找"""
        repo = UserRepositoryImpl(db_session)
        user = User(
            id="user-1",
            username="testuser",
            password_hash="hashed_pwd",
            status="enabled",
            oauth_provider="",
            oauth_provider_id="",
            email=None,
            auth_source="password",
            created_at=utc_now(),
        )

        repo.save(user)
        found = repo.find_by_id("user-1")

        assert found is not None
        assert found.id == "user-1"
        assert found.username == "testuser"
        assert found.password_hash == "hashed_pwd"

    def test_find_by_id_not_found(self, db_session):
        """测试查找不存在的用户"""
        repo = UserRepositoryImpl(db_session)
        found = repo.find_by_id("non-existent")
        assert found is None

    def test_find_by_username(self, db_session):
        """测试通过用户名查找"""
        repo = UserRepositoryImpl(db_session)
        user = User(
            id="user-2",
            username="alice",
            password_hash="hashed",
            status="enabled",
            oauth_provider="",
            oauth_provider_id="",
            email=None,
            auth_source="password",
            created_at=utc_now(),
        )

        repo.save(user)
        found = repo.find_by_username("alice")

        assert found is not None
        assert found.id == "user-2"
        assert found.username == "alice"

    def test_find_by_username_not_found(self, db_session):
        """测试查找不存在的用户名"""
        repo = UserRepositoryImpl(db_session)
        found = repo.find_by_username("nonexistent")
        assert found is None

    def test_find_paginated_searches_username_and_email(self, db_session):
        repo = UserRepositoryImpl(db_session)
        repo.save(
            User(
                id="user-10",
                username="alice",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email="alice@example.com",
                auth_source="password",
            )
        )
        repo.save(
            User(
                id="user-11",
                username="bob",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email="team@example.com",
                auth_source="password",
            )
        )
        repo.save(
            User(
                id="user-12",
                username="carol",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email="carol@example.com",
                auth_source="password",
            )
        )

        users, total = repo.find_paginated(page=1, per_page=10, search="example")

        assert total == 3
        assert {user.username for user in users} == {"alice", "bob", "carol"}

        users, total = repo.find_paginated(page=1, per_page=10, search="team")
        assert total == 1
        assert users[0].username == "bob"

    def test_update_user(self, db_session):
        """测试更新用户"""
        repo = UserRepositoryImpl(db_session)
        user = User(
            id="user-3",
            username="bob",
            password_hash="old_hash",
            status="enabled",
            oauth_provider="",
            oauth_provider_id="",
            email=None,
            auth_source="password",
            created_at=utc_now(),
        )

        repo.save(user)

        user.password_hash = "new_hash"
        repo.save(user)

        found = repo.find_by_id("user-3")
        assert found is not None
        assert found.password_hash == "new_hash"

    def test_set_and_find_roles(self, db_session):
        repo = UserRepositoryImpl(db_session)
        repo.save(
            User(
                id="user-13",
                username="alice",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email=None,
                auth_source="password",
            )
        )
        db_session.add(RoleModel(id="role-1", code="developer", name="Developer"))
        db_session.add(RoleModel(id="role-2", code="admin", name="Admin"))
        db_session.commit()

        repo.set_roles("user-13", ["role-1", "role-2"])
        db_session.commit()

        roles = repo.find_roles("user-13")
        assert [role.code for role in roles] == ["admin", "developer"]

        repo.set_roles("user-13", ["role-1"])
        db_session.commit()

        roles = repo.find_roles("user-13")
        assert [role.code for role in roles] == ["developer"]
        assert db_session.get(UserRoleModel, ("user-13", "role-2")) is None

    def test_find_roles_by_user_ids(self, db_session):
        repo = UserRepositoryImpl(db_session)
        repo.save(
            User(
                id="user-15",
                username="alice",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email=None,
                auth_source="password",
            )
        )
        repo.save(
            User(
                id="user-16",
                username="bob",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email=None,
                auth_source="password",
            )
        )
        db_session.add(RoleModel(id="role-5", code="developer", name="Developer"))
        db_session.add(RoleModel(id="role-6", code="admin", name="Admin"))
        db_session.add(UserRoleModel(user_id="user-15", role_id="role-5"))
        db_session.add(UserRoleModel(user_id="user-15", role_id="role-6"))
        db_session.commit()

        roles_by_user_id = repo.find_roles_by_user_ids(["user-15", "user-16"])

        assert [role.code for role in roles_by_user_id["user-15"]] == ["admin", "developer"]
        assert roles_by_user_id["user-16"] == []

    def test_find_permissions(self, db_session):
        repo = UserRepositoryImpl(db_session)
        repo.save(
            User(
                id="user-14",
                username="alice",
                password_hash="hashed",
                status="enabled",
                oauth_provider="",
                oauth_provider_id="",
                email=None,
                auth_source="password",
            )
        )
        db_session.add(RoleModel(id="role-3", code="developer", name="Developer"))
        db_session.add(RoleModel(id="role-4", code="viewer", name="Viewer"))
        db_session.add(PermissionModel(id="perm-1", code="user:read", name="View Users"))
        db_session.add(RolePermissionModel(role_id="role-3", permission_id="perm-1"))
        db_session.add(RolePermissionModel(role_id="role-4", permission_id="perm-1"))
        db_session.add(UserRoleModel(user_id="user-14", role_id="role-3"))
        db_session.add(UserRoleModel(user_id="user-14", role_id="role-4"))
        db_session.commit()

        permissions = repo.find_permissions("user-14")

        assert [permission.code for permission in permissions] == ["user:read"]

    def test_save_login_history(self, db_session):
        """测试保存登录历史"""
        repo = UserRepositoryImpl(db_session)
        history = LoginHistory(
            id="history-1",
            user_id="user-1",
            username="testuser",
            ip_address="192.168.1.1",
            user_agent="Mozilla/5.0",
            login_at=utc_now(),
            success=True,
        )

        repo.save_login_history(history)
        histories, total = repo.find_login_history(page=1, per_page=10)

        assert total == 1
        assert len(histories) == 1
        assert histories[0].username == "testuser"
        assert histories[0].ip_address == "192.168.1.1"
        assert histories[0].success is True

    def test_find_login_history_pagination(self, db_session):
        """测试登录历史分页"""
        repo = UserRepositoryImpl(db_session)
        for i in range(25):
            history = LoginHistory(
                id=f"history-{i}",
                user_id=f"user-{i}",
                username=f"user{i}",
                ip_address="127.0.0.1",
                user_agent="test",
                login_at=utc_now(),
                success=True,
            )
            repo.save_login_history(history)

        histories, total = repo.find_login_history(page=1, per_page=10)
        assert total == 25
        assert len(histories) == 10

        histories, total = repo.find_login_history(page=3, per_page=10)
        assert total == 25
        assert len(histories) == 5

    def test_find_login_history_search(self, db_session):
        """测试登录历史搜索"""
        repo = UserRepositoryImpl(db_session)
        for i in range(5):
            history = LoginHistory(
                id=f"history-{i}",
                user_id=f"user-{i}",
                username=f"alice{i}" if i < 3 else f"bob{i}",
                ip_address="127.0.0.1",
                user_agent="test",
                login_at=utc_now(),
                success=True,
            )
            repo.save_login_history(history)

        histories, total = repo.find_login_history(search="alice")
        assert total == 3
        assert len(histories) == 3
        assert all("alice" in h.username for h in histories)
