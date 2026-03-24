"""
Infrastructure 层依赖注入配置
"""

from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends

from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.security import SecurityService


def get_security_service(settings: Annotated[Dynaconf, Depends(get_settings)]) -> SecurityService:
    """获取安全服务实例"""
    return SecurityService(settings)
