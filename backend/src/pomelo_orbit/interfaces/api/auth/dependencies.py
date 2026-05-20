from typing import Annotated

from fastapi import Depends
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from pomelo_orbit.domain import AuthorizationError
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import SecurityService, get_security_service
from pomelo_orbit.infrastructure.cd.repositories.di import get_user_repo

security = HTTPBearer()


def get_current_user(
    credentials: Annotated[HTTPAuthorizationCredentials, Depends(security)],
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
) -> User:
    """获取当前用户"""
    token = credentials.credentials
    payload = security_service.decode_access_token(token)
    if payload is None:
        raise AuthorizationError("登录已过期, 请重新登录")
    username = payload.get("sub")
    if username is None:
        raise AuthorizationError("登录已过期, 请重新登录")
    user = user_repo.find_by_username(username)
    if user is None or user.status == "disabled":
        raise AuthorizationError("登录已过期, 请重新登录")
    return user
