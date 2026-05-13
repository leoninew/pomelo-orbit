"""认证应用服务"""

import logging

from google.auth.transport import requests as google_requests
from google.oauth2 import id_token as google_id_token
from ulid import ULID

from pomelo_orbit.application.auth.dtos import LoginReq, LoginResp
from pomelo_orbit.domain import AuthenticationError, BusinessError
from pomelo_orbit.domain.auth.constants import AuthSource, OAuthProvider
from pomelo_orbit.domain.auth.entities import LoginAttempt, LoginHistory, User
from pomelo_orbit.domain.auth.repositories import LoginAttemptRepository
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import SecurityService, hash_password, verify_password
from pomelo_orbit.infrastructure.csrf import verify_csrf_token
from pomelo_orbit.infrastructure.time_utils import utc_now

logger = logging.getLogger(__name__)


class AuthService:
    """认证应用服务"""

    def __init__(
        self,
        user_repo: UserRepository,
        login_attempt_repo: LoginAttemptRepository,
        security_service: SecurityService,
    ):
        self._user_repo = user_repo
        self._login_attempt_repo = login_attempt_repo
        self._security_service = security_service

    def login(self, cmd: LoginReq) -> LoginResp:
        """
        用户登录

        Args:
            cmd: 登录请求

        Returns:
            登录响应（包含 access_token）

        Raises:
            BusinessError: CSRF Token 无效或速率限制
            AuthenticationError: 用户名或密码错误
        """
        # 1. 验证 CSRF Token
        if not verify_csrf_token(cmd.csrf_token, self._security_service.settings.jwt.secret_key):
            raise BusinessError("请求令牌无效或已过期, 请刷新页面重试", status_code=400)

        # 2. 检查速率限制
        self._check_rate_limit(cmd.ip_address, cmd.username)

        # 3. 验证用户名和密码
        user = self._user_repo.find_by_username(cmd.username)
        if user is None or not verify_password(cmd.password, user.password_hash):
            # 记录失败尝试（立即提交，防止事务回滚）
            self._record_login_attempt_immediately(
                username=cmd.username,
                ip_address=cmd.ip_address,
                user_agent=cmd.user_agent,
                success=False,
            )
            logger.warning(f"Login failed: username={cmd.username}, ip={cmd.ip_address}")
            raise AuthenticationError("用户名或密码错误")

        # 3.1. 检查账号状态
        if not user.is_active:
            # 记录失败尝试（账号已禁用）
            self._record_login_attempt_immediately(
                username=cmd.username,
                ip_address=cmd.ip_address,
                user_agent=cmd.user_agent,
                success=False,
            )
            logger.warning(
                f"Login blocked: account disabled, user_id={user.id}, username={cmd.username}, ip={cmd.ip_address}"
            )
            raise BusinessError("账号已被禁用, 请联系管理员", status_code=403)

        # 4. 记录成功尝试
        attempt = LoginAttempt(
            id=str(ULID()),
            username=cmd.username,
            ip_address=cmd.ip_address,
            user_agent=cmd.user_agent,
            success=True,
        )
        self._login_attempt_repo.save(attempt)

        # 4.1. 清理14天前的登录尝试记录
        self._login_attempt_repo.delete_old_records(14)

        # 5. 更新最后登录时间
        user.last_login_at = utc_now()
        self._user_repo.save(user)

        # 6. 记录登录历史
        login_history = LoginHistory(
            id=str(ULID()),
            user_id=user.id,
            username=user.username,
            ip_address=cmd.ip_address,
            user_agent=cmd.user_agent,
            login_at=utc_now(),
            success=True,
        )
        self._user_repo.save_login_history(login_history)

        # 7. 生成 token
        access_token = self._security_service.create_access_token(data={"sub": user.username})
        logger.info(f"Login successful: username={user.username}, ip={cmd.ip_address}")

        return LoginResp(access_token=access_token)

    def _check_rate_limit(self, ip_address: str, username: str) -> None:
        """
        检查速率限制（简化规则：从第3次开始，每分钟只允许尝试一次）

        Args:
            ip_address: 客户端 IP
            username: 用户名

        Raises:
            BusinessError: 超过速率限制
        """
        # 从配置读取参数
        window_minutes = self._security_service.settings.rate_limit.window_minutes
        failure_threshold = self._security_service.settings.rate_limit.failure_threshold
        retry_interval_seconds = self._security_service.settings.rate_limit.retry_interval_seconds

        # IP 级别：窗口内失败 >= 阈值，强制等待间隔
        ip_fail_count, last_ip_attempt = self._login_attempt_repo.get_failed_attempts_summary_by_ip(
            ip_address, window_minutes
        )
        if ip_fail_count >= failure_threshold and last_ip_attempt:
            elapsed = (utc_now() - last_ip_attempt.created_at).total_seconds()
            if elapsed < retry_interval_seconds:
                remaining = int(retry_interval_seconds - elapsed)
                logger.warning(
                    f"Login rate limit exceeded: ip={ip_address}, fail_count={ip_fail_count}, "
                    f"elapsed={int(elapsed)}s, remaining={remaining}s"
                )
                raise BusinessError(f"操作过于频繁, 请 {remaining} 秒后再试", status_code=429)

        # 账号级别：窗口内失败 >= 阈值，强制等待间隔
        username_fail_count, last_username_attempt = self._login_attempt_repo.get_failed_attempts_summary_by_username(
            username, window_minutes
        )
        if username_fail_count >= failure_threshold and last_username_attempt:
            elapsed = (utc_now() - last_username_attempt.created_at).total_seconds()
            if elapsed < retry_interval_seconds:
                remaining = int(retry_interval_seconds - elapsed)
                logger.warning(
                    f"Login rate limit exceeded: username={username}, fail_count={username_fail_count}, "
                    f"elapsed={int(elapsed)}s, remaining={remaining}s"
                )
                raise BusinessError(f"该账号登录失败次数过多, 请 {remaining} 秒后再试", status_code=429)

    def _record_login_attempt_immediately(
        self,
        username: str,
        ip_address: str,
        user_agent: str | None,
        success: bool,
    ) -> None:
        """
        记录登录尝试并立即提交（防止事务回滚）

        Args:
            username: 用户名
            ip_address: 客户端 IP
            user_agent: User-Agent
            success: 是否成功
        """
        attempt = LoginAttempt(
            id=str(ULID()),
            username=username,
            ip_address=ip_address,
            user_agent=user_agent,
            success=success,
        )
        self._login_attempt_repo.save_and_commit(attempt)

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
        if provider == "google":
            return self._google_login(id_token_str)
        raise BusinessError(f"不支持的 OAuth 提供商: {provider}", status_code=400)

    def _google_login(self, id_token_str: str) -> User:
        """Google OAuth 登录"""
        google_client_id = self._security_service.settings.google.client_id
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
                id=str(ULID()),
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


__all__ = ["AuthService"]
