"""项目相关 DTO"""

from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.domain.ci.value_objects import Variable, VariableDeclaration


class RepositoryListResp(BaseModel):
    """Repository 列表响应（简化版）"""

    id: str
    name: str
    code: str
    repository_url: str
    has_credential: bool
    git_credential_id: str | None
    default_branch: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}

    @classmethod
    def from_domain(cls, repository: Repository) -> "RepositoryListResp":
        return cls(
            id=repository.id,
            name=repository.name,
            code=repository.code,
            repository_url=repository.repository_url,
            has_credential=repository.git_credential_id is not None,
            git_credential_id=repository.git_credential_id,
            default_branch=repository.default_branch,
            created_at=repository.created_at,
            updated_at=repository.updated_at,
        )


class RepositoryResp(BaseModel):
    """Repository 详情响应"""

    id: str
    name: str
    code: str
    repository_url: str
    git_credential_id: str | None
    git_credential_name: str | None
    variables: list[Variable]  # 变量列表（包含内置和自定义）
    default_branch: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}

    @classmethod
    def from_domain(
        cls,
        repository: Repository,
        git_credential_name: str | None,
        variables: list[Variable],
    ) -> "RepositoryResp":
        return cls(
            id=repository.id,
            name=repository.name,
            code=repository.code,
            repository_url=repository.repository_url,
            git_credential_id=repository.git_credential_id,
            git_credential_name=git_credential_name,
            variables=variables,
            default_branch=repository.default_branch,
            created_at=repository.created_at,
            updated_at=repository.updated_at,
        )


class RepositoryCreateReq(BaseModel):
    name: str = Field(min_length=1)
    code: str = Field(min_length=1, pattern=r"^[a-z0-9_-]+$")
    repository_url: str = Field(min_length=1)
    git_credential_id: str | None = None
    variable_overrides: list[VariableDeclaration] = Field(default_factory=list)
    default_branch: str = Field(default="master", min_length=1)


class RepositoryUpdateReq(BaseModel):
    name: str | None = Field(default=None, min_length=1)
    repository_url: str | None = Field(default=None, min_length=1)
    git_credential_id: str | None = None
    variable_overrides: list[VariableDeclaration] | None = None
    default_branch: str | None = Field(default=None, min_length=1)
