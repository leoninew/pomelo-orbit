"""CI 模块 DTO"""

# 重新导出所有子模块的 DTO
from pomelo_orbit.interfaces.api.ci.dto.credential import (
    CredentialCreateReq,
    CredentialResp,
    CredentialUpdateReq,
)
from pomelo_orbit.interfaces.api.ci.dto.job import JobLogResp, JobResp
from pomelo_orbit.interfaces.api.ci.dto.project import ProjectCreateReq, ProjectResp, ProjectUpdateReq
from pomelo_orbit.interfaces.api.ci.dto.run import ArtifactResp, PipelineRunResp, TriggerPipelineReq
from pomelo_orbit.interfaces.api.ci.dto.template import (
    PipelineSnapshotListItemResp,
    PipelineSnapshotResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
    StageDefinitionDto,
    VariableDeclarationDto,
)

__all__ = [
    "ArtifactResp",
    "CredentialCreateReq",
    "CredentialResp",
    "CredentialUpdateReq",
    "JobLogResp",
    "JobResp",
    "PipelineRunResp",
    "PipelineSnapshotListItemResp",
    "PipelineSnapshotResp",
    "PipelineTemplateCreateReq",
    "PipelineTemplateResp",
    "PipelineTemplateUpdateReq",
    "ProjectCreateReq",
    "ProjectResp",
    "ProjectUpdateReq",
    "StageDefinitionDto",
    "TriggerPipelineReq",
    "VariableDeclarationDto",
]
