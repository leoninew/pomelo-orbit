"""Pipeline 执行器接口"""

from abc import ABC, abstractmethod
from typing import Any

from pomelo_orbit.domain.ci.value_objects import StageDefinition


class ExecutionContext:
    """执行上下文"""

    def __init__(
        self,
        run_id: str,
        repository_id: str,
        project_code: str,
        repository_url: str,
        credential_id: str | None,
        variables: dict[str, Any],
        workspace_path: str,
        artifacts_path: str,
        retry_of: str | None = None,
    ):
        self.run_id = run_id
        self.repository_id = repository_id
        self.project_code = project_code
        self.repository_url = repository_url
        self.credential_id = credential_id
        self.variables = variables
        self.workspace_path = workspace_path
        self.artifacts_path = artifacts_path
        self.retry_of = retry_of
        self.error_message: str | None = None  # 执行失败时的错误消息


class PipelineExecutor(ABC):
    """Pipeline 执行器抽象接口"""

    @abstractmethod
    async def execute(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        """
        执行 pipeline

        Args:
            context: 执行上下文
            stages: 已解析变量的 Stage 列表

        Returns:
            是否执行成功
        """
        ...
