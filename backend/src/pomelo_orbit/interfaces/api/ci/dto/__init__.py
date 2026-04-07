"""CI 模块 DTO"""

# 重新导出所有子模块的 DTO
from pomelo_orbit.interfaces.api.ci.dto.credential import (
    CredentialCreateReq,
    CredentialResp,
    CredentialUpdateReq,
)
from pomelo_orbit.interfaces.api.ci.dto.project import ProjectCreateReq, ProjectResp, ProjectUpdateReq
from pomelo_orbit.interfaces.api.ci.dto.run import ArtifactResp, PipelineRunResp, TriggerPipelineReq
from pomelo_orbit.interfaces.api.ci.dto.stage_run import StageLogResp, StageRunResp
from pomelo_orbit.interfaces.api.ci.dto.template import (
    PipelineSnapshotListItemResp,
    PipelineSnapshotResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
    VariableDeclarationDto,
)

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
    "ProjectCreateReq",
    "ProjectResp",
    "ProjectUpdateReq",
    "StageLogResp",
    "StageRunResp",
    "TriggerPipelineReq",
    "VariableDeclarationDto",
]
