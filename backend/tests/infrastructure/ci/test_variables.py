"""测试变量处理工具"""

import pytest

from pomelo_orbit.domain.ci.value_objects import VariableDeclaration
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    mask_secrets,
    merge_variables,
    render_template,
    validate_variables,
)


class TestMergeVariables:
    """测试变量合并"""

    def test_merge_empty_variables(self):
        """测试合并空变量"""
        result = merge_variables({}, {}, {})
        assert result == {}

    def test_merge_global_only(self):
        """测试仅全局变量"""
        result = merge_variables({"KEY": "global"}, {}, {})
        assert result == {"KEY": "global"}

    def test_merge_project_overrides_global(self):
        """测试项目变量覆盖全局变量"""
        result = merge_variables(
            {"KEY": "global", "GLOBAL_ONLY": "value"},
            {"KEY": "project"},
            {},
        )
        assert result == {"KEY": "project", "GLOBAL_ONLY": "value"}

    def test_merge_runtime_overrides_all(self):
        """测试运行时变量覆盖所有"""
        result = merge_variables(
            {"KEY": "global"},
            {"KEY": "project"},
            {"KEY": "runtime"},
        )
        assert result == {"KEY": "runtime"}

    def test_merge_all_layers(self):
        """测试三层变量合并"""
        result = merge_variables(
            {"GLOBAL": "g", "KEY": "global"},
            {"PROJECT": "p", "KEY": "project"},
            {"RUNTIME": "r", "KEY": "runtime"},
        )
        assert result == {
            "GLOBAL": "g",
            "PROJECT": "p",
            "RUNTIME": "r",
            "KEY": "runtime",
        }


class TestValidateVariables:
    """测试变量校验"""

    def test_validate_all_required_present(self):
        """测试所有必填变量都存在"""
        declarations = [
            VariableDeclaration(name="KEY1", required=True),
            VariableDeclaration(name="KEY2", required=True),
        ]
        variables = {"KEY1": "value1", "KEY2": "value2"}

        # 不应抛出异常
        validate_variables(variables, declarations)

    def test_validate_optional_missing(self):
        """测试可选变量缺失"""
        declarations = [
            VariableDeclaration(name="KEY1", required=True),
            VariableDeclaration(name="KEY2", required=False),
        ]
        variables = {"KEY1": "value1"}

        # 不应抛出异常
        validate_variables(variables, declarations)

    def test_validate_required_missing(self):
        """测试必填变量缺失"""
        declarations = [
            VariableDeclaration(name="KEY1", required=True),
            VariableDeclaration(name="KEY2", required=True),
        ]
        variables = {"KEY1": "value1"}

        with pytest.raises(VariableError, match="缺少必填变量: KEY2"):
            validate_variables(variables, declarations)


class TestRenderTemplate:
    """测试模板渲染"""

    def test_render_simple_template(self):
        """测试渲染简单模板"""
        template = "Hello {{ name }}!"
        variables = {"name": "World"}

        result = render_template(template, variables)
        assert result == "Hello World!"

    def test_render_multiple_variables(self):
        """测试渲染多个变量"""
        template = "{{ greeting }} {{ name }}!"
        variables = {"greeting": "Hello", "name": "World"}

        result = render_template(template, variables)
        assert result == "Hello World!"

    def test_render_with_filter(self):
        """测试使用过滤器渲染"""
        template = "{{ name | upper }}"
        variables = {"name": "world"}

        result = render_template(template, variables)
        assert result == "WORLD"

    def test_render_with_condition(self):
        """测试条件渲染"""
        template = "{% if enabled %}ON{% else %}OFF{% endif %}"

        result1 = render_template(template, {"enabled": True})
        assert result1 == "ON"

        result2 = render_template(template, {"enabled": False})
        assert result2 == "OFF"

    def test_render_undefined_variable(self):
        """测试未定义变量"""
        template = "Hello {{ name }}!"
        variables: dict[str, str] = {}

        # Jinja2 默认行为是渲染为空字符串，除非设置 undefined=StrictUndefined
        # 我们的实现使用默认行为，所以这个测试应该检查空字符串
        result = render_template(template, variables)
        assert result == "Hello !"

    def test_render_syntax_error(self):
        """测试模板语法错误"""
        template = "Hello {{ name"
        variables = {"name": "World"}

        with pytest.raises(VariableError, match="模板语法错误"):
            render_template(template, variables)


class TestMaskSecrets:
    """测试 secret 脱敏"""

    def test_mask_no_secrets(self):
        """测试没有 secret 变量"""
        declarations = [
            VariableDeclaration(name="KEY1", secret=False),
            VariableDeclaration(name="KEY2", secret=False),
        ]
        variables = {"KEY1": "value1", "KEY2": "value2"}

        result = mask_secrets(variables, declarations)
        assert result == {"KEY1": "value1", "KEY2": "value2"}

    def test_mask_secrets_present(self):
        """测试脱敏 secret 变量"""
        declarations = [
            VariableDeclaration(name="KEY1", secret=False),
            VariableDeclaration(name="PASSWORD", secret=True),
            VariableDeclaration(name="TOKEN", secret=True),
        ]
        variables = {"KEY1": "value1", "PASSWORD": "secret123", "TOKEN": "token456"}

        result = mask_secrets(variables, declarations)
        assert result == {"KEY1": "value1", "PASSWORD": "***", "TOKEN": "***"}

    def test_mask_secrets_not_in_variables(self):
        """测试 secret 变量不在实际变量中"""
        declarations = [
            VariableDeclaration(name="PASSWORD", secret=True),
        ]
        variables = {"KEY1": "value1"}

        result = mask_secrets(variables, declarations)
        assert result == {"KEY1": "value1"}

    def test_mask_does_not_modify_original(self):
        """测试脱敏不修改原始变量"""
        declarations = [
            VariableDeclaration(name="PASSWORD", secret=True),
        ]
        variables = {"PASSWORD": "secret123"}

        result = mask_secrets(variables, declarations)

        assert result["PASSWORD"] == "***"
        assert variables["PASSWORD"] == "secret123"  # 原始变量未修改
