"""Pipeline 执行器接口"""

from abc import ABC, abstractmethod
from typing import Any

from pomelo_orbit.domain.ci.value_objects import StageDefinition


class ExecutionContext:
    """执行上下文"""

    def __init__(
        self,
        run_id: str,
        project_id: str,
        repository_id: str,
        repository_name: str,
        template_id: str,
        template_name: str,
        project_code: str,
        repository_url: str,
        credential_id: str | None,
        variables: dict[str, Any],
        retry_of: str | None = None,
    ):
        self.run_id = run_id
        self.project_id = project_id
        self.repository_id = repository_id
        self.repository_name = repository_name
        self.template_id = template_id
        self.template_name = template_name
        self.project_code = project_code
        self.repository_url = repository_url
        self.credential_id = credential_id
        self.variables = variables
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
