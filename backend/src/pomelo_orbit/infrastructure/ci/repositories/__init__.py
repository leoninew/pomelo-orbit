"""CI Repository 实现"""

from pomelo_orbit.infrastructure.ci.repositories.artifact import ArtifactRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.credential import CredentialRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.pipeline_run import PipelineRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.project import ProjectRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.stage_run import StageLogRepositoryImpl, StageRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.template import (
    PipelineSnapshotRepositoryImpl,
    PipelineStageRepositoryImpl,
    PipelineTemplateRepositoryImpl,
)
from pomelo_orbit.infrastructure.ci.repositories.webhook import ProjectWebhookRepositoryImpl

__all__ = [
    "ArtifactRepositoryImpl",
    "CredentialRepositoryImpl",
    "PipelineRunRepositoryImpl",
    "PipelineSnapshotRepositoryImpl",
    "PipelineStageRepositoryImpl",
    "PipelineTemplateRepositoryImpl",
    "ProjectRepositoryImpl",
    "ProjectWebhookRepositoryImpl",
    "StageLogRepositoryImpl",
    "StageRunRepositoryImpl",
]
