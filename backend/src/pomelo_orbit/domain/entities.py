"""
领域实体定义
"""

from dataclasses import dataclass, field
from datetime import datetime
from enum import StrEnum

from pomelo_orbit.domain.value_objects import ApplicationStatus, OperationType
from pomelo_orbit.infrastructure.time_utils import utc_now


class SourceType(StrEnum):
    """应用源类型"""

    GIT = "git"
    IMAGE = "image"


class TriggerType(StrEnum):
    """部署触发类型"""

    WEBHOOK = "webhook"
    MANUAL = "manual"


class WebhookSource(StrEnum):
    """Webhook 来源"""

    GITHUB = "github"


class CertType(StrEnum):
    """证书类型"""

    MANUAL = "manual"
    LETSENCRYPT = "letsencrypt"
    MKCERT = "mkcert"


class WebhookEventType(StrEnum):
    """回调事件类型"""

    PUSH = "push"
    RELEASE = "release"
    PING = "ping"


class WebhookEventStatus(StrEnum):
    """回调事件状态"""

    RECEIVED = "received"
    MATCHED = "matched"
    IGNORED = "ignored"
    ERROR = "error"


@dataclass
class GitSource:
    """Git 仓库源"""

    id: str
    application_id: str
    repository_url: str
    deploy_branches: str
    auto_deploy: bool
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class ImageSource:
    """镜像源"""

    id: str
    application_id: str
    image_name: str
    registry_url: str | None = None
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class ApplicationConfigFile:
    """应用配置文件"""

    id: str
    application_id: str
    path: str
    content: str
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class Application:
    """应用实体（聚合根）"""

    id: str
    name: str
    code: str
    image_pull_policy: str
    enabled: bool
    status: str
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    # 关联实体
    git_source: GitSource | None = None
    image_source: ImageSource | None = None
    config_files: list[ApplicationConfigFile] = field(default_factory=list)

    def can_deploy(self) -> bool:
        """检查是否可以部署"""
        return self.enabled and self.status != ApplicationStatus.STARTED

    def can_stop(self) -> bool:
        """检查是否可以停止"""
        return self.status == ApplicationStatus.STARTED

    def can_restart(self) -> bool:
        """检查是否可以重启"""
        return self.status == ApplicationStatus.STARTED

    def mark_as_started(self) -> None:
        """标记为已启动"""
        self.status = ApplicationStatus.STARTED
        self.updated_at = utc_now()

    def mark_as_stopped(self) -> None:
        """标记为已停止"""
        self.status = ApplicationStatus.STOPPED
        self.updated_at = utc_now()


@dataclass
class Deployment:
    """部署记录实体"""

    id: str
    application_id: str | None
    application_name: str
    trigger_type: TriggerType
    status: str
    operation_type: OperationType
    is_rollback: bool
    trigger_ref: str | None = None
    webhook_event_id: str | None = None
    image_name: str | None = None
    env_file: str | None = None
    started_at: datetime = field(default_factory=utc_now)
    finished_at: datetime | None = None
    duration_ms: int | None = None
    log_text: str | None = None
    error_message: str | None = None
    rollback_from_deployment_id: str | None = None


@dataclass
class WebhookEvent:
    """回调事件记录"""

    id: str
    source: WebhookSource
    event_type: WebhookEventType
    status: WebhookEventStatus = WebhookEventStatus.RECEIVED
    repository_name: str | None = None
    repository_url: str | None = None
    branch: str | None = None
    sender: str | None = None
    image_name: str | None = None
    payload: str | None = None
    signature_valid: bool | None = None
    matched_application_id: str | None = None
    triggered_deployment_id: str | None = None
    error_message: str | None = None
    received_at: datetime = field(default_factory=utc_now)
    processed_at: datetime | None = None


@dataclass
class User:
    """用户实体"""

    id: str
    username: str
    password_hash: str
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
class Route:
    """路由配置实体"""

    id: str
    name: str
    domain: str
    path_prefix: str
    target_url: str
    enabled: bool
    https_enabled: bool
    cert_pem: str | None = None  # 证书内容（PEM 格式）
    cert_key: str | None = None  # 私钥内容（PEM 格式）
    cert_type: CertType = CertType.MANUAL  # 证书类型
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    def enable(self) -> None:
        """启用路由"""
        self.enabled = True
        self.updated_at = utc_now()

    def disable(self) -> None:
        """停用路由"""
        self.enabled = False
        self.updated_at = utc_now()

    def enable_https(self, cert_pem: str, cert_key: str) -> None:
        """启用 HTTPS（手动证书）"""
        self.https_enabled = True
        self.cert_pem = cert_pem
        self.cert_key = cert_key
        self.cert_type = CertType.MANUAL
        self.updated_at = utc_now()

    def enable_letsencrypt(self) -> None:
        """启用 HTTPS（Let's Encrypt 自动证书）"""
        self.https_enabled = True
        self.cert_pem = None
        self.cert_key = None
        self.cert_type = CertType.LETSENCRYPT
        self.updated_at = utc_now()

    def enable_mkcert(self, cert_pem: str, cert_key: str) -> None:
        """启用 HTTPS（mkcert 本地证书）"""
        self.https_enabled = True
        self.cert_pem = cert_pem
        self.cert_key = cert_key
        self.cert_type = CertType.MKCERT
        self.updated_at = utc_now()

    def disable_https(self) -> None:
        """禁用 HTTPS"""
        self.https_enabled = False
        self.cert_pem = None
        self.cert_key = None
        self.cert_type = CertType.MANUAL
        self.updated_at = utc_now()

    def can_use_letsencrypt(self) -> tuple[bool, str]:
        """检查是否可以使用 Let's Encrypt

        Returns:
            (can_use, reason): 是否可用和原因
        """
        # 检查 localhost 域名
        if self.domain == "localhost" or self.domain.endswith(".local"):
            return False, "Let's Encrypt 不支持内网域名,请使用手动证书或 mkcert"
        return True, ""


__all__ = [
    "Application",
    "ApplicationConfigFile",
    "CertType",
    "Deployment",
    "GitSource",
    "ImageSource",
    "LoginHistory",
    "Route",
    "SourceType",
    "TriggerType",
    "User",
    "WebhookEvent",
    "WebhookEventStatus",
    "WebhookEventType",
    "WebhookSource",
]
