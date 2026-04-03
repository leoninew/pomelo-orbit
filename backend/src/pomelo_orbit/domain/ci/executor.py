"""Pipeline 执行器接口"""

from abc import ABC, abstractmethod
from typing import Any

from pomelo_orbit.domain.ci.value_objects import PipelineDefinition


class ExecutionContext:
    """执行上下文"""

    def __init__(
        self,
        run_id: str,
        project_id: str,
        repository_url: str,
        credential_id: str | None,
        variables: dict[str, Any],
        workspace_path: str,
        artifacts_path: str,
        retry_of: str | None = None,
    ):
        self.run_id = run_id
        self.project_id = project_id
        self.repository_url = repository_url
        self.credential_id = credential_id
        self.variables = variables
        self.workspace_path = workspace_path
        self.artifacts_path = artifacts_path
        self.retry_of = retry_of


class PipelineExecutor(ABC):
    """Pipeline 执行器抽象接口"""

    @abstractmethod
    async def execute(self, context: ExecutionContext, definition: PipelineDefinition) -> bool:
        """
        执行 pipeline

        Args:
            context: 执行上下文
            definition: Pipeline 定义

        Returns:
            是否执行成功
        """
        ...
