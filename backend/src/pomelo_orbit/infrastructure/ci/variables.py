"""变量处理工具"""

import re
from copy import deepcopy
from typing import Any

from jinja2 import Environment, StrictUndefined, TemplateSyntaxError, UndefinedError, nodes

from pomelo_orbit.domain.ci.value_objects import (
    BuiltinVariableSpecs,
    StageDefinition,
    VariableDeclaration,
    VariableSource,
)

# 兜底：语法解析失败时仅提取最简单的 {{ VAR_NAME }} 占位符
PLACEHOLDER_PATTERN = re.compile(r"\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}")


class VariableError(Exception):
    """变量处理错误"""


def _iter_stage_templates(stages: list[StageDefinition]) -> list[str]:
    texts: list[str] = []
    for stage in stages:
        texts.append(stage.script or "")
        if stage.artifacts:
            texts.extend(a.path for a in stage.artifacts)
            texts.extend(a.name for a in stage.artifacts)
    return texts


def _record_default(found: dict[str, Any], name: str, default: Any) -> None:
    current = found.get(name)
    if name not in found:
        found[name] = default
        return
    if current is None and default is not None:
        found[name] = default


def _walk_ast(node: nodes.Node, found: dict[str, Any]) -> None:
    if isinstance(node, nodes.Name):
        found.setdefault(node.name, None)

    if isinstance(node, nodes.Filter) and node.name in {"default", "d"} and isinstance(node.node, nodes.Name):
        default_value = node.args[0].value if node.args and isinstance(node.args[0], nodes.Const) else None
        _record_default(found, node.node.name, default_value)

    for child in node.iter_child_nodes():
        _walk_ast(child, found)


def extract_variables(stages: list[StageDefinition]) -> dict[str, Any]:
    """从 Stage 列表提取所有变量名及 Jinja default 过滤器中的默认值。

    如果模板语法错误，抛出异常而不是静默处理。
    """
    env = Environment(autoescape=False)
    found: dict[str, Any] = {}

    for text in _iter_stage_templates(stages):
        if not text:
            continue
        ast = env.parse(text)
        _walk_ast(ast, found)

    return found


def _build_builtin_declaration(
    name: str,
    default: Any,
    source: VariableSource,
    builtin_specs: BuiltinVariableSpecs | None = None,
) -> VariableDeclaration:
    description = (builtin_specs or {}).get(name, "")
    return VariableDeclaration(
        name=name,
        description=description,
        default=default,
        source=source,
        editable=False,
    )


def merge_declarations(
    extracted_variables: dict[str, Any],
    existing_declarations: list[VariableDeclaration],
    builtin_specs: BuiltinVariableSpecs | None = None,
    source_for_builtin: VariableSource = VariableSource.TEMPLATE,
    source_for_extracted: VariableSource = VariableSource.TEMPLATE_STAGE,
) -> list[VariableDeclaration]:
    """
    将 Stage 提取结果、现有声明和内置变量合并为模板展示用变量列表。

    规则：
    - 如果没有提取到任何变量（无 stage 或 stage 为空），显示所有内置变量 + 所有自定义变量
    - 如果有提取到变量，只显示 stage 中实际用到的变量（包括内置变量和自定义变量）
    - Stage 中提取的变量标记为 TEMPLATE_STAGE
    - 用户自定义的变量保持原有 source（默认 TEMPLATE_CUSTOM）
    """
    declared_builtin_names = set(builtin_specs.keys()) if builtin_specs else set()

    merged = []
    merged_names = set()

    # 如果没有提取到任何变量（无 stage），显示所有内置变量 + 所有自定义变量
    if not extracted_variables:
        merged = [
            _build_builtin_declaration(name, None, source_for_builtin, builtin_specs)
            for name in sorted(declared_builtin_names)
        ]
        merged_names = {decl.name for decl in merged}

        for existing in existing_declarations:
            if existing.name not in merged_names and existing.name not in declared_builtin_names:
                merged.append(existing.model_copy(deep=True))
                merged_names.add(existing.name)
    else:
        existing_map = {d.name: d for d in existing_declarations}

        for name in sorted(extracted_variables):
            if name in declared_builtin_names:
                merged.append(_build_builtin_declaration(name, None, source_for_builtin, builtin_specs))
                merged_names.add(name)
            else:
                stage_default = extracted_variables[name]
                existing_decl = existing_map.get(name)

                if existing_decl is not None:
                    # 保留用户已有的声明（value、description、secret 等），补充 stage default
                    decl = existing_decl.model_copy(deep=True)
                    if decl.default is None and stage_default is not None:
                        decl = decl.model_copy(update={"default": stage_default})
                else:
                    decl = VariableDeclaration(
                        name=name,
                        default=stage_default,
                        source=source_for_extracted,
                    )
                merged.append(decl)
                merged_names.add(name)

    return merged


def merge_variables(
    builtin_vars: dict[str, Any],
    project_vars: list[VariableDeclaration],
    runtime_vars: dict[str, Any],
    declarations: list[VariableDeclaration],
) -> dict[str, Any]:
    """
    按优先级合并变量（高 -> 低）：
    内置全局变量 > 运行时覆盖 > 项目级变量 > 模板声明值
    """
    result: dict[str, Any] = {}
    allowed_names = {decl.name for decl in declarations}

    for decl in declarations:
        if (
            decl.source not in {VariableSource.GLOBAL, VariableSource.REPOSITORY, VariableSource.TEMPLATE}
            and decl.value is not None
        ):
            result[decl.name] = decl.value

    builtin_names = set(builtin_vars)
    project_values = {
        decl.name: decl.value
        for decl in project_vars
        if decl.name in allowed_names and decl.name not in builtin_names and decl.value is not None
    }
    result.update(project_values)
    result.update({k: v for k, v in runtime_vars.items() if k in allowed_names and k not in builtin_names})
    result.update(builtin_vars)

    return result


def _has_value(value: Any) -> bool:
    if value is None:
        return False
    if isinstance(value, str):
        return value.strip() != ""
    return True


def validate_variables(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> None:
    """校验所有变量在合并后都已有值。

    template_stage 变量有 default 值时不强制校验（default 已在合并阶段作为兜底填入）。
    """
    missing = [
        decl.name
        for decl in declarations
        if not _has_value(variables.get(decl.name))
        and not (decl.source == VariableSource.TEMPLATE_STAGE and _has_value(decl.default))
    ]
    if missing:
        raise VariableError(f"缺少变量值: {', '.join(missing)}")


def render_template(template: str, variables: dict[str, Any]) -> str:
    """使用 Jinja2 渲染模板字符串。"""
    try:
        env = Environment(autoescape=False, undefined=StrictUndefined)
        return env.from_string(template).render(**variables)
    except TemplateSyntaxError as e:
        raise VariableError(f"模板语法错误: {e}") from e
    except UndefinedError as e:
        raise VariableError(f"变量未定义: {e}") from e
    except Exception as e:
        raise VariableError(f"模板渲染失败: {e}") from e


def resolve_stage(stage: StageDefinition, variables: dict[str, Any]) -> StageDefinition:
    """将 Stage 中所有占位符替换为变量值。"""

    def render_value(text: str) -> str:
        return render_template(text, variables)

    stage = deepcopy(stage)
    stage.script = render_value(stage.script)
    if stage.artifacts:
        for artifact in stage.artifacts:
            artifact.path = render_value(artifact.path)
            artifact.name = render_value(artifact.name)
    return stage


def mask_secrets(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> list[VariableDeclaration]:
    """脱敏 secret 变量，返回完整的 VariableDeclaration 列表。"""
    result: list[VariableDeclaration] = []
    for decl in declarations:
        value = variables.get(decl.name)
        if decl.secret:
            value = "***"
        result.append(
            VariableDeclaration(
                name=decl.name,
                default=decl.default,
                value=value,
                description=decl.description,
                secret=decl.secret,
                source=decl.source,
                editable=decl.editable,
            )
        )
    return result
