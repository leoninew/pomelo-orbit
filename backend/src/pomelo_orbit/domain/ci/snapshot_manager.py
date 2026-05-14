"""Snapshot 管理领域服务"""

from pomelo_orbit.domain.ci.entities import PipelineSnapshot, PipelineTemplate
from pomelo_orbit.domain.ci.repositories import PipelineSnapshotRepository
from pomelo_orbit.domain.ci.value_objects import VariableDeclaration


class SnapshotManager:
    """Snapshot 管理领域服务

    职责：
    - 根据模板创建或获取快照
    - 快照版本管理
    """

    def __init__(self, snapshot_repo: PipelineSnapshotRepository):
        self.snapshot_repo = snapshot_repo

    def get_or_create_snapshot(
        self,
        template: PipelineTemplate,
        variable_declarations: list[VariableDeclaration],
    ) -> PipelineSnapshot:
        """获取或创建快照（模板版本未变则复用，变更则创建新版本）

        Args:
            template: 模板实体
            variable_declarations: 完整的变量声明列表（内置 + stage + 自定义）

        Returns:
            快照实体
        """
        latest = self.snapshot_repo.find_latest(template.project_id, template.id)

        # 如果最新快照的版本与当前模板一致，直接复用
        if latest and latest.version == template.version:
            return latest

        # 否则创建新版本快照
        snapshot = PipelineSnapshot.create(
            template,
            version=template.version,
            variable_declarations=variable_declarations,
        )
        self.snapshot_repo.save(snapshot)
        return snapshot
