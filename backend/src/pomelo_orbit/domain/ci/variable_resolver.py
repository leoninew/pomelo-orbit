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

from dataclasses import dataclass
from typing import Any

from pomelo_orbit.domain.ci.entities import BuildStage, Repository
from pomelo_orbit.domain.ci.value_objects import BuiltinVariableSpecs, VariableDeclaration, VariableSource
from pomelo_orbit.infrastructure.ci.variables import extract_variables, merge_declarations
from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass(frozen=True)
class BuiltinVarSpec:
    """内置变量的元数据规范"""

    description: str
    editable: bool = False


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
        - 仓库内置变量（source=REPOSITORY，editable 由 spec 决定）
        - 仓库自定义变量（source=REPOSITORY_CUSTOM，editable=True）
        """
        specs = self._get_repository_builtin_specs()
        builtin_values = self._build_repository_builtin_variables(repository, repository.default_branch)
        builtin_vars = [
            VariableDeclaration(
                name=name,
                default=builtin_values[name],
                source=VariableSource.REPOSITORY,
                description=spec.description,
                editable=spec.editable,
            )
            for name, spec in specs.items()
        ]

        custom_vars = [
            v.model_copy(update={"source": VariableSource.REPOSITORY_CUSTOM, "editable": True})
            for v in (repository.variable_overrides or [])
        ]

        return builtin_vars + custom_vars

    # ═══════════════════════════════════════════════════════════════════════════
    # 场景 2 & 3: 模板详情界面 + 模板编排 Stage - 计算变量列表
    # ═══════════════════════════════════════════════════════════════════════════

    def resolve_template_variables(
        self,
        stages: list[BuildStage],
        custom_declarations: list[VariableDeclaration],
    ) -> list[VariableDeclaration]:
        """解析模板变量（用于模板详情页展示 & 前端编排 Stage 时实时计算）

        返回：
        - 模板内置变量（source=TEMPLATE，editable=False）
        - 仓库内置变量（source=TEMPLATE，editable=False）
        - Stage 解析变量（source=TEMPLATE_STAGE，editable=True）
        - 模板自定义变量（source=TEMPLATE_CUSTOM，editable=True）
        """
        stage_defs = [stage.to_stage_definition() for stage in stages]
        extracted = extract_variables(stage_defs)

        all_specs = {**self._get_repository_builtin_specs(), **self._get_template_builtin_specs()}
        all_builtin_specs: BuiltinVariableSpecs = {name: spec.description for name, spec in all_specs.items()}
        all_editable_map: dict[str, bool] = {name: spec.editable for name, spec in all_specs.items()}

        declarations = merge_declarations(
            extracted_variables=extracted,
            existing_declarations=custom_declarations,
            builtin_specs=all_builtin_specs,
            source_for_builtin=VariableSource.TEMPLATE,
            source_for_extracted=VariableSource.TEMPLATE_STAGE,
        )

        # editable：内置变量查 spec，其余默认 True
        return [decl.model_copy(update={"editable": all_editable_map.get(decl.name, True)}) for decl in declarations]

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
        1. 内置变量（全局 + 仓库 + 模板）- 系统注入的运行时事实，不可覆盖
        2. 运行时覆盖变量（用户触发时传入，不能覆盖内置变量）
        3. 仓库自定义变量
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

        # 4. 运行时覆盖变量（用户触发时传入，优先级高于仓库自定义和模板声明）
        if runtime_overrides:
            result.update({n: v for n, v in runtime_overrides.items() if n not in builtin_names})

        # 5. 仓库自定义变量（未被运行时覆盖时才填入）
        for var in repository.variable_overrides or []:
            if var.name not in result and var.value is not None:
                result[var.name] = var.value

        # 6. 模板自定义变量（value 优先，无 value 时用 default）
        for decl in template.variable_declarations:
            if decl.name not in result:
                effective = decl.value if decl.value is not None else decl.default
                if effective is not None:
                    result[decl.name] = effective

        # 7. Stage 提取变量（value 优先，无 value 时用 default，最低优先级）
        for decl in stage_declarations or []:
            if decl.source == VariableSource.TEMPLATE_STAGE and decl.name not in result:
                effective = decl.value if decl.value is not None else decl.default
                if effective is not None:
                    result[decl.name] = effective

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
                # 用户显式设置了 stage 变量的覆盖值，升级为 template_custom 持久化
                # （default 字段保留，value 是用户的覆盖）
                result.append(var.model_copy(update={"source": VariableSource.TEMPLATE_CUSTOM}))

        return result

    def _get_repository_builtin_specs(self) -> dict[str, BuiltinVarSpec]:
        """仓库内置变量规范：description 和 editable"""
        return {
            "repository_id": BuiltinVarSpec("运行时注入: 当前项目 ID", editable=False),
            "repository_name": BuiltinVarSpec("运行时注入: 当前项目名称", editable=False),
            "repository_code": BuiltinVarSpec("运行时注入: 当前项目编码", editable=False),
            "repository_url": BuiltinVarSpec("运行时注入: 当前仓库地址", editable=False),
            "repository_ref": BuiltinVarSpec("运行时注入: 当前分支", editable=True),
        }

    def _get_template_builtin_specs(self) -> dict[str, BuiltinVarSpec]:
        """模板内置变量规范：description 和 editable"""
        return {
            "template_id": BuiltinVarSpec("运行时注入: 当前模板 ID", editable=False),
            "template_name": BuiltinVarSpec("运行时注入: 当前模板名称", editable=False),
            "template_version": BuiltinVarSpec("运行时注入: 当前模板版本", editable=False),
            "runtime_datetime": BuiltinVarSpec("运行时注入: 流水线启动时间 (UTC, 格式 YYYYmmdd-HHmmss)", editable=False),
        }

    def _build_repository_builtin_variables(self, repository: Repository, trigger_ref: str) -> dict[str, Any]:
        """构建仓库内置变量字典"""
        return {
            "repository_id": repository.id,
            "repository_name": repository.name,
            "repository_code": repository.code,
            "repository_url": repository.repository_url,
            "repository_ref": trigger_ref,
        }

    def _build_template_builtin_variables(self, template: Any) -> dict[str, Any]:
        """构建模板内置变量字典"""
        return {
            "template_id": template.id,
            "template_name": template.name,
            "template_version": template.version,
            "runtime_datetime": utc_now().strftime("%Y%m%d-%H%M%S"),
        }
