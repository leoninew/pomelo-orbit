"""UserRepository 单元测试"""

from pomelo_orbit.domain.auth.entities import LoginHistory, User
from pomelo_orbit.infrastructure.cd.repositories.user import UserRepositoryImpl
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

    def test_update_user(self, db_session):
        """测试更新用户"""
        repo = UserRepositoryImpl(db_session)
        user = User(
            id="user-3",
            username="bob",
            password_hash="old_hash",
            created_at=utc_now(),
        )

        repo.save(user)

        user.password_hash = "new_hash"
        repo.save(user)

        found = repo.find_by_id("user-3")
        assert found is not None
        assert found.password_hash == "new_hash"

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
