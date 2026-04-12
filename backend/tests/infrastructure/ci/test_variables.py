"""测试变量处理工具"""

import pytest

from pomelo_orbit.domain.ci.value_objects import (
    BuiltinVariableSpecs,
    StageDefinition,
    VariableDeclaration,
    VariableSource,
)
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    extract_variables,
    mask_secrets,
    merge_declarations,
    merge_variables,
    render_template,
    validate_variables,
)


class TestExtractVariables:
    """测试变量提取"""

    def test_extracts_plain_variables(self):
        stages = [
            StageDefinition(
                id="build",
                name="build",
                image="alpine",
                script="echo {{ IMAGE_NAME }}",
                version=1,
            )
        ]

        assert extract_variables(stages) == {"IMAGE_NAME": None}

    def test_extracts_default_filter_values(self):
        stages = [
            StageDefinition(
                id="build",
                name="build",
                image="alpine",
                script="cd {{ working_dir | default('.') }} && echo {{ retry | default(3) }}",
                version=1,
            )
        ]

        assert extract_variables(stages) == {
            "working_dir": ".",
            "retry": 3,
        }


class TestMergeDeclarations:
    """测试变量声明合并"""

    def test_builtin_variables_are_always_included(self):
        # 当没有提取到变量时（无 stage），显示所有内置变量
        result = merge_declarations(
            {},
            [],
            builtin_specs={"GLOBAL_ENV": "全局环境变量"},
            source_for_builtin=VariableSource.TEMPLATE,
        )

        assert [item.name for item in result] == ["GLOBAL_ENV"]
        assert result[0].source == VariableSource.TEMPLATE
        assert result[0].value is None

    def test_existing_user_metadata_is_preserved(self):
        result = merge_declarations(
            {"IMAGE_TAG": "latest"},
            [
                VariableDeclaration(
                    name="IMAGE_TAG", description="镜像标签", secret=True, source=VariableSource.TEMPLATE_CUSTOM
                )
            ],
            source_for_builtin=VariableSource.TEMPLATE,
        )

        assert result[0].name == "IMAGE_TAG"
        assert result[0].description == "镜像标签"
        assert result[0].secret is True
        assert result[0].default == "latest"  # stage 提取的默认值存入 default
        assert result[0].value is None  # 用户未覆盖

    def test_builtin_name_hides_user_override(self):
        result = merge_declarations(
            {"GLOBAL_ENV": "stage"},
            [VariableDeclaration(name="GLOBAL_ENV", value="local", source=VariableSource.TEMPLATE_CUSTOM)],
            builtin_specs={"GLOBAL_ENV": "全局环境变量"},
            source_for_builtin=VariableSource.TEMPLATE,
        )

        assert len(result) == 1
        assert result[0].name == "GLOBAL_ENV"
        assert result[0].source == VariableSource.TEMPLATE
        assert result[0].value is None
        assert result[0].default is None  # 内置变量 default 在运行时才注入

    def test_runtime_builtin_is_only_included_when_referenced(self):
        runtime_specs: BuiltinVariableSpecs = {
            "repository_trigger_ref": "运行时注入: 本次触发 Ref",
        }

        # 无 stage 时，内置变量会显示
        result_no_stage = merge_declarations(
            {},
            [],
            builtin_specs=runtime_specs,
            source_for_builtin=VariableSource.TEMPLATE,
        )
        assert len(result_no_stage) == 1
        assert result_no_stage[0].name == "repository_trigger_ref"

        # 有 stage 但未引用该变量时，不显示
        result_not_referenced = merge_declarations(
            {"OTHER_VAR": None},
            [],
            builtin_specs=runtime_specs,
            source_for_builtin=VariableSource.TEMPLATE,
        )
        assert all(d.name != "repository_trigger_ref" for d in result_not_referenced)

        # 有 stage 且引用了该变量时，显示
        result = merge_declarations(
            {"repository_trigger_ref": None},
            [],
            builtin_specs=runtime_specs,
            source_for_builtin=VariableSource.TEMPLATE,
        )

        assert len(result) == 1
        assert result[0].name == "repository_trigger_ref"
        assert result[0].source == VariableSource.TEMPLATE
        assert result[0].description == "运行时注入: 本次触发 Ref"

    def test_user_custom_variables_are_preserved(self):
        """测试用户自定义的变量在 Stage 中使用时会保留元数据"""
        result = merge_declarations(
            {"IMAGE_TAG": "latest"},  # 从 Stage 提取的变量
            [
                VariableDeclaration(name="IMAGE_TAG", description="镜像标签", source=VariableSource.TEMPLATE_CUSTOM),
                VariableDeclaration(
                    name="CUSTOM_VAR",
                    value="custom_value",
                    description="用户自定义",
                    source=VariableSource.TEMPLATE_CUSTOM,
                ),
            ],
            source_for_builtin=VariableSource.TEMPLATE,
            source_for_extracted=VariableSource.TEMPLATE_STAGE,
        )

        # 只有在 Stage 中实际使用的变量才会被保留
        assert len(result) == 1
        # IMAGE_TAG 从 Stage 提取，但保留用户元数据
        assert result[0].name == "IMAGE_TAG"
        assert result[0].source == VariableSource.TEMPLATE_CUSTOM
        assert result[0].description == "镜像标签"
        assert result[0].default == "latest"  # stage 提取的默认值存入 default
        assert result[0].value is None  # 用户未覆盖


class TestMergeVariables:
    """测试变量合并"""

    def test_merge_empty_variables(self):
        result = merge_variables({}, [], {}, [])
        assert result == {}

    def test_project_variables_override_template_defaults(self):
        declarations = [VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)]

        result = merge_variables(
            {},
            [VariableDeclaration(name="IMAGE_TAG", value="stable", source=VariableSource.REPOSITORY_CUSTOM)],
            {},
            declarations,
        )

        assert result == {"IMAGE_TAG": "stable"}

    def test_runtime_variables_override_project_variables(self):
        declarations = [VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)]

        result = merge_variables(
            {},
            [VariableDeclaration(name="IMAGE_TAG", value="stable", source=VariableSource.REPOSITORY_CUSTOM)],
            {"IMAGE_TAG": "runtime"},
            declarations,
        )

        assert result == {"IMAGE_TAG": "runtime"}

    def test_undeclared_variables_are_ignored(self):
        declarations = [VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)]

        result = merge_variables({}, [], {"UNUSED": "value"}, declarations)

        assert result == {"IMAGE_TAG": "latest"}

    def test_builtin_variables_override_all_other_layers(self):
        declarations = [VariableDeclaration(name="GLOBAL_ENV", value="local", source=VariableSource.TEMPLATE_CUSTOM)]

        result = merge_variables(
            {"GLOBAL_ENV": "prod"},
            [VariableDeclaration(name="GLOBAL_ENV", value="stage", source=VariableSource.REPOSITORY_CUSTOM)],
            {"GLOBAL_ENV": "runtime"},
            declarations,
        )

        assert result["GLOBAL_ENV"] == "prod"


class TestValidateVariables:
    """测试变量校验"""

    def test_validate_all_variables_present(self):
        declarations = [
            VariableDeclaration(name="KEY1"),
            VariableDeclaration(name="KEY2"),
        ]

        validate_variables({"KEY1": "value1", "KEY2": "value2"}, declarations)

    def test_validate_empty_string_is_missing(self):
        declarations = [VariableDeclaration(name="KEY1")]

        with pytest.raises(VariableError, match="缺少变量值: KEY1"):
            validate_variables({"KEY1": ""}, declarations)

    def test_validate_builtin_variable_missing(self):
        declarations = [VariableDeclaration(name="GLOBAL_ENV", source=VariableSource.GLOBAL)]

        with pytest.raises(VariableError, match="缺少变量值: GLOBAL_ENV"):
            validate_variables({}, declarations)


class TestRenderTemplate:
    """测试模板渲染"""

    def test_render_simple_template(self):
        result = render_template("Hello {{ name }}!", {"name": "World"})
        assert result == "Hello World!"

    def test_render_multiple_variables(self):
        result = render_template("{{ greeting }} {{ name }}!", {"greeting": "Hello", "name": "World"})
        assert result == "Hello World!"

    def test_render_with_filter(self):
        result = render_template("{{ name | upper }}", {"name": "world"})
        assert result == "WORLD"

    def test_render_with_condition(self):
        assert render_template("{% if enabled %}ON{% else %}OFF{% endif %}", {"enabled": True}) == "ON"
        assert render_template("{% if enabled %}ON{% else %}OFF{% endif %}", {"enabled": False}) == "OFF"

    def test_render_undefined_variable_raises(self):
        with pytest.raises(VariableError, match="变量未定义"):
            render_template("Hello {{ name }}!", {})

    def test_render_syntax_error(self):
        with pytest.raises(VariableError, match="模板语法错误"):
            render_template("Hello {{ name", {"name": "World"})


class TestMaskSecrets:
    """测试 secret 脱敏"""

    def test_mask_no_secrets(self):
        declarations = [
            VariableDeclaration(name="KEY1", secret=False),
            VariableDeclaration(name="KEY2", secret=False),
        ]
        variables = {"KEY1": "value1", "KEY2": "value2"}

        result = mask_secrets(variables, declarations)
        assert len(result) == 2
        assert result[0].name == "KEY1"
        assert result[0].value == "value1"
        assert result[1].name == "KEY2"
        assert result[1].value == "value2"

    def test_mask_secrets_present(self):
        declarations = [
            VariableDeclaration(name="KEY1", secret=False),
            VariableDeclaration(name="PASSWORD", secret=True),
            VariableDeclaration(name="TOKEN", secret=True),
        ]
        variables = {"KEY1": "value1", "PASSWORD": "secret123", "TOKEN": "token456"}

        result = mask_secrets(variables, declarations)
        assert len(result) == 3
        assert result[0].name == "KEY1"
        assert result[0].value == "value1"
        assert result[1].name == "PASSWORD"
        assert result[1].value == "***"
        assert result[2].name == "TOKEN"
        assert result[2].value == "***"

    def test_mask_secrets_not_in_variables(self):
        declarations = [VariableDeclaration(name="PASSWORD", secret=True)]
        variables = {"KEY1": "value1"}

        result = mask_secrets(variables, declarations)
        assert len(result) == 1
        assert result[0].name == "PASSWORD"
        assert result[0].value == "***"

    def test_mask_does_not_modify_original(self):
        declarations = [VariableDeclaration(name="PASSWORD", secret=True)]
        variables = {"PASSWORD": "secret123"}

        result = mask_secrets(variables, declarations)

        assert result[0].value == "***"
        assert variables["PASSWORD"] == "secret123"


class TestValidateVariablesTemplateStage:
    """测试 template_stage 变量的校验行为"""

    def test_template_stage_with_default_value_does_not_require_runtime_value(self):
        """template_stage 变量有 default 值时，即使运行时没有提供值也不报错"""
        declarations = [
            VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE),
        ]
        validate_variables({}, declarations)

    def test_template_stage_without_default_value_requires_runtime_value(self):
        """template_stage 变量没有 default 值时，运行时必须提供"""
        declarations = [
            VariableDeclaration(name="IMAGE_TAG", default=None, source=VariableSource.TEMPLATE_STAGE),
        ]
        with pytest.raises(VariableError, match="缺少变量值: IMAGE_TAG"):
            validate_variables({}, declarations)

    def test_template_stage_runtime_value_overrides_default(self):
        """运行时提供了值时，正常通过校验"""
        declarations = [
            VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE),
        ]
        validate_variables({"working_dir": "frontend"}, declarations)

    def test_template_custom_without_value_still_fails(self):
        """template_custom 变量没有值时仍然报错（不受 template_stage 规则影响）"""
        declarations = [
            VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE),
            VariableDeclaration(name="IMAGE_TAG", default=None, source=VariableSource.TEMPLATE_CUSTOM),
        ]
        with pytest.raises(VariableError, match="缺少变量值: IMAGE_TAG"):
            validate_variables({"working_dir": "."}, declarations)
