from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import LoginAttempt


class LoginAttemptRepository(ABC):
    """登录尝试记录仓储接口"""

    @abstractmethod
    def save(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录"""
        ...

    @abstractmethod
    def save_and_commit(self, attempt: LoginAttempt) -> None:
        """保存登录尝试记录并立即提交（用于失败记录，防止事务回滚）"""
        ...

    @abstractmethod
    def get_failed_attempts_summary_by_ip(self, ip_address: str, minutes: int) -> tuple[int, LoginAttempt | None]:
        """获取 IP 失败次数和最后失败记录（单次查询优化）"""
        ...

    @abstractmethod
    def get_failed_attempts_summary_by_username(self, username: str, minutes: int) -> tuple[int, LoginAttempt | None]:
        """获取用户名失败次数和最后失败记录（单次查询优化）"""
        ...

    @abstractmethod
    def delete_old_records(self, days: int) -> None:
        """删除指定天数之前的登录尝试记录"""
        ...
