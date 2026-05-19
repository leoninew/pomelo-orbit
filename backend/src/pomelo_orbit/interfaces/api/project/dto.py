from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.domain.auth.entities import User


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
    is_active: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ProjectMemberReq(BaseModel):
    user_id: str = Field(min_length=1)


class ProjectMemberResp(BaseModel):
    id: str
    username: str
    email: str | None

    @classmethod
    def from_domain(cls, user: User) -> "ProjectMemberResp":
        return cls(id=user.id, username=user.username, email=user.email)
