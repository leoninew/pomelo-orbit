"""Template 聚合的应用服务"""

import logging
import re

from pomelo_orbit.domain.ci.entities import BuildStage, PipelineSnapshot, PipelineTemplate
from pomelo_orbit.domain.ci.repositories import (
    BuildStageRepository,
    PipelineSnapshotRepository,
    PipelineTemplateRepository,
    RepositoryWebhookRepository,
)
from pomelo_orbit.domain.ci.value_objects import StageOrchestration, VariableDeclaration
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.domain.exceptions import BusinessError

logger = logging.getLogger(__name__)


class TemplateService:
    """PipelineTemplate 聚合根的应用服务

    职责：
    - Template 的 CRUD 操作
    - Template 编排管理（StageOrchestration）
    - Template 变量声明管理
    - Snapshot 创建和管理（Snapshot 属于 Template 聚合）
    - Template 版本管理

    依赖：
    - BuildStageRepository: 加载 Stage 实体
    - VariableResolver: 变量解析
    """

    def __init__(
        self,
        template_repo: PipelineTemplateRepository,
        snapshot_repo: PipelineSnapshotRepository,
        webhook_repo: RepositoryWebhookRepository,
        stage_repo: BuildStageRepository,
        variable_resolver: VariableResolver,
    ):
        self.template_repo = template_repo
        self.snapshot_repo = snapshot_repo
        self.webhook_repo = webhook_repo
        self.stage_repo = stage_repo
        self.variable_resolver = variable_resolver

    def list_templates(
        self, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[PipelineTemplate], int]:
        """分页查询模板列表"""
        return self.template_repo.find_paginated(page=page, per_page=per_page, search=search)

    def get_template(self, template_id: str) -> PipelineTemplate:
        """获取存储的模板（不含变量装饰）"""
        tmpl = self.template_repo.find_by_id(template_id)
        if not tmpl:
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)
        return tmpl

    def get_template_variables(self, template: PipelineTemplate) -> list[VariableDeclaration]:
        """获取模板的完整变量声明列表（供 API 层调用）"""
        return self.variable_resolver.resolve_template_variables(
            template.stages,
            template.variable_declarations,
        )

    def create_template(
        self,
        name: str,
        description: str = "",
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> PipelineTemplate:
        """创建模板"""
        tmpl = PipelineTemplate.create(
            name=name,
            variable_declarations=self.variable_resolver.sanitize_variable_overrides(variable_declarations or []),
            description=description,
        )
        self.template_repo.save(tmpl)
        return tmpl

    def update_template(
        self,
        template_id: str,
        name: str | None = None,
        description: str | None = None,
        orchestration: list[StageOrchestration] | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> PipelineTemplate:
        """更新模板（支持编排和变量声明更新）"""
        tmpl = self.get_template(template_id)

        orch_changed = False
        stages = list(tmpl.stages)

        if orchestration is not None:
            stages = self._load_stages_by_ids([o.stage_id for o in orchestration]) if orchestration else []
            stage_version_map = {s.id: s.version for s in stages}

            # 将当前 stage 版本写入编排记录
            orchestration_with_version = [
                o.model_copy(update={"stage_version": stage_version_map[o.stage_id]}) for o in orchestration
            ]

            # 仅在编排实际变更时才写库和递增版本
            if tmpl.has_orchestration_changed(orchestration_with_version):
                self.template_repo.save_orchestration(template_id, orchestration_with_version)
                orch_changed = True
                tmpl.orchestration = orchestration_with_version
                tmpl.stages = stages

        fields_changed = tmpl.update(
            name=name,
            description=description,
            variable_declarations=self.variable_resolver.sanitize_variable_overrides(
                variable_declarations if variable_declarations is not None else list(tmpl.variable_declarations)
            ),
        )

        if orch_changed or fields_changed:
            old_version = tmpl.version
            tmpl.bump_version()
            logger.info(
                f"Template version bumped: template_id={tmpl.id}, old_version={old_version}, new_version={tmpl.version}"
            )

        self.template_repo.save(tmpl)
        return tmpl

    def delete_template(self, template_id: str) -> None:
        """删除模板（检查 webhook 引用）"""
        tmpl = self.get_template(template_id)
        webhooks = self.webhook_repo.find_by_template(template_id)
        if webhooks:
            raise BusinessError("Template is referenced by webhooks, cannot delete", status_code=409)
        self.template_repo.delete(tmpl)

    def duplicate_template(self, template_id: str) -> PipelineTemplate:
        """复制模板（自动生成唯一名称）"""
        tmpl = self.get_template(template_id)

        # 获取基础名称（去掉可能的 " copy" 或 " copy N" 后缀）
        base_name = re.sub(r" copy( \d+)?$", "", tmpl.name)

        # 从 "原名称 copy" 开始尝试，依次递增
        i = 1
        while True:
            new_name = f"{base_name} copy" if i == 1 else f"{base_name} copy {i}"
            if not self.template_repo.find_by_name(new_name):
                break
            i += 1

        new_template = PipelineTemplate.create(
            name=new_name,
            variable_declarations=list(tmpl.variable_declarations),
            description=tmpl.description,
        )
        self.template_repo.save(new_template)

        # 复制编排
        if tmpl.orchestration:
            self.template_repo.save_orchestration(new_template.id, tmpl.orchestration)

        # 更新 stages
        new_template.orchestration = tmpl.orchestration
        new_template.stages = tmpl.stages
        self.template_repo.save(new_template)
        return new_template

    def resolve_template_variables(
        self,
        orchestration: list[StageOrchestration],
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> list[VariableDeclaration]:
        """解析模板变量（用于前端预览）"""
        stages = self._load_stages_by_ids([o.stage_id for o in orchestration]) if orchestration else []
        return self.variable_resolver.resolve_template_variables(stages, variable_declarations or [])

    # ── Snapshot 管理 ────────────────────────────────────────────────────────

    def get_snapshot(self, snapshot_id: str) -> PipelineSnapshot:
        """获取快照"""
        snapshot = self.snapshot_repo.find_by_id(snapshot_id)
        if not snapshot:
            raise BusinessError(f"PipelineSnapshot {snapshot_id} not found", status_code=404)
        return snapshot

    # ── 私有方法 ──────────────────────────────────────────────────────────────

    def _load_stages_by_ids(self, stage_ids: list[str]) -> list[BuildStage]:
        """批量加载 Stage（用于模板编排）"""
        stages = self.stage_repo.find_by_ids(stage_ids)
        if len(stages) != len(stage_ids):
            found = {stage.id for stage in stages}
            missing = [stage_id for stage_id in stage_ids if stage_id not in found]
            raise BusinessError(f"Stage(s) not found: {missing}", status_code=404)
        return stages
