from ulid import ULID

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.constants import AuthSource
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure import hash_password
from pomelo_orbit.infrastructure.time_utils import utc_now


class UserService:
    def __init__(self, user_repo: UserRepository):
        self.user_repo = user_repo

    def list_users(self, page: int, per_page: int, search: str | None) -> tuple[list[User], int]:
        return self.user_repo.find_paginated(page, per_page, search)

    def get_user(self, user_id: str) -> User:
        user = self.user_repo.find_by_id(user_id)
        if not user:
            raise BusinessError(f"User {user_id} not found", status_code=404)
        return user

    def create_user(self, *, username: str, password: str, email: str | None) -> User:
        self._ensure_username_available(username)
        self._ensure_email_available(email)
        now = utc_now()
        user = User(
            id=str(ULID()),
            username=username,
            password_hash=hash_password(password),
            email=email,
            auth_source=AuthSource.PASSWORD,
            is_active=True,
            created_at=now,
            updated_at=now,
        )
        self.user_repo.save(user)
        return user

    def update_user(self, user_id: str, *, password: str) -> User:
        user = self.get_user(user_id)
        user.password_hash = hash_password(password)
        user.updated_at = utc_now()
        self.user_repo.save(user)
        return user

    def disable_user(self, current_user_id: str, user_id: str) -> None:
        if current_user_id == user_id:
            raise BusinessError("Cannot disable current user", status_code=400)
        user = self.get_user(user_id)
        if not user.is_active:
            return
        user.is_active = False
        user.updated_at = utc_now()
        self.user_repo.save(user)

    def enable_user(self, user_id: str) -> None:
        user = self.get_user(user_id)
        if user.is_active:
            return
        user.is_active = True
        user.updated_at = utc_now()
        self.user_repo.save(user)

    def delete_user(self, current_user_id: str, user_id: str) -> None:
        if current_user_id == user_id:
            raise BusinessError("Cannot delete current user", status_code=400)
        user = self.get_user(user_id)
        self.user_repo.delete(user)

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
