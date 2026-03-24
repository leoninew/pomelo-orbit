"""
数据库连接和会话管理
"""

from functools import lru_cache
from pathlib import Path

from sqlalchemy import Engine, create_engine
from sqlalchemy.orm import Session, sessionmaker

from pomelo_orbit.infrastructure.config import get_project_root, get_settings


def _build_database_url() -> str:
    """根据配置生成数据库连接 URL"""
    settings = get_settings()
    db_type = settings.database.type

    if db_type == "sqlite":
        db_path = Path(settings.database.sqlite.path)
        if not db_path.is_absolute():
            db_path = get_project_root() / db_path
        db_path.parent.mkdir(parents=True, exist_ok=True)
        return f"sqlite:///{db_path}"
    if db_type == "mysql":
        mysql_config = settings.database.mysql
        return (
            f"mysql+pymysql://{mysql_config.user}:{mysql_config.password}"
            f"@{mysql_config.host}:{mysql_config.port}/{mysql_config.database}"
        )
    raise ValueError(f"Unsupported database type: {db_type}")


@lru_cache
def get_engine() -> Engine:
    """获取数据库引擎（单例，懒加载）"""
    settings = get_settings()
    return create_engine(
        _build_database_url(),
        echo=settings.database.echo,
        pool_pre_ping=True,
    )


@lru_cache
def get_session_factory() -> sessionmaker[Session]:
    """获取 Session 工厂（单例，懒加载）"""
    return sessionmaker(autocommit=False, autoflush=True, bind=get_engine())
