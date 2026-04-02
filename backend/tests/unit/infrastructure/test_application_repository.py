"""ApplicationRepository 单元测试"""

from pomelo_orbit.domain.cd.entities import Application
from pomelo_orbit.infrastructure.cd.repositories.application import ApplicationRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestApplicationRepository:
    """应用仓储测试"""

    def test_save_and_find_by_id(self, db_session):
        """测试保存应用并通过 ID 查找"""
        repo = ApplicationRepositoryImpl(db_session)
        app = Application(
            id="app-1",
            name="Test App",
            code="test-app",
            image_pull_policy="IfNotPresent",
            enabled=True,
            status="stopped",
            created_at=utc_now(),
        )

        repo.save(app)
        found = repo.find_by_id("app-1")

        assert found is not None
        assert found.id == "app-1"
        assert found.name == "Test App"
        assert found.code == "test-app"

    def test_find_by_name(self, db_session):
        """测试通过名称查找应用"""
        repo = ApplicationRepositoryImpl(db_session)
        app = Application(
            id="app-2",
            name="My App",
            code="my-app",
            image_pull_policy="IfNotPresent",
            enabled=True,
            status="stopped",
            created_at=utc_now(),
        )

        repo.save(app)
        found = repo.find_by_name("My App")

        assert found is not None
        assert found.id == "app-2"
        assert found.name == "My App"

    def test_find_by_code(self, db_session):
        """测试通过编码查找应用"""
        repo = ApplicationRepositoryImpl(db_session)
        app = Application(
            id="app-3",
            name="Code App",
            code="code-app",
            image_pull_policy="IfNotPresent",
            enabled=True,
            status="stopped",
            created_at=utc_now(),
        )

        repo.save(app)
        found = repo.find_by_code("code-app")

        assert found is not None
        assert found.id == "app-3"
        assert found.code == "code-app"

    def test_update_application(self, db_session):
        """测试更新应用"""
        repo = ApplicationRepositoryImpl(db_session)
        app = Application(
            id="app-4",
            name="Old Name",
            code="old-code",
            image_pull_policy="IfNotPresent",
            enabled=True,
            status="stopped",
            created_at=utc_now(),
        )
        repo.save(app)

        app.name = "New Name"
        app.enabled = False
        repo.save(app)

        found = repo.find_by_id("app-4")
        assert found is not None
        assert found.name == "New Name"
        assert found.enabled is False

    def test_delete_application(self, db_session):
        """测试删除应用"""
        repo = ApplicationRepositoryImpl(db_session)
        app = Application(
            id="app-5",
            name="To Delete",
            code="to-delete",
            image_pull_policy="IfNotPresent",
            enabled=True,
            status="stopped",
            created_at=utc_now(),
        )
        repo.save(app)

        repo.delete(app)
        found = repo.find_by_id("app-5")
        assert found is None

    def test_find_paginated(self, db_session):
        """测试分页查询"""
        repo = ApplicationRepositoryImpl(db_session)
        for i in range(25):
            app = Application(
                id=f"app-page-{i}",
                name=f"App {i}",
                code=f"app-{i}",
                image_pull_policy="IfNotPresent",
                enabled=True,
                status="stopped",
                created_at=utc_now(),
            )
            repo.save(app)

        apps, total = repo.find_paginated(page=1, per_page=10)
        assert total == 25
        assert len(apps) == 10

        apps, total = repo.find_paginated(page=3, per_page=10)
        assert total == 25
        assert len(apps) == 5

    def test_find_paginated_with_search(self, db_session):
        """测试带搜索的分页查询"""
        repo = ApplicationRepositoryImpl(db_session)
        for i in range(10):
            name = f"Frontend {i}" if i < 5 else f"Backend {i}"
            app = Application(
                id=f"app-search-{i}",
                name=name,
                code=f"app-{i}",
                image_pull_policy="IfNotPresent",
                enabled=True,
                status="stopped",
                created_at=utc_now(),
            )
            repo.save(app)

        apps, total = repo.find_paginated(search="Frontend")
        assert total == 5
        assert len(apps) == 5
        assert all("Frontend" in app.name for app in apps)
