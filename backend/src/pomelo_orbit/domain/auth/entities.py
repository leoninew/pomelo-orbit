"""共享领域实体 - 跨模块使用的实体"""

from dataclasses import dataclass, field
from datetime import datetime

from pomelo_orbit.infrastructure.time_utils import utc_now


# 用户状态常量
USER_STATUS_ENABLED = "enabled"
USER_STATUS_DISABLED = "disabled"
VALID_USER_STATUSES = {USER_STATUS_ENABLED, USER_STATUS_DISABLED}


@dataclass
class User:
    """用户实体"""

    id: str
    username: str
    password_hash: str
    status: str
    oauth_provider: str
    oauth_provider_id: str
    email: str | None
    auth_source: str
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)
    last_login_at: datetime | None = None

    def __post_init__(self):
        """验证状态字段"""
        if self.status not in VALID_USER_STATUSES:
            raise ValueError(f"Invalid user status: {self.status}. Must be one of {VALID_USER_STATUSES}")


@dataclass
class Role:
    id: str
    code: str
    name: str
    description: str | None
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class Permission:
    id: str
    code: str
    name: str
    description: str | None
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


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
