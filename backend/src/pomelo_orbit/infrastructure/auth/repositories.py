from datetime import timedelta

from sqlalchemy.orm import Session

from pomelo_orbit.domain.auth.entities import LoginAttempt
from pomelo_orbit.domain.auth.repositories import LoginAttemptRepository
from pomelo_orbit.infrastructure.persistence.mappers import LoginAttemptMapper
from pomelo_orbit.infrastructure.persistence.models import LoginAttemptModel
from pomelo_orbit.infrastructure.time_utils import utc_now


class LoginAttemptRepositoryImpl(LoginAttemptRepository):
    """登录尝试记录仓储实现"""

    def __init__(self, db: Session):
        self._session = db

    def save(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录"""
        model = LoginAttemptMapper.to_orm(attempt)
        self._session.add(model)

    def count_failed_by_ip(self, ip_address: str, minutes: int) -> int:
        """统计指定 IP 在指定时间内的失败次数"""
        cutoff_time = utc_now() - timedelta(minutes=minutes)
        return (
            self._session.query(LoginAttemptModel)
            .filter(
                LoginAttemptModel.ip_address == ip_address,
                LoginAttemptModel.success == False,  # noqa: E712
                LoginAttemptModel.created_at >= cutoff_time,
            )
            .count()
        )

    def count_failed_by_username(self, username: str, minutes: int) -> int:
        """统计指定用户名在指定时间内的失败次数"""
        cutoff_time = utc_now() - timedelta(minutes=minutes)
        return (
            self._session.query(LoginAttemptModel)
            .filter(
                LoginAttemptModel.username == username,
                LoginAttemptModel.success == False,  # noqa: E712
                LoginAttemptModel.created_at >= cutoff_time,
            )
            .count()
        )

    def delete_old_records(self, days: int) -> None:
        """删除指定天数之前的登录尝试记录"""
        cutoff_time = utc_now() - timedelta(days=days)
        self._session.query(LoginAttemptModel).filter(LoginAttemptModel.created_at < cutoff_time).delete()
