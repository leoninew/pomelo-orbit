from ulid import ULID

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.entities import Permission, Role
from pomelo_orbit.domain.auth.repositories import PermissionRepository, RoleRepository
from pomelo_orbit.infrastructure.time_utils import utc_now


class RoleService:
    def __init__(self, role_repo: RoleRepository, permission_repo: PermissionRepository):
        self.role_repo = role_repo
        self.permission_repo = permission_repo

    def list_roles(self, page: int, per_page: int, search: str | None) -> tuple[list[Role], int]:
        return self.role_repo.find_paginated(page, per_page, search)

    def get_role(self, role_id: str) -> Role:
        role = self.role_repo.find_by_id(role_id)
        if not role:
            raise BusinessError(f"Role {role_id} not found", status_code=404)
        return role

    def create_role(self, *, code: str, name: str, description: str | None, permission_codes: list[str]) -> Role:
        self._ensure_code_available(code)
        self._ensure_name_available(name)
        self._ensure_permissions_exist(permission_codes)
        now = utc_now()
        role = Role(
            id=str(ULID()),
            code=code,
            name=name,
            description=description,
            created_at=now,
            updated_at=now,
        )
        self.role_repo.save(role)
        self.permission_repo.set_role_permissions(role.id, permission_codes)
        return role

    def update_role(
        self, role_id: str, *, code: str, name: str, description: str | None, permission_codes: list[str]
    ) -> Role:
        role = self.get_role(role_id)
        self._ensure_code_available(code, role_id)
        self._ensure_name_available(name, role_id)
        self._ensure_permissions_exist(permission_codes)
        role.code = code
        role.name = name
        role.description = description
        role.updated_at = utc_now()
        self.role_repo.save(role)
        self.permission_repo.set_role_permissions(role_id, permission_codes)
        return role

    def delete_role(self, role_id: str) -> None:
        role = self.get_role(role_id)
        self.role_repo.delete(role)

    def list_permissions(self) -> list[Permission]:
        return self.permission_repo.list_all()

    def get_role_permission_codes(self, role_id: str) -> list[str]:
        return [permission.code for permission in self.permission_repo.find_by_role_id(role_id)]

    def get_roles_permission_codes(self, role_ids: list[str]) -> dict[str, list[str]]:
        permissions_by_role = self.permission_repo.find_by_role_ids(role_ids)
        return {role_id: [p.code for p in permissions] for role_id, permissions in permissions_by_role.items()}

    def _ensure_code_available(self, code: str, role_id: str | None = None) -> None:
        existing = self.role_repo.find_by_code(code)
        if existing and existing.id != role_id:
            raise BusinessError(f"Role code {code} already exists", status_code=409)

    def _ensure_name_available(self, name: str, role_id: str | None = None) -> None:
        existing = self.role_repo.find_by_name(name)
        if existing and existing.id != role_id:
            raise BusinessError(f"Role name {name} already exists", status_code=409)

    def _ensure_permissions_exist(self, permission_codes: list[str]) -> None:
        permissions = self.permission_repo.find_by_codes(permission_codes)
        found_codes = {permission.code for permission in permissions}
        missing_codes = set(permission_codes) - found_codes
        if missing_codes:
            raise BusinessError(f"Permission {sorted(missing_codes)[0]} not found", status_code=404)
