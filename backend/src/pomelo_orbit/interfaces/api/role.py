import math
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from pydantic import BaseModel, ConfigDict, Field

from pomelo_orbit.application.auth.di import get_role_service
from pomelo_orbit.application.auth.role_service import RoleService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.common import PaginatedResp

router = APIRouter(prefix="/role", tags=["role"])


class RoleCreateReq(BaseModel):
    code: str = Field(min_length=1, max_length=50, pattern=r"^[A-Za-z0-9_-]+$")
    name: str = Field(min_length=1, max_length=100)
    description: str | None = Field(default=None, max_length=500)


class RoleUpdateReq(BaseModel):
    code: str = Field(min_length=1, max_length=50, pattern=r"^[A-Za-z0-9_-]+$")
    name: str = Field(min_length=1, max_length=100)
    description: str | None = Field(default=None, max_length=500)


class RoleResp(BaseModel):
    id: str
    code: str
    name: str
    description: str | None
    is_active: bool
    created_at: datetime
    updated_at: datetime

    model_config = ConfigDict(from_attributes=True)


@router.get("", response_model=PaginatedResp[RoleResp])
def list_roles(
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[RoleResp]:
    roles, total = role_service.list_roles(page=page, per_page=per_page, search=search)
    return PaginatedResp(
        items=[RoleResp.model_validate(role) for role in roles],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=RoleResp, status_code=201)
def create_role(
    data: RoleCreateReq,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> RoleResp:
    role = role_service.create_role(code=data.code, name=data.name, description=data.description)
    return RoleResp.model_validate(role)


@router.get("/{role_id}", response_model=RoleResp)
def get_role(
    role_id: str,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> RoleResp:
    role = role_service.get_role(role_id)
    return RoleResp.model_validate(role)


@router.put("/{role_id}", response_model=RoleResp)
def update_role(
    role_id: str,
    data: RoleUpdateReq,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> RoleResp:
    role = role_service.update_role(role_id, code=data.code, name=data.name, description=data.description)
    return RoleResp.model_validate(role)



@router.delete("/{role_id}", status_code=204)
def delete_role(
    role_id: str,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    role_service.delete_role(role_id)
