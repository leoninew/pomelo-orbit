"""
Docker 基础设施
"""

# 避免循环导入：不在这里导入 ApplicationManagerImpl
from pomelo_orbit.infrastructure.docker.path_resolver import (
    detect_container_id,
    get_container_mount_source,
)

__all__ = [
    "detect_container_id",
    "get_container_mount_source",
]
