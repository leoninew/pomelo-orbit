"""认证服务依赖注入"""

from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends

from pomelo_orbit.application.auth import AuthService
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure.cd.repositories.di import get_user_repo
from pomelo_orbit.infrastructure.config import get_settings


def get_auth_service(
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> AuthService:
    """获取认证服务实例"""
    return AuthService(user_repo, settings)
