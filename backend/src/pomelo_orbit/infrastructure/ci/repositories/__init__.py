"""CI Repository 实现"""

from pomelo_orbit.infrastructure.ci.repositories.artifact import ArtifactRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.credential import CredentialRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.pipeline_run import PipelineRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.repository import RepositoryRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.stage_run import StageRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.repositories.template import (
    PipelineSnapshotRepositoryImpl,
    PipelineStageRepositoryImpl,
    PipelineTemplateRepositoryImpl,
)
from pomelo_orbit.infrastructure.ci.repositories.webhook import RepositoryWebhookRepositoryImpl

__all__ = [
    "ArtifactRepositoryImpl",
    "CredentialRepositoryImpl",
    "PipelineRunRepositoryImpl",
    "PipelineSnapshotRepositoryImpl",
    "PipelineStageRepositoryImpl",
    "PipelineTemplateRepositoryImpl",
    "RepositoryRepositoryImpl",
    "RepositoryWebhookRepositoryImpl",
    "StageRunRepositoryImpl",
]
