from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import LoginAttempt


class LoginAttemptRepository(ABC):
    @abstractmethod
    def save(self, attempt: LoginAttempt) -> None: ...

    @abstractmethod
    def save_and_commit(self, attempt: LoginAttempt) -> None: ...

    @abstractmethod
    def get_failed_attempts_summary_by_ip(self, ip_address: str, minutes: int) -> tuple[int, LoginAttempt | None]: ...

    @abstractmethod
    def get_failed_attempts_summary_by_username(self, username: str, minutes: int) -> tuple[int, LoginAttempt | None]: ...

    @abstractmethod
    def delete_old_records(self, days: int) -> None: ...
