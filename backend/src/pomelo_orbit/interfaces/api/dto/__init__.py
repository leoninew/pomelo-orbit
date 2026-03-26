"""
DTO (Data Transfer Object) 定义
"""

from pomelo_orbit.interfaces.api.dto.application import (
    ApplicationCreateReq,
    ApplicationResp,
    ApplicationUpdateReq,
    ConfigFileReq,
    ConfigFileResp,
    GitSourceReq,
    GitSourceResp,
    ImageSourceReq,
    ImageSourceResp,
)
from pomelo_orbit.interfaces.api.dto.auth import (
    LoginHistoryResp,
    LoginReq,
    PasswordChangeReq,
    TokenResp,
    UserInfo,
)
from pomelo_orbit.interfaces.api.dto.common import MessageResp, PaginatedResp
from pomelo_orbit.interfaces.api.dto.deployment import (
    DeploymentDetailResp,
    DeploymentResp,
)
from pomelo_orbit.interfaces.api.dto.event import (
    WebhookEventDetailResp,
    WebhookEventResp,
)
from pomelo_orbit.interfaces.api.dto.route import (
    RouteCreateReq,
    RouteResp,
    RouteUpdateReq,
)
from pomelo_orbit.interfaces.api.dto.setting import (
    SystemConfigResetReq,
    SystemConfigResp,
    SystemConfigUpdateReq,
)

# Rebuild models to resolve forward references
ApplicationResp.model_rebuild()

__all__ = [
    "ApplicationCreateReq",
    "ApplicationResp",
    "ApplicationUpdateReq",
    "ConfigFileReq",
    "ConfigFileResp",
    "DeploymentDetailResp",
    "DeploymentResp",
    "GitSourceReq",
    "GitSourceResp",
    "ImageSourceReq",
    "ImageSourceResp",
    "LoginHistoryResp",
    "LoginReq",
    "MessageResp",
    "PaginatedResp",
    "PasswordChangeReq",
    "RouteCreateReq",
    "RouteResp",
    "RouteUpdateReq",
    "SystemConfigResetReq",
    "SystemConfigResp",
    "SystemConfigUpdateReq",
    "TokenResp",
    "UserInfo",
    "WebhookEventDetailResp",
    "WebhookEventResp",
]
