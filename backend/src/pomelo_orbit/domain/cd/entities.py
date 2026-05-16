"""
领域实体定义
"""

from dataclasses import dataclass, field
from datetime import datetime
from enum import StrEnum

from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, OperationType
from pomelo_orbit.infrastructure.time_utils import utc_now


class TriggerType(StrEnum):
    """部署触发类型"""

    MANUAL = "manual"


class CertType(StrEnum):
    """证书类型"""

    MANUAL = "manual"
    LETSENCRYPT = "letsencrypt"
    MKCERT = "mkcert"


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
class ApplicationRoute:
    """应用路由配置"""

    id: str
    application_id: str
    service_name: str
    domain: str
    port: int
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class ApplicationServiceConfig:
    """应用 service 级配置"""

    id: str
    application_id: str
    service_name: str
    image: str | None = None
    environment: str | None = None
    volumes: str | None = None
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)


@dataclass
class Application:
    """应用实体（聚合根）"""

    id: str
    project_id: str
    name: str
    code: str
    image_pull_policy: str
    status: str
    route_managed: bool = False
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    config_files: list[ApplicationConfigFile] = field(default_factory=list)
    routes: list[ApplicationRoute] = field(default_factory=list)

    def can_deploy(self) -> bool:
        return self.status != ApplicationStatus.DEPLOYING

    def can_stop(self) -> bool:
        return self.status == ApplicationStatus.DEPLOYED

    def can_restart(self) -> bool:
        return self.status == ApplicationStatus.DEPLOYED

    def mark_as_deploying(self) -> None:
        self.status = ApplicationStatus.DEPLOYING
        self.updated_at = utc_now()

    def mark_as_deployed(self) -> None:
        self.status = ApplicationStatus.DEPLOYED
        self.updated_at = utc_now()

    def mark_as_deploy_failed(self) -> None:
        self.status = ApplicationStatus.DEPLOY_FAILED
        self.updated_at = utc_now()

    def mark_as_undeployed(self) -> None:
        self.status = ApplicationStatus.UNDEPLOYED
        self.updated_at = utc_now()


@dataclass
class Deployment:
    """部署记录实体"""

    id: str
    project_id: str
    application_id: str | None
    application_name: str
    trigger_type: TriggerType
    status: str
    operation_type: OperationType
    is_rollback: bool
    started_at: datetime = field(default_factory=utc_now)
    finished_at: datetime | None = None
    duration_ms: int | None = None
    log_text: str | None = None
    error_message: str | None = None
    rollback_from_deployment_id: str | None = None


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
    cert_pem: str | None = None
    cert_key: str | None = None
    cert_type: CertType = CertType.MANUAL
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    def enable(self) -> None:
        self.enabled = True
        self.updated_at = utc_now()

    def disable(self) -> None:
        self.enabled = False
        self.updated_at = utc_now()

    def enable_https(self, cert_pem: str, cert_key: str) -> None:
        self.https_enabled = True
        self.cert_pem = cert_pem
        self.cert_key = cert_key
        self.cert_type = CertType.MANUAL
        self.updated_at = utc_now()

    def enable_letsencrypt(self) -> None:
        self.https_enabled = True
        self.cert_pem = None
        self.cert_key = None
        self.cert_type = CertType.LETSENCRYPT
        self.updated_at = utc_now()

    def enable_mkcert(self, cert_pem: str, cert_key: str) -> None:
        self.https_enabled = True
        self.cert_pem = cert_pem
        self.cert_key = cert_key
        self.cert_type = CertType.MKCERT
        self.updated_at = utc_now()

    def disable_https(self) -> None:
        self.https_enabled = False
        self.cert_pem = None
        self.cert_key = None
        self.cert_type = CertType.MANUAL
        self.updated_at = utc_now()

    def can_use_letsencrypt(self) -> tuple[bool, str]:
        if self.domain == "localhost" or self.domain.endswith(".local"):
            return False, "Let's Encrypt 不支持内网域名,请使用手动证书或 mkcert"
        return True, ""


__all__ = [
    "Application",
    "ApplicationConfigFile",
    "ApplicationRoute",
    "ApplicationServiceConfig",
    "CertType",
    "Deployment",
    "Route",
    "TriggerType",
]
