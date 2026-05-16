from datetime import datetime

from pydantic import BaseModel, Field


class ProjectCreateReq(BaseModel):
    name: str = Field(min_length=1, max_length=100)
    code: str = Field(min_length=1, max_length=100, pattern=r"^[a-z0-9_-]+$")


class ProjectUpdateReq(BaseModel):
    name: str = Field(min_length=1, max_length=100)
    code: str = Field(min_length=1, max_length=100, pattern=r"^[a-z0-9_-]+$")


class ProjectResp(BaseModel):
    id: str
    name: str
    code: str
    owner_user_id: str
    is_active: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}
