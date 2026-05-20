import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.auth.di import get_role_service, get_user_service
from pomelo_orbit.application.auth.role_service import RoleService
from pomelo_orbit.application.auth.user_service import UserService
from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.permissions import require_permission
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.dto.user import UserCreateReq, UserListResp, UserResp, UserRoleUpdateReq, UserUpdateReq

router = APIRouter(prefix="/user", tags=["user"])


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
    role_service: Annotated[RoleService, Depends(get_role_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> UserResp:
    user = user_service.create_user(username=data.username, password=data.password, email=data.email)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    roles_permissions = role_service.get_roles_permission_codes([role.id for role in roles])
    return UserResp.from_domain(user, roles, permission_codes, roles_permissions)


@router.get("/{user_id}", response_model=UserResp)
def get_user(
    user_id: str,
    user_service: Annotated[UserService, Depends(get_user_service)],
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("user:read"))],
) -> UserResp:
    user = user_service.get_user(user_id)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    roles_permissions = role_service.get_roles_permission_codes([role.id for role in roles])
    return UserResp.from_domain(user, roles, permission_codes, roles_permissions)


@router.put("/{user_id}", response_model=UserResp)
def update_user(
    user_id: str,
    data: UserUpdateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    role_service: Annotated[RoleService, Depends(get_role_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> UserResp:
    user = user_service.update_user(current_user.id, user_id, username=data.username, password=data.password, status=data.status)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    roles_permissions = role_service.get_roles_permission_codes([role.id for role in roles])
    return UserResp.from_domain(user, roles, permission_codes, roles_permissions)


@router.put("/{user_id}/role", response_model=UserResp)
def update_user_roles(
    user_id: str,
    data: UserRoleUpdateReq,
    user_service: Annotated[UserService, Depends(get_user_service)],
    role_service: Annotated[RoleService, Depends(get_role_service)],
    current_user: Annotated[User, Depends(require_permission("user:write"))],
) -> UserResp:
    if not user_service.can_assign_roles(current_user.id):
        raise BusinessError("Permission denied", status_code=403)
    user = user_service.update_user_roles(user_id, data.role_ids)
    roles = user_service.get_user_roles(user.id)
    permission_codes = user_service.get_user_permission_codes(user.id)
    roles_permissions = role_service.get_roles_permission_codes([role.id for role in roles])
    return UserResp.from_domain(user, roles, permission_codes, roles_permissions)


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
