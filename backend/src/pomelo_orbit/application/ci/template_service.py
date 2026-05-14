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
    """PipelineTemplate 聚合根的应用服务"""

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
        self, project_id: str, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[PipelineTemplate], int]:
        """分页查询模板列表"""
        return self.template_repo.find_paginated_by_project_id(
            project_id=project_id, page=page, per_page=per_page, search=search
        )

    def get_template(self, project_id: str, template_id: str) -> PipelineTemplate:
        """获取存储的模板（不含变量装饰）"""
        tmpl = self.template_repo.find_by_id_in_project(project_id, template_id)
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
        project_id: str,
        name: str,
        description: str = "",
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> PipelineTemplate:
        """创建模板"""
        tmpl = PipelineTemplate.create(
            project_id=project_id,
            name=name,
            variable_declarations=self.variable_resolver.sanitize_variable_overrides(variable_declarations or []),
            description=description,
        )
        self.template_repo.save(tmpl)
        return tmpl

    def update_template(
        self,
        project_id: str,
        template_id: str,
        name: str | None = None,
        description: str | None = None,
        orchestration: list[StageOrchestration] | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> PipelineTemplate:
        """更新模板（支持编排和变量声明更新）"""
        tmpl = self.get_template(project_id, template_id)

        orch_changed = False
        stages = list(tmpl.stages)

        if orchestration is not None:
            stages = self._load_stages_by_ids(project_id, [o.stage_id for o in orchestration]) if orchestration else []
            stage_version_map = {s.id: s.version for s in stages}
            orchestration_with_version = [
                o.model_copy(update={"stage_version": stage_version_map[o.stage_id]}) for o in orchestration
            ]

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

    def delete_template(self, project_id: str, template_id: str) -> None:
        """删除模板（检查 webhook 引用）"""
        tmpl = self.get_template(project_id, template_id)
        webhooks = self.webhook_repo.find_by_template(template_id)
        if webhooks:
            raise BusinessError("Template is referenced by webhooks, cannot delete", status_code=409)
        self.template_repo.delete(tmpl)

    def duplicate_template(self, project_id: str, template_id: str) -> PipelineTemplate:
        """复制模板（自动生成唯一名称）"""
        tmpl = self.get_template(project_id, template_id)
        base_name = re.sub(r" copy( \d+)?$", "", tmpl.name)

        i = 1
        while True:
            new_name = f"{base_name} copy" if i == 1 else f"{base_name} copy {i}"
            if not self.template_repo.find_by_name(project_id, new_name):
                break
            i += 1

        new_template = PipelineTemplate.create(
            project_id=project_id,
            name=new_name,
            variable_declarations=list(tmpl.variable_declarations),
            description=tmpl.description,
        )
        self.template_repo.save(new_template)

        if tmpl.orchestration:
            self.template_repo.save_orchestration(new_template.id, tmpl.orchestration)

        new_template.orchestration = tmpl.orchestration
        new_template.stages = tmpl.stages
        self.template_repo.save(new_template)
        return new_template

    def resolve_template_variables(
        self,
        project_id: str,
        orchestration: list[StageOrchestration],
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> list[VariableDeclaration]:
        """解析模板变量（用于前端预览）"""
        stages = self._load_stages_by_ids(project_id, [o.stage_id for o in orchestration]) if orchestration else []
        return self.variable_resolver.resolve_template_variables(stages, variable_declarations or [])

    def get_snapshot(self, project_id: str, snapshot_id: str) -> PipelineSnapshot:
        """获取快照"""
        snapshot = self.snapshot_repo.find_by_id_in_project(project_id, snapshot_id)
        if not snapshot:
            raise BusinessError(f"PipelineSnapshot {snapshot_id} not found", status_code=404)
        return snapshot

    def _load_stages_by_ids(self, project_id: str, stage_ids: list[str]) -> list[BuildStage]:
        """批量加载 Stage（用于模板编排）"""
        stages = self.stage_repo.find_by_ids(project_id, stage_ids)
        if len(stages) != len(stage_ids):
            found = {stage.id for stage in stages}
            missing = [stage_id for stage_id in stage_ids if stage_id not in found]
            raise BusinessError(f"Stage(s) not found: {missing}", status_code=404)
        return stages
