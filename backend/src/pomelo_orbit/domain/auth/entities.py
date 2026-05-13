"""共享领域实体 - 跨模块使用的实体"""

from dataclasses import dataclass, field
from datetime import datetime

from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass
class User:
    """用户实体"""

    id: str
    username: str
    password_hash: str
    oauth_provider: str = ""  # OAuth 提供商（google, github 等）
    oauth_provider_id: str = ""  # OAuth 提供商的用户 ID
    email: str | None = None
    auth_source: str = "password"  # password | oauth
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)
    last_login_at: datetime | None = None


@dataclass
class LoginHistory:
    """登录历史"""

    id: str
    user_id: str
    username: str
    success: bool
    ip_address: str | None = None
    user_agent: str | None = None
    login_at: datetime = field(default_factory=utc_now)


@dataclass
class LoginAttempt:
    """登录尝试记录（用于速率限制）"""

    id: str
    username: str | None  # 可为空，因为用户名可能不存在
    ip_address: str
    success: bool
    user_agent: str | None = None
    created_at: datetime = field(default_factory=utc_now)
