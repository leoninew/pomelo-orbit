"""变量解析领域服务

变量来源分类：
1. 全局内置变量：来自配置（如 CI_ENV），所有场景可用
2. 仓库内置变量：运行时注入（如 repository_id, repository_url）
3. 仓库自定义变量：用户在仓库上定义的变量
4. 模板内置变量：运行时注入（如 template_id, trigger_ref）
5. Stage 解析变量：从 Stage 脚本中提取的变量
6. 模板自定义变量：用户在模板上定义的变量

业务场景：
1. 仓库详情界面：展示仓库内置变量 + 自定义变量（只有自定义变量可修改）
2. 模板详情界面：展示模板内置变量 + Stage 解析变量 + 自定义变量（只有自定义变量可修改）
3. 模板编排 Stage：计算变量列表（用于前端实时预览）
4. 运行流水线：合并所有来源的变量并校验

来源标签映射：
- 仓库详情界面：内置变量显示为"项目运行时"，自定义变量显示为"项目自定义"
- 模板详情界面：内置变量显示为"运行时"，Stage变量显示为"Stage"，自定义变量显示为"自定义"
"""

from typing import Any

from pomelo_orbit.domain.ci.entities import PipelineStage, Repository
from pomelo_orbit.domain.ci.value_objects import BuiltinVariableSpecs, VariableDeclaration, VariableSource
from pomelo_orbit.infrastructure.ci.variables import extract_variables, merge_declarations


class VariableResolver:
    """变量解析领域服务"""

    def __init__(self, global_variables: dict[str, Any] | None = None):
        self.global_variables = global_variables or {}

    # ═══════════════════════════════════════════════════════════════════════════
    # 场景 1: 仓库详情界面 - 展示仓库内置变量 + 自定义变量
    # ═══════════════════════════════════════════════════════════════════════════

    def get_repository_variables(self, repository: Repository) -> list[VariableDeclaration]:
        """获取仓库变量列表（用于仓库详情页展示）

        返回：
        - 仓库内置变量（source=REPOSITORY，前端显示为"项目运行时"）
        - 仓库自定义变量（source=REPOSITORY_CUSTOM，前端显示为"项目自定义"）
        """
        builtin_specs = self._get_repository_builtin_specs()
        builtin_values = self._build_repository_builtin_variables(repository, repository.default_branch)
        builtin_vars = [
            VariableDeclaration(
                name=name,
                value=builtin_values[name],
                source=VariableSource.REPOSITORY,
                description=builtin_specs[name],
            )
            for name in builtin_specs
        ]

        custom_vars = [
            v.model_copy(update={"source": VariableSource.REPOSITORY_CUSTOM})
            for v in (repository.variable_overrides or [])
        ]

        return builtin_vars + custom_vars

    # ═══════════════════════════════════════════════════════════════════════════
    # 场景 2 & 3: 模板详情界面 + 模板编排 Stage - 计算变量列表
    # ═══════════════════════════════════════════════════════════════════════════

    def resolve_template_variables(
        self,
        stages: list[PipelineStage],
        custom_declarations: list[VariableDeclaration],
    ) -> list[VariableDeclaration]:
        """解析模板变量（用于模板详情页展示 & 前端编排 Stage 时实时计算）

        返回：
        - 模板内置变量（source=TEMPLATE，前端显示为"运行时"）
        - 仓库内置变量（source=TEMPLATE，前端显示为"运行时"）
        - Stage 解析变量（source=TEMPLATE_STAGE，前端显示为"Stage"）
        - 模板自定义变量（source=TEMPLATE_CUSTOM，前端显示为"自定义"）

        注意：不包含仓库变量的实际值，仓库变量是运行时才注入的
        """
        # 从 Stage 脚本中提取变量
        stage_defs = [stage.to_stage_definition() for stage in stages]
        extracted = extract_variables(stage_defs)

        # 合并所有内置变量规范（模板 + 仓库），value=None 表示运行时注入
        all_builtin_specs: BuiltinVariableSpecs = {
            **self._get_repository_builtin_specs(),
            **self._get_template_builtin_specs(),
        }

        return merge_declarations(
            extracted_variables=extracted,
            existing_declarations=custom_declarations,
            builtin_specs=all_builtin_specs,
            source_for_builtin=VariableSource.TEMPLATE,
            source_for_extracted=VariableSource.TEMPLATE_STAGE,
        )

    # ═══════════════════════════════════════════════════════════════════════════
    # 场景 4: 运行流水线 - 合并所有来源的变量
    # ═══════════════════════════════════════════════════════════════════════════

    def build_runtime_variables(
        self,
        repository: Repository,
        template: Any,  # PipelineTemplate
        trigger_ref: str,
        runtime_overrides: dict[str, Any] | None = None,
        stage_declarations: list[VariableDeclaration] | None = None,
    ) -> dict[str, Any]:
        """构建运行时变量（用于流水线执行）

        合并优先级（高 -> 低）：
        1. 内置变量（全局 + 仓库 + 模板）- 不可覆盖
        2. 仓库自定义变量
        3. 运行时覆盖变量（用户触发时传入，不能覆盖内置变量）
        4. 模板声明变量（template_custom）
        5. Stage 提取变量默认值（template_stage）
        """
        # 1. 全局内置变量
        result = dict(self.global_variables)

        # 2. 仓库内置变量
        result.update(self._build_repository_builtin_variables(repository, trigger_ref))

        # 3. 模板内置变量
        result.update(self._build_template_builtin_variables(template))

        builtin_names = self.get_builtin_variable_names()

        # 4. 仓库自定义变量
        for var in repository.variable_overrides or []:
            if var.name not in result and var.value is not None:
                result[var.name] = var.value

        # 5. 运行时覆盖变量（用户触发时传入，可覆盖 template_stage 和 template_custom）
        if runtime_overrides:
            result.update({n: v for n, v in runtime_overrides.items() if n not in builtin_names})

        # 6. 模板自定义变量默认值（未被运行时覆盖时才填入）
        for decl in template.variable_declarations:
            if decl.name not in result and decl.value is not None:
                result[decl.name] = decl.value

        # 7. Stage 提取变量默认值（最低优先级，未被任何上层覆盖时才填入）
        for decl in stage_declarations or []:
            if decl.source == VariableSource.TEMPLATE_STAGE and decl.name not in result and decl.value is not None:
                result[decl.name] = decl.value

        return result

    # ═══════════════════════════════════════════════════════════════════════════
    # 辅助方法
    # ═══════════════════════════════════════════════════════════════════════════

    def get_builtin_variable_names(self) -> set[str]:
        """获取所有内置变量名称"""
        return (
            set(self.global_variables)
            | set(self._get_repository_builtin_specs().keys())
            | set(self._get_template_builtin_specs().keys())
        )

    def sanitize_variable_overrides(self, variables: list[VariableDeclaration] | None) -> list[VariableDeclaration]:
        """过滤掉内置变量，只保留可持久化的自定义变量。

        规则：
        1. 保留 source=template_custom 或 source=repository_custom 的变量
        2. template_stage 变量若用户已设置值，升级为 template_custom 持久化
        3. 排除所有内置变量名称（即使 source 是 custom）
        """
        if not variables:
            return []

        builtin_names = self.get_builtin_variable_names()
        result = []

        for var in variables:
            if var.name in builtin_names:
                continue
            if var.source in {VariableSource.TEMPLATE_CUSTOM, VariableSource.REPOSITORY_CUSTOM}:
                result.append(var)
            elif var.source == VariableSource.TEMPLATE_STAGE and var.value is not None:
                # 用户显式设置了 stage 变量的值，升级为 template_custom 持久化
                result.append(var.model_copy(update={"source": VariableSource.TEMPLATE_CUSTOM}))

        return result

    def _get_repository_builtin_specs(self) -> BuiltinVariableSpecs:
        """仓库内置变量规范（用于生成描述）"""
        return {
            "repository_id": "运行时注入: 当前项目 ID",
            "repository_name": "运行时注入: 当前项目名称",
            "repository_url": "运行时注入: 当前仓库地址",
            "repository_ref": "运行时注入: 当前分支",
        }

    def _get_template_builtin_specs(self) -> BuiltinVariableSpecs:
        """模板内置变量规范（用于生成描述）"""
        return {
            "template_id": "运行时注入: 当前模板 ID",
            "template_name": "运行时注入: 当前模板名称",
            "template_version": "运行时注入: 当前模板版本",
        }

    def _build_repository_builtin_variables(self, repository: Repository, trigger_ref: str) -> dict[str, Any]:
        """构建仓库内置变量字典"""
        return {
            "repository_id": repository.id,
            "repository_name": repository.name,
            "repository_url": repository.repository_url,
            "repository_ref": trigger_ref,
        }

    def _build_template_builtin_variables(self, template: Any) -> dict[str, Any]:
        """构建模板内置变量字典"""
        return {
            "template_id": template.id,
            "template_name": template.name,
            "template_version": template.version,
        }
