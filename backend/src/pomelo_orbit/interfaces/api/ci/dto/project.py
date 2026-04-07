"""项目相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel

from pomelo_orbit.domain.ci.entities import Project


class ProjectResp(BaseModel):
    id: str
    name: str
    code: str
    repository_url: str
    git_credential_id: str | None
    git_credential_name: str | None
    variable_overrides: dict[str, Any]
    default_branch: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}

    @classmethod
    def from_domain(cls, project: Project, git_credential_name: str | None = None) -> "ProjectResp":
        return cls(
            id=project.id,
            name=project.name,
            code=project.code,
            repository_url=project.repository_url,
            git_credential_id=project.git_credential_id,
            git_credential_name=git_credential_name,
            variable_overrides=project.variable_overrides,
            default_branch=project.default_branch,
            created_at=project.created_at,
            updated_at=project.updated_at,
        )


class ProjectCreateReq(BaseModel):
    name: str
    code: str
    repository_url: str
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] = {}
    default_branch: str = "master"


class ProjectUpdateReq(BaseModel):
    name: str | None = None
    repository_url: str | None = None
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] | None = None
    default_branch: str | None = None
