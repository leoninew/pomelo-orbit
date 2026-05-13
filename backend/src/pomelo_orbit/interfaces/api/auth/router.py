"""
认证 API 路由
"""

import logging
import math
import urllib.parse
from typing import Annotated

import httpx
from dynaconf import Dynaconf
from fastapi import APIRouter, Depends, HTTPException, Query, Request, status
from fastapi.responses import RedirectResponse
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from ulid import ULID

from pomelo_orbit.application.auth import AuthService
from pomelo_orbit.application.auth.di import get_auth_service
from pomelo_orbit.domain import AuthenticationError, AuthorizationError, BusinessError
from pomelo_orbit.domain.auth.entities import LoginHistory, User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import SecurityService, get_security_service, hash_password, verify_password
from pomelo_orbit.infrastructure.cd.repositories.di import get_user_repo
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.mappers import LoginHistoryMapper
from pomelo_orbit.infrastructure.time_utils import utc_now
from pomelo_orbit.interfaces.api.auth.dto import (
    GoogleCallbackReq,
    LoginHistoryResp,
    LoginReq,
    PasswordChangeReq,
    TokenResp,
    UserInfo,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp

router = APIRouter(prefix="/auth", tags=["auth"])
security = HTTPBearer()
logger = logging.getLogger(__name__)


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
    if user is None:
        raise AuthorizationError("登录已过期, 请重新登录")
    return user


@router.post("/login", response_model=TokenResp)
def login(
    login_req: LoginReq,
    request: Request,
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
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
        ip_address=request.headers.get(
            "X-Forwarded-For", request.headers.get("X-Real-IP", request.client.host if request.client else None)
        ),
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
        email=current_user.email,
        auth_source=current_user.auth_source,
        created_at=current_user.created_at,
        last_login_at=current_user.last_login_at,
    )


@router.put("/password")
def change_password(
    request: PasswordChangeReq,
    current_user: Annotated[User, Depends(get_current_user)],
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
) -> dict:
    """修改密码"""
    if not verify_password(request.old_password, current_user.password_hash):
        raise AuthenticationError("用户名或密码错误")
    current_user.password_hash = hash_password(request.new_password)
    user_repo.save(current_user)
    return {"message": "Password changed successfully"}


@router.get("/login-history", response_model=PaginatedResp[LoginHistoryResp])
def list_login_history(
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
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


@router.get("/google")
def google_oauth_redirect(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> RedirectResponse:
    """重定向到 Google OAuth 授权页面"""
    client_id = settings.google.client_id
    redirect_uri = settings.google.redirect_uri
    if not client_id or not redirect_uri:
        raise HTTPException(status_code=status.HTTP_503_SERVICE_UNAVAILABLE, detail="Google OAuth 未配置")
    params = urllib.parse.urlencode(
        {
            "client_id": client_id,
            "redirect_uri": redirect_uri,
            "response_type": "code",
            "scope": "openid email profile",
            "access_type": "offline",
            "prompt": "select_account",
        }
    )
    return RedirectResponse(url=f"https://accounts.google.com/o/oauth2/v2/auth?{params}")


@router.post("/google/callback", response_model=TokenResp)
def google_callback(
    req: GoogleCallbackReq,
    auth_service: Annotated[AuthService, Depends(get_auth_service)],
    user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> TokenResp:
    """Google OAuth 回调处理"""
    client_id = settings.google.client_id
    client_secret = settings.google.client_secret
    redirect_uri = settings.google.redirect_uri
    assert client_id, "google.client_id 未配置"
    assert client_secret, "google.client_secret 未配置"
    assert redirect_uri, "google.redirect_uri 未配置"

    # 用 code 换取 token
    try:
        token_resp = httpx.post(
            "https://oauth2.googleapis.com/token",
            data={
                "code": req.code,
                "client_id": client_id,
                "client_secret": client_secret,
                "redirect_uri": redirect_uri,
                "grant_type": "authorization_code",
            },
            timeout=10.0,
        )
    except httpx.HTTPError as e:
        raise BusinessError("Google 授权请求失败, 请检查网络或稍后重试", status_code=503) from e

    if token_resp.status_code != 200:
        logger.warning(f"Google token exchange failed: status={token_resp.status_code}")
        raise BusinessError("Google 授权失败, 请重试", status_code=400)

    token_data = token_resp.json()
    id_token = token_data.get("id_token")
    if not id_token:
        raise BusinessError("Google 未返回 id_token", status_code=400)

    # 验证 id_token 并创建/查找用户
    user = auth_service.oauth_login("google", id_token)

    # 更新最后登录时间
    user.last_login_at = utc_now()
    user_repo.save(user)

    # 生成 JWT token
    access_token = security_service.create_access_token(data={"sub": user.username})
    return TokenResp(access_token=access_token)
