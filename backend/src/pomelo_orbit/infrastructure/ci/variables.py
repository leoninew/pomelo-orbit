"""变量处理工具"""

from typing import Any

from jinja2 import Environment, TemplateSyntaxError, UndefinedError

from pomelo_orbit.domain.ci.value_objects import VariableDeclaration


class VariableError(Exception):
    """变量处理错误"""


def merge_variables(
    global_vars: dict[str, Any],
    project_vars: dict[str, Any],
    runtime_vars: dict[str, Any],
) -> dict[str, Any]:
    """
    合并三层变量，就近优先

    Args:
        global_vars: 全局变量
        project_vars: 项目级变量
        runtime_vars: 运行时临时变量

    Returns:
        合并后的变量字典
    """
    merged = {}
    merged.update(global_vars)
    merged.update(project_vars)
    merged.update(runtime_vars)
    return merged


def validate_variables(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> None:
    """
    校验变量是否满足声明要求

    Args:
        variables: 实际变量值
        declarations: 变量声明

    Raises:
        VariableError: 校验失败
    """
    for decl in declarations:
        if decl.required and decl.name not in variables:
            raise VariableError(f"缺少必填变量: {decl.name}")


def render_template(template: str, variables: dict[str, Any]) -> str:
    """
    使用 Jinja2 渲染模板

    Args:
        template: 模板文本
        variables: 变量字典

    Returns:
        渲染后的文本

    Raises:
        VariableError: 渲染失败
    """
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


def mask_secrets(variables: dict[str, Any], declarations: list[VariableDeclaration]) -> dict[str, Any]:
    """
    脱敏 secret 变量

    Args:
        variables: 变量字典
        declarations: 变量声明

    Returns:
        脱敏后的变量字典
    """
    masked = variables.copy()
    secret_names = {decl.name for decl in declarations if decl.secret}

    for name in secret_names:
        if name in masked:
            masked[name] = "***"

    return masked
