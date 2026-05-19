import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.auth.di import get_role_service
from pomelo_orbit.application.auth.role_service import RoleService
from pomelo_orbit.domain.auth.entities import Permission, Role, User
from pomelo_orbit.interfaces.api.auth.permissions import require_permission
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.dto.role import PermissionResp, RoleCreateReq, RoleResp, RoleUpdateReq

router = APIRouter(prefix="/role", tags=["role"])


def _to_permission_resp(permission: Permission) -> PermissionResp:
    return PermissionResp(
        id=permission.id,
        code=permission.code,
        name=permission.name,
        description=permission.description,
    )


def _to_role_resp(role: Role, role_service: RoleService) -> RoleResp:
    return RoleResp(
        id=role.id,
        code=role.code,
        name=role.name,
        description=role.description,
        created_at=role.created_at,
        updated_at=role.updated_at,
        permission_codes=role_service.get_role_permission_codes(role.id),
    )


@router.get("/permission", response_model=list[PermissionResp])
def list_permissions(
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:read"))],
) -> list[PermissionResp]:
    return [_to_permission_resp(permission) for permission in role_service.list_permissions()]


@router.get("", response_model=PaginatedResp[RoleResp])
def list_roles(
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:read"))],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[RoleResp]:
    roles, total = role_service.list_roles(page=page, per_page=per_page, search=search)
    # 批量查询所有角色的权限码，避免 N+1 查询
    role_ids = [role.id for role in roles]
    permissions_by_role = role_service.get_roles_permission_codes(role_ids)
    return PaginatedResp(
        items=[
            RoleResp(
                id=role.id,
                code=role.code,
                name=role.name,
                description=role.description,
                created_at=role.created_at,
                updated_at=role.updated_at,
                permission_codes=permissions_by_role.get(role.id, []),
            )
            for role in roles
        ],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=RoleResp, status_code=201)
def create_role(
    data: RoleCreateReq,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:write"))],
) -> RoleResp:
    role = role_service.create_role(
        code=data.code,
        name=data.name,
        description=data.description,
        permission_codes=data.permission_codes,
    )
    return _to_role_resp(role, role_service)


@router.get("/{role_id}", response_model=RoleResp)
def get_role(
    role_id: str,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:read"))],
) -> RoleResp:
    role = role_service.get_role(role_id)
    return _to_role_resp(role, role_service)


@router.put("/{role_id}", response_model=RoleResp)
def update_role(
    role_id: str,
    data: RoleUpdateReq,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:write"))],
) -> RoleResp:
    role = role_service.update_role(
        role_id,
        code=data.code,
        name=data.name,
        description=data.description,
        permission_codes=data.permission_codes,
    )
    return _to_role_resp(role, role_service)


@router.delete("/{role_id}", status_code=204)
def delete_role(
    role_id: str,
    role_service: Annotated[RoleService, Depends(get_role_service)],
    _current_user: Annotated[User, Depends(require_permission("role:write"))],
) -> None:
    role_service.delete_role(role_id)
