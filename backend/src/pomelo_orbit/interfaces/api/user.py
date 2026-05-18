import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.auth.di import get_user_service
from pomelo_orbit.application.auth.user_service import UserService
from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.permissions import require_permission
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.dto.user import UserCreateReq, UserListResp, UserResp, UserUpdateReq

router = APIRouter(prefix="/user", tags=["user"])


def _ensure_can_assign_roles(current_user: User, user_service: UserService) -> None:
    if "role:write" not in user_service.get_user_permission_codes(current_user.id):
        raise BusinessError("Permission denied", status_code=403)


def _ensure_can_reset_password(current_user: User, target_user_id: str, user_service: UserService) -> None:
    if current_user.id == target_user_id:
        return
    if "role:write" in user_service.get_user_permission_codes(current_user.id):
        return
    if "role:write" in user_service.get_user_permission_codes(target_user_id):
        raise BusinessError("Permission denied", status_code=403)


@router.get("", response_model=PaginatedResp[UserListResp])
def list_users(
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(require_permission("user:read"))],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[UserListResp]:
    users, total = user_service.list_users(page=page, per_page=per_page, search=search)
    roles_by_user_id = user_service.get_users_roles([user.id for user in users])
    return PaginatedResp(
        items=[UserListResp.from_domain(user, roles_by_user_id.get(user.id, [])) for user in users],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=UserResp, status_code=201)
def create_user(
    data: UserCreateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> UserResp:
    if data.role_ids:
        _ensure_can_assign_roles(current_user, user_service)
    user = user_service.create_user(username=data.username, password=data.password, email=data.email, role_ids=data.role_ids)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    return UserResp.from_domain(user, roles, permission_codes)


@router.get("/{user_id}", response_model=UserResp)
def get_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(require_permission("user:read"))],
) -> UserResp:
    user = user_service.get_user(user_id)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    return UserResp.from_domain(user, roles, permission_codes)


@router.put("/{user_id}", response_model=UserResp)
def update_user(
    user_id: str,
    data: UserUpdateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> UserResp:
    if data.role_ids is not None:
        _ensure_can_assign_roles(current_user, user_service)
    if data.password is not None:
        _ensure_can_reset_password(current_user, user_id, user_service)
    user = user_service.update_user(user_id, password=data.password, role_ids=data.role_ids)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    return UserResp.from_domain(user, roles, permission_codes)


@router.post("/{user_id}/disable", status_code=204)
def disable_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> None:
    user_service.disable_user(current_user.id, user_id)


@router.post("/{user_id}/enable", status_code=204)
def enable_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> None:
    user_service.enable_user(user_id)


@router.delete("/{user_id}", status_code=204)
def delete_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> None:
    user_service.delete_user(current_user.id, user_id)
