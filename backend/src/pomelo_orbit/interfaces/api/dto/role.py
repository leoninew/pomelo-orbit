from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field


class RoleCreateReq(BaseModel):
    code: str = Field(min_length=1, max_length=50, pattern=r"^[A-Za-z0-9_-]+$")
    name: str = Field(min_length=1, max_length=100)
    description: str | None = Field(default=None, max_length=500)
    permission_codes: list[str] = Field(default_factory=list)


class RoleUpdateReq(BaseModel):
    code: str = Field(min_length=1, max_length=50, pattern=r"^[A-Za-z0-9_-]+$")
    name: str = Field(min_length=1, max_length=100)
    description: str | None = Field(default=None, max_length=500)
    permission_codes: list[str] = Field(default_factory=list)


class PermissionResp(BaseModel):
    id: str
    code: str
    name: str
    description: str | None


class RoleResp(BaseModel):
    id: str
    code: str
    name: str
    description: str | None
    is_active: bool
    created_at: datetime
    updated_at: datetime
    permission_codes: list[str] = Field(default_factory=list)

    model_config = ConfigDict(from_attributes=True)
