"""应用管理器接口 - 隔离领域层与基础设施（Docker）的依赖"""

from abc import ABC, abstractmethod


class ApplicationManager(ABC):
    """应用管理器抽象接口"""

    @abstractmethod
    async def deploy(
        self,
        application_code: str,
        config_files: list,
        pull_policy: str,
        deployment_id: str,
        env_file: str | None = None,
    ) -> None: ...

    @abstractmethod
    async def start(self, application_code: str) -> str: ...

    @abstractmethod
    async def stop(self, application_code: str, remove_volumes: bool = False, env_file: str | None = None) -> str: ...

    @abstractmethod
    async def restart(self, application_code: str, env_file: str | None = None) -> str: ...

    @abstractmethod
    async def status(self, application_code: str) -> str: ...

    @abstractmethod
    async def logs(self, application_code: str, tail: int = 100) -> str: ...

    @abstractmethod
    def purge(self, application_code: str) -> None: ...

    @abstractmethod
    def read_deployment_log(self, application_code: str, deployment_id: str, offset: int = 0) -> tuple[str, int]: ...
