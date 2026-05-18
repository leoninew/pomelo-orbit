from sqlalchemy import or_
from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.entities import LoginHistory, Permission, Role, User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import LoginHistoryMapper, PermissionMapper, RoleMapper, UserMapper
from pomelo_orbit.infrastructure.persistence.models import (
    LoginHistoryModel,
    PermissionModel,
    RoleModel,
    RolePermissionModel,
    UserModel,
    UserRoleModel,
)


class UserRepositoryImpl(BaseRepository[User, UserModel], UserRepository):
    """用户仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, UserModel, UserMapper)

    def find_by_username(self, username: str) -> User | None:
        model = self._session.query(UserModel).filter(UserModel.username == username).first()
        return self._mapper.to_domain(model) if model else None

    def find_by_oauth_account(self, provider: str, provider_id: str) -> User | None:
        if not provider or not provider_id:
            return None
        model = (
            self._session.query(UserModel)
            .filter(UserModel.oauth_provider == provider, UserModel.oauth_provider_id == provider_id)
            .first()
        )
        return self._mapper.to_domain(model) if model else None

    def find_by_email(self, email: str) -> User | None:
        if not email:
            return None
        model = self._session.query(UserModel).filter(UserModel.email == email).first()
        return self._mapper.to_domain(model) if model else None

    def find_paginated(self, page: int = 1, per_page: int = 20, search: str | None = None) -> tuple[list[User], int]:
        query = self._session.query(UserModel)
        if search:
            keyword = f"%{search}%"
            query = query.filter(or_(UserModel.username.ilike(keyword), UserModel.email.ilike(keyword)))
        total = query.count()
        models = query.order_by(UserModel.created_at.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(m) for m in models], total

    def find_roles(self, user_id: str) -> list[Role]:
        models = (
            self._session.query(RoleModel)
            .join(UserRoleModel, UserRoleModel.role_id == RoleModel.id)
            .filter(UserRoleModel.user_id == user_id)
            .order_by(RoleModel.code.asc())
            .all()
        )
        return [RoleMapper.to_domain(model) for model in models]

    def find_roles_by_user_ids(self, user_ids: list[str]) -> dict[str, list[Role]]:
        if not user_ids:
            return {}
        rows = (
            self._session.query(UserRoleModel.user_id, RoleModel)
            .join(RoleModel, RoleModel.id == UserRoleModel.role_id)
            .filter(UserRoleModel.user_id.in_(user_ids))
            .order_by(UserRoleModel.user_id.asc(), RoleModel.code.asc())
            .all()
        )
        roles_by_user_id: dict[str, list[Role]] = {user_id: [] for user_id in user_ids}
        for user_id, role_model in rows:
            roles_by_user_id[user_id].append(RoleMapper.to_domain(role_model))
        return roles_by_user_id

    def find_permissions(self, user_id: str) -> list[Permission]:
        models = (
            self._session.query(PermissionModel)
            .join(RolePermissionModel, RolePermissionModel.permission_id == PermissionModel.id)
            .join(UserRoleModel, UserRoleModel.role_id == RolePermissionModel.role_id)
            .filter(UserRoleModel.user_id == user_id)
            .distinct()
            .order_by(PermissionModel.code.asc())
            .all()
        )
        return [PermissionMapper.to_domain(model) for model in models]

    def set_roles(self, user_id: str, role_ids: list[str]) -> None:
        self._session.query(UserRoleModel).filter(UserRoleModel.user_id == user_id).delete()
        for role_id in role_ids:
            self._session.add(UserRoleModel(user_id=user_id, role_id=role_id))

    def save_login_history(self, history: LoginHistory) -> None:
        model = LoginHistoryMapper.to_orm(history)
        self._session.add(model)

    def find_login_history(
        self, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[LoginHistory], int]:
        query = self._session.query(LoginHistoryModel)
        if search:
            query = query.filter(LoginHistoryModel.username.ilike(f"%{search}%"))
        total = query.count()
        offset = (page - 1) * per_page
        models = query.order_by(LoginHistoryModel.id.desc()).offset(offset).limit(per_page).all()
        return [LoginHistoryMapper.to_domain(m) for m in models], total
