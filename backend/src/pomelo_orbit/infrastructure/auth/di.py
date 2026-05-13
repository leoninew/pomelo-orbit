"""认证相关依赖注入"""

from collections.abc import Generator
from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.repositories import LoginAttemptRepository
from pomelo_orbit.infrastructure.auth.repositories import LoginAttemptRepositoryImpl
from pomelo_orbit.infrastructure.persistence.di import get_db


def get_login_attempt_repo(
    db: Annotated[Session, Depends(get_db)],
) -> Generator[LoginAttemptRepository, None, None]:
    """获取登录尝试记录仓储"""
    yield LoginAttemptRepositoryImpl(db)
