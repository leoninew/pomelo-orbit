"""凭据相关 DTO"""

from datetime import datetime
from typing import Literal

from pydantic import BaseModel


class CredentialResp(BaseModel):
    id: str
    name: str
    type: str
    created_at: datetime

    model_config = {"from_attributes": True}


class CredentialCreateReq(BaseModel):
    name: str
    type: Literal["git_ssh", "git_token", "registry_token"]
    data: str


class CredentialUpdateReq(BaseModel):
    name: str | None = None
    data: str | None = None
