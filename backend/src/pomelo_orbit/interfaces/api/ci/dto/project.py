"""项目相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel


class ProjectResp(BaseModel):
    id: str
    name: str
    repository_url: str
    pipeline_snapshot_id: str
    git_credential_id: str | None
    variable_overrides: dict[str, Any]
    default_branch: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ProjectCreateReq(BaseModel):
    name: str
    repository_url: str
    pipeline_snapshot_id: str
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] = {}
    default_branch: str = "master"


class ProjectUpdateReq(BaseModel):
    name: str | None = None
    repository_url: str | None = None
    pipeline_snapshot_id: str | None = None
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] | None = None
    default_branch: str | None = None
