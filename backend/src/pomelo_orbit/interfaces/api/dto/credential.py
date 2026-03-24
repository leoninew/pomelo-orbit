"""
凭据相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel, Field


class CredentialCreateReq(BaseModel):
    """创建凭据"""

    application_id: str = Field(..., min_length=1)
    name: str = Field(..., min_length=1, max_length=100)
    type: str = Field(..., pattern="^(github_token|docker_registry|ssh_key)$")
    value: str = Field(..., min_length=1)
    extra_data: str | None = None


class CredentialUpdateReq(BaseModel):
    """更新凭据"""

    name: str | None = Field(None, min_length=1, max_length=100)
    value: str | None = Field(None, min_length=1)
    extra_data: str | None = None


class CredentialResp(BaseModel):
    """凭据响应（不包含值）"""

    id: str
    application_id: str
    name: str
    type: str
    extra_data: str | None
    created_at: datetime

    model_config = {"from_attributes": True}


class CredentialDetailResp(BaseModel):
    """凭据详情响应（包含解密后的值）"""

    id: str
    application_id: str
    name: str
    type: str
    value: str
    extra_data: str | None
    created_at: datetime

    model_config = {"from_attributes": True}


class CredentialAssociationReq(BaseModel):
    """关联凭证请求"""

    credential_id: str = Field(..., min_length=1)
