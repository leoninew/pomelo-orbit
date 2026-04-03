"""变量处理工具"""

import re
from copy import deepcopy
from typing import Any

from jinja2 import Environment, TemplateSyntaxError, UndefinedError

from pomelo_orbit.domain.ci.value_objects import (
    CheckoutStageConfig,
    DockerBuildStageConfig,
    StageDefinition,
    StageType,
    UnitTestStageConfig,
    VariableDeclaration,
)

# {{ VAR_NAME }} 格式，VAR_NAME 为大写字母、数字、下划线
PLACEHOLDER_PATTERN = re.compile(r"\{\{\s*([A-Z][A-Z0-9_]*)\s*\}\}")

# 内置变量名称集合，由项目属性自动注入，不需要用户声明
BUILTIN_VARIABLES = frozenset(
    {
        "REPOSITORY_URL",
        "DEFAULT_BRANCH",
        "GIT_CREDENTIAL_ID",
        "trigger_ref",
        "trigger_type",
        "project_name",
    }
)


class VariableError(Exception):
    """变量处理错误"""


def extract_variables(stages: list[StageDefinition]) -> set[str]:
    """
    从 Stage 列表提取所有非内置变量占位符名称（去重）。

    扫描范围：
    - checkout: ref
    - docker_build: context, dockerfile, image_name
    - unit_test: image, 每条 command, 每个 artifact_path
    - custom: 所有 StepDefinition 字符串字段（image, commands, volumes）
    """
    found: set[str] = set()

    def _scan(text: str) -> None:
        for match in PLACEHOLDER_PATTERN.finditer(text):
            name = match.group(1)
            if name not in BUILTIN_VARIABLES:
                found.add(name)

    def _scan_list(items: list[str] | None) -> None:
        for item in items or []:
            _scan(item)

    for stage in stages:
        if stage.type == StageType.CHECKOUT and isinstance(stage.config, CheckoutStageConfig):
            _scan(stage.config.ref)
        elif stage.type == StageType.DOCKER_BUILD and isinstance(stage.config, DockerBuildStageConfig):
            _scan(stage.config.context)
            _scan(stage.config.dockerfile)
            _scan(stage.config.image_name)
        elif stage.type == StageType.UNIT_TEST and isinstance(stage.config, UnitTestStageConfig):
            _scan(stage.config.image)
            _scan_list(stage.config.commands)
            _scan_list(stage.config.artifact_paths)
        elif stage.type == StageType.CUSTOM:
            for step in stage.steps or []:
                if step.image:
                    _scan(step.image)
                _scan_list(step.commands)
                for vol in step.volumes or []:
                    _scan(vol)

    return found


def merge_declarations(
    extracted_names: set[str],
    existing_declarations: list[VariableDeclaration],
) -> list[VariableDeclaration]:
    """
    将提取到的变量名与现有声明合并：
    - 新出现的名称补充为空声明（仅有名称）
    - 已删除的名称从声明列表移除
    - 现有声明的元数据（description/required/default/secret/locked）保留
    """
    existing_map = {d.name: d for d in existing_declarations}
    result = []
    for name in sorted(extracted_names):
        if name in existing_map:
            result.append(existing_map[name])
        else:
            result.append(VariableDeclaration(name=name))
    return result


def merge_variables(
    global_vars: dict[str, Any],
    project_vars: dict[str, Any],
    runtime_vars: dict[str, Any],
    declarations: list[VariableDeclaration],
    builtin_vars: dict[str, Any],
) -> dict[str, Any]:
    """
    按优先级合并变量（高→低）：
    内置变量（不可覆盖）> 运行时临时变量（非 locked）> 项目级变量 > 全局变量 > 声明默认值
    """
    result: dict[str, Any] = {}

    # 1. 声明默认值（最低优先级）
    for decl in declarations:
        if decl.default is not None:
            result[decl.name] = decl.default

    # 2. 全局变量
    result.update(global_vars)

    # 3. 项目级变量
    result.update(project_vars)

    # 4. 运行时临时变量（locked 变量跳过，不可被临时变量覆盖）
    locked_names = {d.name for d in declarations if d.locked}
    result.update({k: v for k, v in runtime_vars.items() if k not in locked_names})

    # 5. 内置变量（最高优先级，强制覆盖）
    result.update(builtin_vars)

    return result


def validate_variables(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> None:
    """
    校验所有 required 变量在合并后变量表中是否有值。

    Raises:
        VariableError: 存在缺失变量时，错误信息包含所有缺失变量名称
    """
    missing = [d.name for d in declarations if d.required and d.name not in variables]
    if missing:
        raise VariableError(f"缺少必填变量: {', '.join(missing)}")


def render_template(template: str, variables: dict[str, Any]) -> str:
    """使用 Jinja2 渲染模板字符串"""
    try:
        env = Environment(autoescape=False)
        jinja_template = env.from_string(template)
        return jinja_template.render(**variables)
    except TemplateSyntaxError as e:
        raise VariableError(f"模板语法错误: {e}")
    except UndefinedError as e:
        raise VariableError(f"变量未定义: {e}")
    except Exception as e:
        raise VariableError(f"模板渲染失败: {e}")


def resolve_stage(stage: StageDefinition, variables: dict[str, Any]) -> StageDefinition:
    """
    将 Stage 中所有字符串字段的占位符替换为变量值，返回新的 StageDefinition。
    """

    def r(text: str) -> str:
        return render_template(text, variables)

    def r_list(items: list[str]) -> list[str]:
        return [r(item) for item in items]

    stage = deepcopy(stage)

    if stage.type == StageType.CHECKOUT and isinstance(stage.config, CheckoutStageConfig):
        stage.config.ref = r(stage.config.ref)
    elif stage.type == StageType.DOCKER_BUILD and isinstance(stage.config, DockerBuildStageConfig):
        stage.config.context = r(stage.config.context)
        stage.config.dockerfile = r(stage.config.dockerfile)
        stage.config.image_name = r(stage.config.image_name)
    elif stage.type == StageType.UNIT_TEST and isinstance(stage.config, UnitTestStageConfig):
        stage.config.image = r(stage.config.image)
        stage.config.commands = r_list(stage.config.commands)
        stage.config.artifact_paths = r_list(stage.config.artifact_paths)
    elif stage.type == StageType.CUSTOM:
        for step in stage.steps or []:
            if step.image:
                step.image = r(step.image)
            if step.commands:
                step.commands = r_list(step.commands)
            if step.volumes:
                step.volumes = r_list(step.volumes)

    return stage


def mask_secrets(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> dict[str, Any]:
    """脱敏 secret 变量"""
    masked = variables.copy()
    for decl in declarations:
        if decl.secret and decl.name in masked:
            masked[decl.name] = "***"
    return masked
