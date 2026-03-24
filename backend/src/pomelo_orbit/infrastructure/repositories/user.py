from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.entities import LoginHistory, User
from pomelo_orbit.domain.repositories import UserRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.mappers import LoginHistoryMapper, UserMapper
from pomelo_orbit.infrastructure.persistence.models import LoginHistoryModel, UserModel


class UserRepositoryImpl(BaseRepository[User, UserModel], UserRepository):
    """用户仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, UserModel, UserMapper)

    def find_by_username(self, username: str) -> User | None:
        model = self._session.query(UserModel).filter(UserModel.username == username).first()
        return self._mapper.to_domain(model) if model else None

    def save_login_history(self, history: LoginHistory) -> None:
        model = LoginHistoryMapper.to_orm(history)
        self._session.add(model)
        self._session.commit()

    def find_login_history(
        self, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[LoginHistory], int]:
        query = self._session.query(LoginHistoryModel)
        if search:
            query = query.filter(LoginHistoryModel.username.contains(search))
        total = query.count()
        offset = (page - 1) * per_page
        models = query.order_by(LoginHistoryModel.login_at.desc()).offset(offset).limit(per_page).all()
        return [LoginHistoryMapper.to_domain(m) for m in models], total


def get_user_repository(db: Annotated[Session, Depends(get_db)]) -> UserRepository:
    """获取用户仓储实例"""
    return UserRepositoryImpl(db)
