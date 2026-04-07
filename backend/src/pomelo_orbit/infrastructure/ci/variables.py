"""变量处理工具"""

import re
from copy import deepcopy
from typing import Any

from jinja2 import Environment, TemplateSyntaxError, UndefinedError

from pomelo_orbit.domain.ci.value_objects import (
    StageDefinition,
    VariableDeclaration,
)

# 匹配所有 {{ VAR_NAME }} 占位符，大小写均可
PLACEHOLDER_PATTERN = re.compile(r"\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}")


class VariableError(Exception):
    """变量处理错误"""


def extract_variables(stages: list[StageDefinition]) -> set[str]:
    """从 Stage 列表提取所有占位符变量名"""
    all_text: list[str] = []
    for stage in stages:
        all_text.append(stage.script or "")
        all_text.extend((stage.env or {}).values())
        if stage.artifacts:
            all_text.extend(a.path for a in stage.artifacts)
            all_text.extend(a.name for a in stage.artifacts)

    return {m.group(1) for text in all_text for m in PLACEHOLDER_PATTERN.finditer(text)}


def merge_declarations(
    extracted_names: set[str],
    existing_declarations: list[VariableDeclaration],
) -> list[VariableDeclaration]:
    """
    将提取到的变量名与现有声明合并：
    - 新出现的名称补充为空声明
    - 已删除的名称从声明列表移除
    - 现有声明的元数据（default/secret 等）保留
    """
    existing_map = {d.name: d for d in existing_declarations}
    return [
        existing_map[name] if name in existing_map else VariableDeclaration(name=name)
        for name in sorted(extracted_names)
    ]


def merge_variables(
    global_vars: dict[str, Any],
    project_vars: dict[str, Any],
    runtime_vars: dict[str, Any],
    declarations: list[VariableDeclaration],
    builtin_vars: dict[str, Any],
) -> dict[str, Any]:
    """
    按优先级合并变量（高→低）：
    内置（项目属性+运行时上下文）> 运行时临时变量 > 项目级变量 > 全局变量 > 声明默认值
    """
    result: dict[str, Any] = {}

    for decl in declarations:
        if decl.default is not None:
            result[decl.name] = decl.default

    result.update(global_vars)
    result.update(project_vars)

    locked_names = {d.name for d in declarations if d.locked}
    result.update({k: v for k, v in runtime_vars.items() if k not in locked_names})

    result.update(builtin_vars)

    return result


def validate_variables(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> None:
    """校验所有 required 变量在合并后变量表中是否有值"""
    missing = [d.name for d in declarations if d.required and d.name not in variables]
    if missing:
        raise VariableError(f"缺少必填变量: {', '.join(missing)}")


def render_template(template: str, variables: dict[str, Any]) -> str:
    """使用 Jinja2 渲染模板字符串"""
    try:
        env = Environment(autoescape=False)
        return env.from_string(template).render(**variables)
    except TemplateSyntaxError as e:
        raise VariableError(f"模板语法错误: {e}")
    except UndefinedError as e:
        raise VariableError(f"变量未定义: {e}")
    except Exception as e:
        raise VariableError(f"模板渲染失败: {e}")


def resolve_stage(stage: StageDefinition, variables: dict[str, Any]) -> StageDefinition:
    """将 Stage 中所有占位符替换为变量值"""

    def r(text: str) -> str:
        return render_template(text, variables)

    stage = deepcopy(stage)
    stage.script = r(stage.script)
    stage.env = {k: r(v) for k, v in stage.env.items()}
    if stage.artifacts:
        for artifact in stage.artifacts:
            artifact.path = r(artifact.path)
            artifact.name = r(artifact.name)
    return stage


def mask_secrets(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> dict[str, Any]:
    """脱敏 secret 变量"""
    masked = variables.copy()
    for decl in declarations:
        if decl.secret and decl.name in masked:
            masked[decl.name] = "***"
    return masked
