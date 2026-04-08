"""CI 领域仓储抽象接口"""

from pomelo_orbit.domain.ci.repositories.artifact import ArtifactRepository
from pomelo_orbit.domain.ci.repositories.credential import CredentialRepository
from pomelo_orbit.domain.ci.repositories.pipeline_run import PipelineRunRepository
from pomelo_orbit.domain.ci.repositories.project import ProjectRepository
from pomelo_orbit.domain.ci.repositories.stage_run import StageRunRepository
from pomelo_orbit.domain.ci.repositories.template import (
    PipelineSnapshotRepository,
    PipelineStageRepository,
    PipelineTemplateRepository,
)
from pomelo_orbit.domain.ci.repositories.webhook import ProjectWebhookRepository

__all__ = [
    "ArtifactRepository",
    "CredentialRepository",
    "PipelineRunRepository",
    "PipelineSnapshotRepository",
    "PipelineStageRepository",
    "PipelineTemplateRepository",
    "ProjectRepository",
    "ProjectWebhookRepository",
    "StageRunRepository",
]
