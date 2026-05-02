"""凭据相关 DTO"""

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field


class CredentialResp(BaseModel):
    id: str
    name: str
    type: str
    created_at: datetime

    model_config = {"from_attributes": True}


class CredentialCreateReq(BaseModel):
    name: str = Field(min_length=1)
    type: Literal["git_ssh", "git_token", "gitee_token", "registry_token"]
    data: str = Field(min_length=1)


class CredentialUpdateReq(BaseModel):
    name: str | None = Field(default=None, min_length=1)
    data: str | None = Field(default=None, min_length=1)


class CredentialExportResp(BaseModel):
    """凭据导出响应"""

    version: str = "1.0"
    name: str
    type: Literal["git_ssh", "git_token", "gitee_token", "registry_token"]
    data: str


class CredentialImportReq(BaseModel):
    """凭据导入请求"""

    version: str = "1.0"
    name: str = Field(min_length=1)
    type: Literal["git_ssh", "git_token", "gitee_token", "registry_token"]
    data: str = Field(min_length=1)
