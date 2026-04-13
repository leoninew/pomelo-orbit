"""CI 领域仓储抽象接口"""

from pomelo_orbit.domain.ci.repositories.artifact import ArtifactRepository
from pomelo_orbit.domain.ci.repositories.credential import CredentialRepository
from pomelo_orbit.domain.ci.repositories.pipeline_run import PipelineRunRepository
from pomelo_orbit.domain.ci.repositories.repository import RepositoryRepository
from pomelo_orbit.domain.ci.repositories.stage_run import StageRunRepository
from pomelo_orbit.domain.ci.repositories.template import (
    BuildStageRepository,
    PipelineSnapshotRepository,
    PipelineTemplateRepository,
)
from pomelo_orbit.domain.ci.repositories.webhook import RepositoryWebhookRepository

__all__ = [
    "ArtifactRepository",
    "BuildStageRepository",
    "CredentialRepository",
    "PipelineRunRepository",
    "PipelineSnapshotRepository",
    "PipelineTemplateRepository",
    "RepositoryRepository",
    "RepositoryWebhookRepository",
    "StageRunRepository",
]
