import math
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from pydantic import BaseModel, ConfigDict, Field

from pomelo_orbit.application.auth.di import get_user_service
from pomelo_orbit.application.auth.user_service import UserService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.dto import UserInfo
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.common import PaginatedResp

router = APIRouter(prefix="/user", tags=["user"])


class UserCreateReq(BaseModel):
    username: str = Field(min_length=1, max_length=50)
    password: str = Field(min_length=6, max_length=255)
    email: str | None = Field(default=None, max_length=255)


class UserUpdateReq(BaseModel):
    password: str = Field(min_length=6, max_length=255)


class UserResp(UserInfo):
    is_active: bool
    updated_at: datetime

    model_config = ConfigDict(from_attributes=True)


@router.get("", response_model=PaginatedResp[UserResp])
def list_users(
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[UserResp]:
    users, total = user_service.list_users(page=page, per_page=per_page, search=search)
    return PaginatedResp(
        items=[UserResp.model_validate(user) for user in users],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=UserResp, status_code=201)
def create_user(
    data: UserCreateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> UserResp:
    user = user_service.create_user(username=data.username, password=data.password, email=data.email)
    return UserResp.model_validate(user)


@router.get("/{user_id}", response_model=UserResp)
def get_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> UserResp:
    user = user_service.get_user(user_id)
    return UserResp.model_validate(user)


@router.put("/{user_id}", response_model=UserResp)
def update_user(
    user_id: str,
    data: UserUpdateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> UserResp:
    user = user_service.update_user(user_id, password=data.password)
    return UserResp.model_validate(user)


@router.post("/{user_id}/disable", status_code=204)
def disable_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    user_service.disable_user(current_user.id, user_id)


@router.post("/{user_id}/enable", status_code=204)
def enable_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    user_service.enable_user(user_id)


@router.delete("/{user_id}", status_code=204)
def delete_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    user_service.delete_user(current_user.id, user_id)
