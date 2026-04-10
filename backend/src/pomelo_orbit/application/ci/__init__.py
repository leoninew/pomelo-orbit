"""CI 应用服务层

按照 DDD 聚合根划分的应用服务：
- CredentialService: Credential 聚合
- StageService: Stage 聚合
- TemplateService: Template 聚合（包含 Snapshot）
- RepositoryService: Repository 聚合
- WebhookService: Webhook 聚合
- PipelineRunService: PipelineRun 聚合（包含 StageRun、Artifact）
"""

from pomelo_orbit.application.ci.credential_service import CredentialService
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.application.ci.repository_service import RepositoryService
from pomelo_orbit.application.ci.stage_service import StageService
from pomelo_orbit.application.ci.template_service import TemplateService
from pomelo_orbit.application.ci.webhook_service import WebhookService

__all__ = [
    "CredentialService",
    "PipelineRunService",
    "RepositoryService",
    "StageService",
    "TemplateService",
    "WebhookService",
]
