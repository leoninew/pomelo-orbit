"""认证服务 - OAuth"""

import logging

import ulid
from dynaconf import Dynaconf
from google.auth.transport import requests as google_requests
from google.oauth2 import id_token as google_id_token

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.constants import AuthSource, OAuthProvider
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import hash_password

logger = logging.getLogger(__name__)


class AuthService:
    """认证服务"""

    def __init__(self, user_repo: UserRepository, settings: Dynaconf):
        self._user_repo = user_repo
        self._settings = settings

    def oauth_login(self, provider: str, id_token_str: str) -> User:
        """
        OAuth 登录（通用方法）

        Args:
            provider: OAuth 提供商（google, github 等）
            id_token_str: OAuth id_token

        Returns:
            用户实体

        Raises:
            BusinessError: 验证失败或配置错误
        """
        if provider == OAuthProvider.GOOGLE:
            return self._google_login(id_token_str)
        raise BusinessError(f"不支持的 OAuth 提供商: {provider}", status_code=400)

    def _google_login(self, id_token_str: str) -> User:
        """Google OAuth 登录"""
        google_client_id = self._settings.google.client_id
        assert google_client_id, "google.client_id 未配置"

        # 验证 id_token
        try:
            id_info = google_id_token.verify_oauth2_token(
                id_token_str,
                google_requests.Request(),
                google_client_id,
                clock_skew_in_seconds=10,
            )
        except ValueError as e:
            raise BusinessError(f"Google token 无效: {e}", status_code=400) from e

        provider_id = id_info["sub"]
        email = id_info.get("email", "")
        if not email:
            raise BusinessError("Google 账号未提供邮箱, 无法注册", status_code=400)

        name = id_info.get("name", "") or id_info.get("given_name", "") or email.split("@")[0]

        # 查找或创建用户
        user = self._user_repo.find_by_oauth_account(OAuthProvider.GOOGLE, provider_id)
        if user is None:
            # 尝试通过邮箱关联已有账号
            user = self._user_repo.find_by_email(email)
            if user is not None:
                user.oauth_provider = OAuthProvider.GOOGLE
                user.oauth_provider_id = provider_id
                user.auth_source = AuthSource.OAUTH
                self._user_repo.save(user)
                logger.info(f"OAuth account linked: provider={OAuthProvider.GOOGLE}, user_id={user.id}, email={email}")
                return user

        if user is None:
            # 创建新用户
            base_username = name[:20] if name else "user"
            username = base_username
            suffix = 1
            max_retries = 100  # 防止无限循环
            while suffix <= max_retries and self._user_repo.find_by_username(username) is not None:
                username = f"{base_username}{suffix}"
                suffix += 1

            if suffix > max_retries:
                raise BusinessError("无法生成唯一用户名, 请联系管理员", status_code=500)

            user = User(
                id=str(ulid.ULID()),
                username=username,
                password_hash=hash_password(""),  # OAuth 用户无密码
                oauth_provider=OAuthProvider.GOOGLE,
                oauth_provider_id=provider_id,
                email=email,
                auth_source=AuthSource.OAUTH,
            )
            self._user_repo.save(user)
            logger.info(f"New OAuth user created: provider={OAuthProvider.GOOGLE}, user_id={user.id}, email={email}")

        return user
