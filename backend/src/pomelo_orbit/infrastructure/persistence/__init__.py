"""
Persistence Layer - 持久化层
包含数据库模型、仓储实现、映射器
"""

from pomelo_orbit.infrastructure.persistence.base_mapper import BaseMapper
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.mappers import (
    ApplicationConfigFileMapper,
    ApplicationMapper,
    CredentialMapper,
    DeploymentMapper,
    GitSourceMapper,
    ImageSourceMapper,
    LoginHistoryMapper,
    RouteMapper,
    UserMapper,
    WebhookEventMapper,
)
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
    Base,
    CredentialModel,
    DeploymentModel,
    GitSourceModel,
    ImageSourceModel,
    LoginHistoryModel,
    RouteModel,
    UserModel,
    WebhookEventModel,
)

__all__ = [
    "ApplicationConfigFileMapper",
    "ApplicationConfigFileModel",
    "ApplicationMapper",
    "ApplicationModel",
    "Base",
    "BaseMapper",
    "BaseRepository",
    "CredentialMapper",
    "CredentialModel",
    "DeploymentMapper",
    "DeploymentModel",
    "GitSourceMapper",
    "GitSourceModel",
    "ImageSourceMapper",
    "ImageSourceModel",
    "LoginHistoryMapper",
    "LoginHistoryModel",
    "RouteMapper",
    "RouteModel",
    "UserMapper",
    "UserModel",
    "WebhookEventMapper",
    "WebhookEventModel",
    "get_db",
    "get_session_factory",
]
