from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import LoginAttempt


class LoginAttemptRepository(ABC):
    """登录尝试记录仓储接口"""

    @abstractmethod
    def save(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录"""
        ...

    @abstractmethod
    def count_failed_by_ip(self, ip_address: str, minutes: int) -> int:
        """统计指定 IP 在指定时间内的失败次数"""
        ...

    @abstractmethod
    def count_failed_by_username(self, username: str, minutes: int) -> int:
        """统计指定用户名在指定时间内的失败次数"""
        ...

    @abstractmethod
    def delete_old_records(self, days: int) -> None:
        """删除指定天数之前的登录尝试记录"""
        ...
