from __future__ import annotations

from datetime import datetime
from typing import TYPE_CHECKING, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from pomelo_orbit.interfaces.api.auth.dto import UserInfo

if TYPE_CHECKING:
    from pomelo_orbit.domain.auth.entities import Role, User

UserStatus = Literal["enabled", "disabled"]


def user_status(user: User) -> UserStatus:
    if user.status == "enabled":
        return "enabled"
    if user.status == "disabled":
        return "disabled"
    raise ValueError(f"Unsupported user status: {user.status}")


class UserCreateReq(BaseModel):
    username: str = Field(min_length=1, max_length=50)
    password: str = Field(min_length=6, max_length=255)
    email: str | None = Field(default=None, max_length=255)


class UserUpdateReq(BaseModel):
    password: str | None = Field(default=None, min_length=6, max_length=255)
    status: UserStatus


class UserRoleUpdateReq(BaseModel):
    role_ids: list[str] = Field(default_factory=list)

    @field_validator("role_ids")
    @classmethod
    def role_ids_must_be_unique(cls, role_ids: list[str]) -> list[str]:
        if len(role_ids) != len(set(role_ids)):
            raise ValueError("role_ids must be unique")
        return role_ids


class UserRoleResp(BaseModel):
    id: str
    code: str
    name: str

    @classmethod
    def from_domain(cls, role: Role) -> UserRoleResp:
        return cls(id=role.id, code=role.code, name=role.name)


class UserListResp(BaseModel):
    id: str
    username: str
    email: str | None
    auth_source: str
    created_at: datetime
    last_login_at: datetime | None
    status: UserStatus
    updated_at: datetime
    role_items: list[UserRoleResp] = Field(default_factory=list)

    @classmethod
    def from_domain(cls, user: User, roles: list[Role]) -> UserListResp:
        return cls(
            id=user.id,
            username=user.username,
            email=user.email,
            auth_source=user.auth_source,
            created_at=user.created_at,
            last_login_at=user.last_login_at,
            status=user_status(user),
            updated_at=user.updated_at,
            role_items=[UserRoleResp.from_domain(role) for role in roles],
        )


class UserResp(UserInfo):
    status: UserStatus
    updated_at: datetime
    role_items: list[UserRoleResp] = Field(default_factory=list)

    model_config = ConfigDict(from_attributes=True)

    @classmethod
    def from_domain(cls, user: User, roles: list[Role], permission_codes: list[str]) -> UserResp:
        return cls(
            id=user.id,
            username=user.username,
            email=user.email,
            auth_source=user.auth_source,
            created_at=user.created_at,
            last_login_at=user.last_login_at,
            roles=[role.code for role in roles],
            permissions=permission_codes,
            status=user_status(user),
            updated_at=user.updated_at,
            role_items=[UserRoleResp.from_domain(role) for role in roles],
        )
