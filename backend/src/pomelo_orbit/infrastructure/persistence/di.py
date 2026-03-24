"""Persistence layer - dependency injection."""

from collections.abc import Generator

from sqlalchemy.orm import Session

from pomelo_orbit.infrastructure.persistence.database import get_session_factory

__all__ = ["get_db"]


def get_db() -> Generator[Session, None, None]:
    """获取数据库会话（用于 FastAPI 依赖注入）"""
    db = get_session_factory()()
    try:
        yield db
        db.commit()
    except Exception:
        db.rollback()
        raise
    finally:
        db.close()
