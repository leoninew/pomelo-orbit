from datetime import timedelta

from sqlalchemy import or_
from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.entities import LoginAttempt, Permission, Role
from pomelo_orbit.domain.auth.repositories import LoginAttemptRepository, PermissionRepository, RoleRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import LoginAttemptMapper, PermissionMapper, RoleMapper
from pomelo_orbit.infrastructure.persistence.models import (
    LoginAttemptModel,
    PermissionModel,
    RoleModel,
    RolePermissionModel,
    UserRoleModel,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


class RoleRepositoryImpl(BaseRepository[Role, RoleModel], RoleRepository):
    def __init__(self, db: Session):
        super().__init__(db, RoleModel, RoleMapper)

    def find_by_code(self, code: str) -> Role | None:
        model = self._session.query(RoleModel).filter(RoleModel.code == code).first()
        return self._mapper.to_domain(model) if model else None

    def find_by_name(self, name: str) -> Role | None:
        model = self._session.query(RoleModel).filter(RoleModel.name == name).first()
        return self._mapper.to_domain(model) if model else None

    def find_paginated(self, page: int = 1, per_page: int = 20, search: str | None = None) -> tuple[list[Role], int]:
        query = self._session.query(RoleModel)
        if search:
            keyword = f"%{search}%"
            query = query.filter(
                or_(RoleModel.code.ilike(keyword), RoleModel.name.ilike(keyword), RoleModel.description.ilike(keyword))
            )
        total = query.count()
        models = query.order_by(RoleModel.created_at.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(model) for model in models], total


class PermissionRepositoryImpl(PermissionRepository):
    def __init__(self, db: Session):
        self._session = db

    def list_all(self) -> list[Permission]:
        models = self._session.query(PermissionModel).order_by(PermissionModel.code.asc()).all()
        return [PermissionMapper.to_domain(model) for model in models]

    def find_by_codes(self, codes: list[str]) -> list[Permission]:
        if not codes:
            return []
        models = self._session.query(PermissionModel).filter(PermissionModel.code.in_(codes)).all()
        return [PermissionMapper.to_domain(model) for model in models]

    def find_by_user_id(self, user_id: str) -> list[Permission]:
        models = (
            self._session.query(PermissionModel)
            .join(RolePermissionModel, RolePermissionModel.permission_id == PermissionModel.id)
            .join(RoleModel, RoleModel.id == RolePermissionModel.role_id)
            .filter(RoleModel.id.in_(self._role_ids_for_user(user_id)))
            .distinct()
            .order_by(PermissionModel.code.asc())
            .all()
        )
        return [PermissionMapper.to_domain(model) for model in models]

    def find_by_role_id(self, role_id: str) -> list[Permission]:
        models = (
            self._session.query(PermissionModel)
            .join(RolePermissionModel, RolePermissionModel.permission_id == PermissionModel.id)
            .filter(RolePermissionModel.role_id == role_id)
            .order_by(PermissionModel.code.asc())
            .all()
        )
        return [PermissionMapper.to_domain(model) for model in models]

    def set_role_permissions(self, role_id: str, permission_codes: list[str]) -> None:
        permissions = self.find_by_codes(permission_codes)
        self._session.query(RolePermissionModel).filter(RolePermissionModel.role_id == role_id).delete()
        for permission in permissions:
            self._session.add(RolePermissionModel(role_id=role_id, permission_id=permission.id))

    def _role_ids_for_user(self, user_id: str) -> list[str]:
        rows = self._session.query(UserRoleModel.role_id).filter(UserRoleModel.user_id == user_id).all()
        return [row[0] for row in rows]


class LoginAttemptRepositoryImpl(LoginAttemptRepository):
    """登录尝试记录仓储实现"""

    def __init__(self, db: Session):
        self._session = db

    def save(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录"""
        model = LoginAttemptMapper.to_orm(attempt)
        self._session.add(model)

    def save_and_commit(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录并立即提交（用于失败记录，防止事务回滚）"""
        model = LoginAttemptMapper.to_orm(attempt)
        self._session.add(model)
        self._session.commit()

    def get_failed_attempts_summary_by_ip(self, ip_address: str, minutes: int) -> tuple[int, LoginAttempt | None]:
        """获取 IP 失败次数和最后失败记录（单次查询优化）"""
        cutoff_time = utc_now() - timedelta(minutes=minutes)

        # 查询窗口内的所有失败记录
        failed_attempts = (
            self._session.query(LoginAttemptModel)
            .filter(
                LoginAttemptModel.ip_address == ip_address,
                LoginAttemptModel.success == False,  # noqa: E712
                LoginAttemptModel.created_at >= cutoff_time,
            )
            .order_by(LoginAttemptModel.created_at.desc())
            .all()
        )

        if not failed_attempts:
            return 0, None

        return len(failed_attempts), LoginAttemptMapper.to_domain(failed_attempts[0])

    def get_failed_attempts_summary_by_username(self, username: str, minutes: int) -> tuple[int, LoginAttempt | None]:
        """获取用户名失败次数和最后失败记录（单次查询优化）"""
        cutoff_time = utc_now() - timedelta(minutes=minutes)

        # 查询窗口内的所有失败记录
        failed_attempts = (
            self._session.query(LoginAttemptModel)
            .filter(
                LoginAttemptModel.username == username,
                LoginAttemptModel.success == False,  # noqa: E712
                LoginAttemptModel.created_at >= cutoff_time,
            )
            .order_by(LoginAttemptModel.created_at.desc())
            .all()
        )

        if not failed_attempts:
            return 0, None

        return len(failed_attempts), LoginAttemptMapper.to_domain(failed_attempts[0])

    def delete_old_records(self, days: int) -> None:
        """删除指定天数之前的登录尝试记录"""
        cutoff_time = utc_now() - timedelta(days=days)
        self._session.query(LoginAttemptModel).filter(LoginAttemptModel.created_at < cutoff_time).delete()
