"""CI 领域仓储抽象接口"""

from pomelo_orbit.domain.ci.repositories.artifact import ArtifactRepository
from pomelo_orbit.domain.ci.repositories.credential import CredentialRepository
from pomelo_orbit.domain.ci.repositories.job import JobLogRepository, JobRepository
from pomelo_orbit.domain.ci.repositories.pipeline_run import PipelineRunRepository
from pomelo_orbit.domain.ci.repositories.project import ProjectRepository
from pomelo_orbit.domain.ci.repositories.template import PipelineTemplateRepository

__all__ = [
    "ArtifactRepository",
    "CredentialRepository",
    "JobLogRepository",
    "JobRepository",
    "PipelineRunRepository",
    "PipelineTemplateRepository",
    "ProjectRepository",
]
