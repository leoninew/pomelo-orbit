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
    ApplicationServiceConfigMapper,
    DeploymentMapper,
    LoginHistoryMapper,
    PermissionMapper,
    RoleMapper,
    RouteMapper,
    UserMapper,
)
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
    ApplicationServiceConfigModel,
    Base,
    DeploymentModel,
    LoginHistoryModel,
    PermissionModel,
    RoleModel,
    RolePermissionModel,
    RouteModel,
    UserModel,
    UserRoleModel,
)

__all__ = [
    "ApplicationConfigFileMapper",
    "ApplicationConfigFileModel",
    "ApplicationMapper",
    "ApplicationModel",
    "ApplicationServiceConfigMapper",
    "ApplicationServiceConfigModel",
    "Base",
    "BaseMapper",
    "BaseRepository",
    "DeploymentMapper",
    "DeploymentModel",
    "LoginHistoryMapper",
    "LoginHistoryModel",
    "PermissionMapper",
    "PermissionModel",
    "RoleMapper",
    "RoleModel",
    "RolePermissionModel",
    "RouteMapper",
    "RouteModel",
    "UserMapper",
    "UserModel",
    "UserRoleModel",
    "get_db",
    "get_session_factory",
]
