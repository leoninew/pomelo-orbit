"""CI 模块 DTO"""

# 重新导出所有子模块的 DTO
from pomelo_orbit.interfaces.api.ci.dto.credential import (
    CredentialCreateReq,
    CredentialResp,
    CredentialUpdateReq,
)
from pomelo_orbit.interfaces.api.ci.dto.pipeline_run import ArtifactResp, PipelineRunResp, TriggerPipelineReq
from pomelo_orbit.interfaces.api.ci.dto.pipeline_stage_run import StageRunResp
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import (
    PipelineSnapshotListItemResp,
    PipelineSnapshotResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
    VariableDeclarationDto,
)
from pomelo_orbit.interfaces.api.ci.dto.repository import RepositoryCreateReq, RepositoryResp, RepositoryUpdateReq

__all__ = [
    "ArtifactResp",
    "CredentialCreateReq",
    "CredentialResp",
    "CredentialUpdateReq",
    "PipelineRunResp",
    "PipelineSnapshotListItemResp",
    "PipelineSnapshotResp",
    "PipelineTemplateCreateReq",
    "PipelineTemplateResp",
    "PipelineTemplateUpdateReq",
    "RepositoryCreateReq",
    "RepositoryResp",
    "RepositoryUpdateReq",
    "StageRunResp",
    "TriggerPipelineReq",
    "VariableDeclarationDto",
]
