from ulid import ULID

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.entities import Role
from pomelo_orbit.domain.auth.repositories import RoleRepository
from pomelo_orbit.infrastructure.time_utils import utc_now


class RoleService:
    def __init__(self, role_repo: RoleRepository):
        self.role_repo = role_repo

    def list_roles(self, page: int, per_page: int, search: str | None) -> tuple[list[Role], int]:
        return self.role_repo.find_paginated(page, per_page, search)

    def get_role(self, role_id: str) -> Role:
        role = self.role_repo.find_by_id(role_id)
        if not role:
            raise BusinessError(f"Role {role_id} not found", status_code=404)
        return role

    def create_role(self, *, code: str, name: str, description: str | None) -> Role:
        self._ensure_code_available(code)
        self._ensure_name_available(name)
        now = utc_now()
        role = Role(
            id=str(ULID()),
            code=code,
            name=name,
            description=description,
            is_active=True,
            created_at=now,
            updated_at=now,
        )
        self.role_repo.save(role)
        return role

    def update_role(self, role_id: str, *, code: str, name: str, description: str | None) -> Role:
        role = self.get_role(role_id)
        self._ensure_code_available(code, role_id)
        self._ensure_name_available(name, role_id)
        role.code = code
        role.name = name
        role.description = description
        role.updated_at = utc_now()
        self.role_repo.save(role)
        return role

    def delete_role(self, role_id: str) -> None:
        role = self.get_role(role_id)
        self.role_repo.delete(role)

    def _ensure_code_available(self, code: str, role_id: str | None = None) -> None:
        existing = self.role_repo.find_by_code(code)
        if existing and existing.id != role_id:
            raise BusinessError(f"Role code {code} already exists", status_code=409)

    def _ensure_name_available(self, name: str, role_id: str | None = None) -> None:
        existing = self.role_repo.find_by_name(name)
        if existing and existing.id != role_id:
            raise BusinessError(f"Role name {name} already exists", status_code=409)
