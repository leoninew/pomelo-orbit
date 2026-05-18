"""认证服务依赖注入"""

from typing import Annotated

from fastapi import Depends

from pomelo_orbit.application.auth import AuthService, RoleService, UserService
from pomelo_orbit.domain.auth.repositories import LoginAttemptRepository, PermissionRepository, RoleRepository
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import SecurityService, get_security_service
from pomelo_orbit.infrastructure.auth.di import get_login_attempt_repo, get_permission_repo, get_role_repo
from pomelo_orbit.infrastructure.cd.repositories.di import get_user_repo


def get_auth_service(
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    login_attempt_repo: Annotated[LoginAttemptRepository, Depends(get_login_attempt_repo)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
) -> AuthService:
    """获取认证服务实例"""
    return AuthService(user_repo, login_attempt_repo, security_service)


def get_user_service(
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    role_repo: Annotated[RoleRepository, Depends(get_role_repo)],
) -> UserService:
    return UserService(user_repo, role_repo)


def get_role_service(
    role_repo: Annotated[RoleRepository, Depends(get_role_repo)],
    permission_repo: Annotated[PermissionRepository, Depends(get_permission_repo)],
) -> RoleService:
    return RoleService(role_repo, permission_repo)
