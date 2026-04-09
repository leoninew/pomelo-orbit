"""项目相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel

from pomelo_orbit.domain.ci.entities import Repository


class RepositoryResp(BaseModel):
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
    def from_domain(cls, repository: Repository, git_credential_name: str | None = None) -> "RepositoryResp":
        return cls(
            id=repository.id,
            name=repository.name,
            code=repository.code,
            repository_url=repository.repository_url,
            git_credential_id=repository.git_credential_id,
            git_credential_name=git_credential_name,
            variable_overrides=repository.variable_overrides,
            default_branch=repository.default_branch,
            created_at=repository.created_at,
            updated_at=repository.updated_at,
        )


class RepositoryCreateReq(BaseModel):
    name: str
    code: str
    repository_url: str
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] = {}
    default_branch: str = "master"


class RepositoryUpdateReq(BaseModel):
    name: str | None = None
    repository_url: str | None = None
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] | None = None
    default_branch: str | None = None
