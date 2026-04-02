"""
pytest 配置和 fixtures
"""

import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool

from pomelo_orbit.infrastructure.persistence.models import (
    Base,
    UserModel,
)
import pomelo_orbit.infrastructure.ci.models  # noqa: F401 — register CI ORM models with Base.metadata
from pomelo_orbit.infrastructure.security import hash_password


@pytest.fixture
def db_session():
    """创建内存数据库会话用于测试"""
    import pomelo_orbit.infrastructure.persistence.database as db_module

    # 使用 StaticPool 确保所有连接使用同一个内存数据库
    test_engine = create_engine(
        "sqlite:///:memory:",
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
    Base.metadata.create_all(bind=test_engine)
    test_session_factory = sessionmaker(bind=test_engine)
    session = test_session_factory()

    # 替换懒加载函数的缓存，使 get_engine() / get_session_factory() 返回测试实例
    db_module.get_engine.cache_clear()
    db_module.get_session_factory.cache_clear()

    original_get_engine = db_module.get_engine
    original_get_session_factory = db_module.get_session_factory

    db_module.get_engine = lambda: test_engine  # type: ignore[assignment]
    db_module.get_session_factory = lambda: test_session_factory  # type: ignore[assignment]

    yield session

    # 恢复原始函数
    db_module.get_engine = original_get_engine
    db_module.get_session_factory = original_get_session_factory
    db_module.get_engine.cache_clear()
    db_module.get_session_factory.cache_clear()

    session.close()
    Base.metadata.drop_all(bind=test_engine)
    test_engine.dispose()


@pytest.fixture
def test_user(db_session):
    """创建测试用户"""
    user = UserModel(
        username="testuser",
        password_hash=hash_password("testpassword"),
    )
    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)
    return user


@pytest.fixture
def client(db_session):
    """创建测试客户端（使用测试数据库）"""
    from fastapi.testclient import TestClient

    from pomelo_orbit.infrastructure.persistence.di import get_db
    from pomelo_orbit.main import app

    # 覆盖 get_db 依赖，使用测试数据库
    def override_get_db():
        try:
            yield db_session
            db_session.commit()
        except Exception:
            db_session.rollback()
            raise

    app.dependency_overrides[get_db] = override_get_db

    # 不使用上下文管理器，避免触发 lifespan 事件
    test_client = TestClient(app)
    yield test_client
    test_client.close()

    # 清理依赖覆盖
    app.dependency_overrides.clear()


def pytest_sessionfinish(session, exitstatus):
    """测试结束后清理数据库连接"""
    try:
        from pomelo_orbit.infrastructure.persistence.database import get_engine

        get_engine().dispose()
    except Exception:
        pass
