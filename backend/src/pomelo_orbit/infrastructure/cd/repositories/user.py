from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.entities import LoginHistory, User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import LoginHistoryMapper, UserMapper
from pomelo_orbit.infrastructure.persistence.models import LoginHistoryModel, UserModel


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
