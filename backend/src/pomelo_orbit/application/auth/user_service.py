from typing import Literal

from ulid import ULID

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.constants import AuthSource
from pomelo_orbit.domain.auth.entities import Role, User
from pomelo_orbit.domain.auth.repositories import RoleRepository
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import hash_password
from pomelo_orbit.infrastructure.time_utils import utc_now


class UserService:
    def __init__(self, user_repo: UserRepository, role_repo: RoleRepository):
        self.user_repo = user_repo
        self.role_repo = role_repo

    def list_users(self, page: int, per_page: int, search: str | None) -> tuple[list[User], int]:
        return self.user_repo.find_paginated(page, per_page, search)

    def get_user(self, user_id: str) -> User:
        user = self.user_repo.find_by_id(user_id)
        if not user:
            raise BusinessError(f"User {user_id} not found", status_code=404)
        return user

    def create_user(self, *, username: str, password: str, email: str | None, role_ids: list[str]) -> User:
        self._ensure_username_available(username)
        self._ensure_email_available(email)
        self._ensure_roles_exist(role_ids)
        now = utc_now()
        user = User(
            id=str(ULID()),
            username=username,
            password_hash=hash_password(password),
            status="enabled",
            oauth_provider="",
            oauth_provider_id="",
            email=email,
            auth_source=AuthSource.PASSWORD,
            created_at=now,
            updated_at=now,
        )
        self.user_repo.save(user)
        self.user_repo.set_roles(user.id, role_ids)
        return user

    def update_user(
        self,
        user_id: str,
        *,
        password: str | None,
        role_ids: list[str] | None,
        status: Literal["enabled", "disabled"],
    ) -> User:
        user = self.get_user(user_id)
        should_save = False
        if role_ids is not None:
            self._ensure_roles_exist(role_ids)
            self.user_repo.set_roles(user_id, role_ids)
            user.updated_at = utc_now()
            should_save = True
        if password is not None:
            user.password_hash = hash_password(password)
            user.updated_at = utc_now()
            should_save = True
        if user.status != status:
            user.status = status
            user.updated_at = utc_now()
            should_save = True
        if should_save:
            self.user_repo.save(user)
        return user

    def disable_user(self, current_user_id: str, user_id: str) -> None:
        if current_user_id == user_id:
            raise BusinessError("Cannot disable current user", status_code=400)
        user = self.get_user(user_id)
        if user.status == "disabled":
            return
        user.status = "disabled"
        user.updated_at = utc_now()
        self.user_repo.save(user)

    def enable_user(self, user_id: str) -> None:
        user = self.get_user(user_id)
        if user.status == "enabled":
            return
        user.status = "enabled"
        user.updated_at = utc_now()
        self.user_repo.save(user)

    def delete_user(self, current_user_id: str, user_id: str) -> None:
        if current_user_id == user_id:
            raise BusinessError("Cannot delete current user", status_code=400)
        user = self.get_user(user_id)
        self.user_repo.delete(user)

    def get_user_roles(self, user_id: str) -> list[Role]:
        return self.user_repo.find_roles(user_id)

    def get_users_roles(self, user_ids: list[str]) -> dict[str, list[Role]]:
        return self.user_repo.find_roles_by_user_ids(user_ids)

    def get_user_permission_codes(self, user_id: str) -> list[str]:
        return [permission.code for permission in self.user_repo.find_permissions(user_id)]

    def can_assign_roles(self, user_id: str) -> bool:
        """检查用户是否有权限分配角色"""
        return "role:write" in self.get_user_permission_codes(user_id)

    def can_reset_password(self, current_user_id: str, target_user_id: str) -> bool:
        """检查用户是否有权限重置目标用户的密码"""
        # 用户可以重置自己的密码
        if current_user_id == target_user_id:
            return True
        # 拥有 role:write 权限的用户可以重置任何人的密码
        if "role:write" in self.get_user_permission_codes(current_user_id):
            return True
        # 否则，不能重置拥有 role:write 权限用户的密码
        return "role:write" not in self.get_user_permission_codes(target_user_id)

    def _ensure_username_available(self, username: str, user_id: str | None = None) -> None:
        existing = self.user_repo.find_by_username(username)
        if existing and existing.id != user_id:
            raise BusinessError(f"Username {username} already exists", status_code=409)

    def _ensure_email_available(self, email: str | None, user_id: str | None = None) -> None:
        if not email:
            return
        existing = self.user_repo.find_by_email(email)
        if existing and existing.id != user_id:
            raise BusinessError(f"Email {email} already exists", status_code=409)

    def _ensure_roles_exist(self, role_ids: list[str]) -> None:
        for role_id in role_ids:
            if not self.role_repo.find_by_id(role_id):
                raise BusinessError(f"Role {role_id} not found", status_code=404)
