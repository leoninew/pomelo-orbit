"""
认证 API 路由
"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query, Request
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from ulid import ULID

from pomelo_orbit.domain import AuthenticationError, AuthorizationError
from pomelo_orbit.domain.entities import LoginHistory, User
from pomelo_orbit.domain.repositories import UserRepository
from pomelo_orbit.infrastructure import SecurityService, get_security_service, hash_password, verify_password
from pomelo_orbit.infrastructure.persistence.mappers import LoginHistoryMapper
from pomelo_orbit.infrastructure.repositories import get_user_repository
from pomelo_orbit.infrastructure.time_utils import utc_now
from pomelo_orbit.interfaces.api.dto import (
    LoginHistoryResp,
    LoginReq,
    PaginatedResp,
    PasswordChangeReq,
    TokenResp,
    UserInfo,
)

router = APIRouter(prefix="/auth", tags=["auth"])
security = HTTPBearer()


def get_current_user(
    credentials: Annotated[HTTPAuthorizationCredentials, Depends(security)],
    user_repo: Annotated[UserRepository, Depends(get_user_repository)],
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
    if user is None:
        raise AuthorizationError("登录已过期, 请重新登录")
    return user


@router.post("/login", response_model=TokenResp)
def login(
    login_req: LoginReq,
    request: Request,
    user_repo: Annotated[UserRepository, Depends(get_user_repository)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
) -> TokenResp:
    """用户登录"""
    user = user_repo.find_by_username(login_req.username)
    if user is None or not verify_password(login_req.password, user.password_hash):
        raise AuthenticationError("用户名或密码错误")

    # 更新最后登录时间
    user.last_login_at = utc_now()
    user_repo.save(user)

    # 记录登录历史
    login_history = LoginHistory(
        id=str(ULID()),
        user_id=user.id,
        username=user.username,
        ip_address=request.client.host if request.client else None,
        user_agent=request.headers.get("user-agent"),
        login_at=utc_now(),
        success=True,
    )
    user_repo.save_login_history(login_history)

    # 生成 token
    access_token = security_service.create_access_token(data={"sub": user.username})
    return TokenResp(access_token=access_token)


@router.post("/logout")
def logout() -> dict:
    """用户登出（客户端删除 token 即可）"""
    return {"message": "Logged out successfully"}


@router.get("/me", response_model=UserInfo)
def get_me(current_user: Annotated[User, Depends(get_current_user)]) -> UserInfo:
    """获取当前用户信息"""
    return UserInfo(
        id=current_user.id,
        username=current_user.username,
        created_at=current_user.created_at,
        last_login_at=current_user.last_login_at,
    )


@router.put("/password")
def change_password(
    request: PasswordChangeReq,
    current_user: Annotated[User, Depends(get_current_user)],
    user_repo: Annotated[UserRepository, Depends(get_user_repository)],
) -> dict:
    """修改密码"""
    if not verify_password(request.old_password, current_user.password_hash):
        raise AuthenticationError("用户名或密码错误")
    current_user.password_hash = hash_password(request.new_password)
    user_repo.save(current_user)
    return {"message": "Password changed successfully"}


@router.get("/login-history", response_model=PaginatedResp[LoginHistoryResp])
def list_login_history(
    user_repo: Annotated[UserRepository, Depends(get_user_repository)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[LoginHistoryResp]:
    """列出登录历史"""
    history, total = user_repo.find_login_history(page=page, per_page=per_page, search=search)
    return PaginatedResp(
        items=[LoginHistoryResp.model_validate(LoginHistoryMapper.to_orm(h)) for h in history],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )
