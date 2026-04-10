"""
领域值对象
"""

from enum import StrEnum


class ApplicationStatus(StrEnum):
    """应用状态"""

    UNDEPLOYED = "undeployed"
    DEPLOYING = "deploying"
    DEPLOYED = "deployed"
    DEPLOY_FAILED = "deploy_failed"


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


class TaskStatus(StrEnum):
    """异步任务状态（CI PipelineRun、CD Deployment 统一使用）"""

    WAITING_TO_RUN = "waiting_to_run"
    RUNNING = "running"
    RAN_TO_COMPLETION = "ran_to_completion"
    FAULTED = "faulted"
    CANCELED = "canceled"


__all__ = ["ApplicationStatus", "ImagePullPolicy", "OperationType", "TaskStatus"]
