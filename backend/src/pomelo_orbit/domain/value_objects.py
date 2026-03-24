"""
领域值对象
"""

from enum import StrEnum


class ApplicationStatus(StrEnum):
    """应用状态"""

    STARTED = "started"
    STOPPED = "stopped"


class OperationType(StrEnum):
    """部署操作类型"""

    DEPLOY = "deploy"
    STOP = "stop"
    RESTART = "restart"


class ImagePullPolicy(StrEnum):
    """镜像拉取策略"""

    ALWAYS = "always"
    MISSING = "missing"
    NEVER = "never"


class DeployStatus(StrEnum):
    """部署状态"""

    QUEUED = "queued"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"


__all__ = ["ApplicationStatus", "DeployStatus", "ImagePullPolicy", "OperationType"]
