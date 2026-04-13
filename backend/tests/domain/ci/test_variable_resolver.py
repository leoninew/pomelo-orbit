"""VariableResolver 单元测试

覆盖场景：
1. get_repository_variables - 仓库变量列表（内置 + 自定义，editable 规范）
2. resolve_template_variables - 模板变量解析（内置/stage/custom，editable 规范）
3. build_runtime_variables - 运行时变量合并优先级
4. sanitize_variable_overrides - 持久化过滤（内置过滤、stage 升级）
5. get_builtin_variable_names - 内置变量名集合
"""

from dataclasses import dataclass, field

from pomelo_orbit.domain.ci.entities import BuildStage, Repository
from pomelo_orbit.domain.ci.value_objects import VariableDeclaration, VariableSource
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver

#  测试用 fixtures


def make_repository(
    *,
    id: str = "repo-1",
    name: str = "my-repo",
    url: str = "https://github.com/org/repo.git",
    default_branch: str = "main",
    variable_overrides: list[VariableDeclaration] | None = None,
) -> Repository:
    return Repository(
        id=id,
        name=name,
        code="my-repo",
        repository_url=url,
        default_branch=default_branch,
        variable_overrides=variable_overrides or [],
    )


@dataclass
class FakeTemplate:
    id: str = "tmpl-1"
    name: str = "my-template"
    version: int = 1
    variable_declarations: list = field(default_factory=list)


#  场景 1: get_repository_variables


class TestGetRepositoryVariables:
    def setup_method(self):
        self.resolver = VariableResolver()
        self.repo = make_repository()

    def test_returns_all_builtin_vars(self):
        result = self.resolver.get_repository_variables(self.repo)
        names = {v.name for v in result}
        assert {"repository_id", "repository_name", "repository_url", "repository_ref"} <= names

    def test_builtin_vars_have_correct_source(self):
        result = self.resolver.get_repository_variables(self.repo)
        specs = self.resolver._get_repository_builtin_specs()
        builtins = [v for v in result if v.source == VariableSource.REPOSITORY]
        assert len(builtins) == len(specs)

    def test_builtin_default_reflects_repo_values(self):
        result = self.resolver.get_repository_variables(self.repo)
        by_name = {v.name: v for v in result}
        assert by_name["repository_id"].default == "repo-1"
        assert by_name["repository_name"].default == "my-repo"
        assert by_name["repository_url"].default == "https://github.com/org/repo.git"
        assert by_name["repository_ref"].default == "main"  # default_branch

    def test_repository_ref_is_editable(self):
        result = self.resolver.get_repository_variables(self.repo)
        by_name = {v.name: v for v in result}
        assert by_name["repository_ref"].editable is True

    def test_other_builtins_are_not_editable(self):
        result = self.resolver.get_repository_variables(self.repo)
        by_name = {v.name: v for v in result}
        assert by_name["repository_id"].editable is False
        assert by_name["repository_name"].editable is False
        assert by_name["repository_url"].editable is False

    def test_custom_vars_appended_with_correct_source(self):
        repo = make_repository(
            variable_overrides=[
                VariableDeclaration(name="DEPLOY_ENV", value="prod"),
            ]
        )
        result = self.resolver.get_repository_variables(repo)
        custom = [v for v in result if v.source == VariableSource.REPOSITORY_CUSTOM]
        assert len(custom) == 1
        assert custom[0].name == "DEPLOY_ENV"
        assert custom[0].value == "prod"
        assert custom[0].editable is True

    def test_no_custom_vars_when_overrides_empty(self):
        result = self.resolver.get_repository_variables(self.repo)
        custom = [v for v in result if v.source == VariableSource.REPOSITORY_CUSTOM]
        assert custom == []


#  场景 2: resolve_template_variables ─


class TestResolveTemplateVariables:
    def setup_method(self):
        self.resolver = VariableResolver()

    def test_no_stages_returns_all_builtins_and_custom(self):
        custom = [VariableDeclaration(name="MY_VAR", value="v", source=VariableSource.TEMPLATE_CUSTOM)]
        result = self.resolver.resolve_template_variables([], custom)
        names = {v.name for v in result}
        # 所有内置变量都应出现
        assert "repository_id" in names
        assert "template_id" in names
        # 自定义变量也出现
        assert "MY_VAR" in names

    def test_builtin_vars_are_not_editable(self):
        # editable 由 spec 决定，直接从 spec 推导期望值，不硬编码例外列表
        all_specs = {
            **self.resolver._get_repository_builtin_specs(),
            **self.resolver._get_template_builtin_specs(),
        }
        result_map = {v.name: v for v in self.resolver.resolve_template_variables([], [])}
        for name, spec in all_specs.items():
            assert result_map[name].editable == spec.editable, (
                f"{name}: expected editable={spec.editable}, got {result_map[name].editable}"
            )

    def test_repository_ref_is_editable_in_template_context(self):
        result = self.resolver.resolve_template_variables([], [])
        by_name = {v.name: v for v in result}
        assert by_name["repository_ref"].editable is True

    def test_stage_extracted_vars_are_editable(self):
        stage = _make_stage(script="cd {{ working_dir | default('.') }}")
        result = self.resolver.resolve_template_variables([stage], [])
        by_name = {v.name: v for v in result}
        assert "working_dir" in by_name
        assert by_name["working_dir"].source == VariableSource.TEMPLATE_STAGE
        assert by_name["working_dir"].editable is True

    def test_stage_extracted_default_stored_in_default_field(self):
        stage = _make_stage(script="cd {{ working_dir | default('.') }}")
        result = self.resolver.resolve_template_variables([stage], [])
        by_name = {v.name: v for v in result}
        assert by_name["working_dir"].default == "."
        assert by_name["working_dir"].value is None

    def test_custom_var_with_same_name_as_stage_var_preserves_metadata(self):
        """用户已声明的 template_custom 变量与 stage 提取的同名变量合并时保留用户元数据"""
        custom = [
            VariableDeclaration(
                name="working_dir",
                value="frontend",
                description="工作目录",
                source=VariableSource.TEMPLATE_CUSTOM,
            )
        ]
        stage = _make_stage(script="cd {{ working_dir | default('.') }}")
        result = self.resolver.resolve_template_variables([stage], custom)
        by_name = {v.name: v for v in result}
        wd = by_name["working_dir"]
        assert wd.source == VariableSource.TEMPLATE_CUSTOM
        assert wd.value == "frontend"
        assert wd.description == "工作目录"
        assert wd.default == "."  # stage default 补充进来

    def test_custom_vars_are_editable(self):
        custom = [VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)]
        result = self.resolver.resolve_template_variables([], custom)
        by_name = {v.name: v for v in result}
        assert by_name["IMAGE_TAG"].editable is True

    def test_only_referenced_vars_shown_when_stages_present(self):
        """有 stage 时，只显示 stage 中实际引用的变量"""
        custom = [
            VariableDeclaration(name="UNUSED_VAR", value="x", source=VariableSource.TEMPLATE_CUSTOM),
            VariableDeclaration(name="USED_VAR", value="y", source=VariableSource.TEMPLATE_CUSTOM),
        ]
        stage = _make_stage(script="echo {{ USED_VAR }}")
        result = self.resolver.resolve_template_variables([stage], custom)
        names = {v.name for v in result}
        assert "USED_VAR" in names
        assert "UNUSED_VAR" not in names


#  场景 3: build_runtime_variables ─


class TestBuildRuntimeVariables:
    def setup_method(self):
        self.resolver = VariableResolver(global_variables={"CI_ENV": "production"})
        self.repo = make_repository(id="r1", name="repo", url="https://git.example.com/repo.git")
        self.template = FakeTemplate()

    def test_builtin_vars_always_present(self):
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main")
        assert result["repository_id"] == "r1"
        assert result["repository_name"] == "repo"
        assert result["repository_url"] == "https://git.example.com/repo.git"
        assert result["repository_ref"] == "main"
        assert result["template_id"] == "tmpl-1"
        assert result["template_name"] == "my-template"
        assert result["template_version"] == 1
        assert result["CI_ENV"] == "production"

    def test_trigger_ref_sets_repository_ref(self):
        result = self.resolver.build_runtime_variables(self.repo, self.template, "feature/x")
        assert result["repository_ref"] == "feature/x"

    def test_runtime_overrides_applied(self):
        self.template.variable_declarations = [
            VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)
        ]
        result = self.resolver.build_runtime_variables(
            self.repo, self.template, "main", runtime_overrides={"IMAGE_TAG": "v2.0"}
        )
        assert result["IMAGE_TAG"] == "v2.0"

    def test_runtime_overrides_cannot_override_builtins(self):
        result = self.resolver.build_runtime_variables(
            self.repo, self.template, "main", runtime_overrides={"repository_id": "fake", "CI_ENV": "dev"}
        )
        assert result["repository_id"] == "r1"
        assert result["CI_ENV"] == "production"

    def test_runtime_overrides_beat_repo_custom(self):
        repo = make_repository(variable_overrides=[VariableDeclaration(name="DEPLOY_ENV", value="staging")])
        self.template.variable_declarations = [
            VariableDeclaration(name="DEPLOY_ENV", value="dev", source=VariableSource.TEMPLATE_CUSTOM)
        ]
        result = self.resolver.build_runtime_variables(
            repo, self.template, "main", runtime_overrides={"DEPLOY_ENV": "prod"}
        )
        assert result["DEPLOY_ENV"] == "prod"

    def test_repo_custom_beats_template_custom(self):
        repo = make_repository(variable_overrides=[VariableDeclaration(name="IMAGE_TAG", value="repo-value")])
        self.template.variable_declarations = [
            VariableDeclaration(name="IMAGE_TAG", value="tmpl-value", source=VariableSource.TEMPLATE_CUSTOM)
        ]
        result = self.resolver.build_runtime_variables(repo, self.template, "main")
        assert result["IMAGE_TAG"] == "repo-value"

    def test_template_custom_value_used_when_no_override(self):
        self.template.variable_declarations = [
            VariableDeclaration(name="IMAGE_TAG", value="latest", source=VariableSource.TEMPLATE_CUSTOM)
        ]
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main")
        assert result["IMAGE_TAG"] == "latest"

    def test_template_custom_default_used_when_value_is_none(self):
        """template_custom 变量 value=None 时用 default"""
        self.template.variable_declarations = [
            VariableDeclaration(
                name="IMAGE_TAG", value=None, default="from-default", source=VariableSource.TEMPLATE_CUSTOM
            )
        ]
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main")
        assert result["IMAGE_TAG"] == "from-default"

    def test_stage_default_used_as_lowest_priority(self):
        stage_decls = [VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE)]
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main", stage_declarations=stage_decls)
        assert result["working_dir"] == "."

    def test_stage_value_overrides_stage_default(self):
        """template_stage 变量 value 优先于 default"""
        stage_decls = [
            VariableDeclaration(name="working_dir", value="frontend", default=".", source=VariableSource.TEMPLATE_STAGE)
        ]
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main", stage_declarations=stage_decls)
        assert result["working_dir"] == "frontend"

    def test_runtime_override_beats_stage_default(self):
        stage_decls = [VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE)]
        result = self.resolver.build_runtime_variables(
            self.repo,
            self.template,
            "main",
            runtime_overrides={"working_dir": "backend"},
            stage_declarations=stage_decls,
        )
        assert result["working_dir"] == "backend"

    def test_repo_custom_beats_stage_default(self):
        repo = make_repository(variable_overrides=[VariableDeclaration(name="working_dir", value="infra")])
        stage_decls = [VariableDeclaration(name="working_dir", default=".", source=VariableSource.TEMPLATE_STAGE)]
        result = self.resolver.build_runtime_variables(repo, self.template, "main", stage_declarations=stage_decls)
        assert result["working_dir"] == "infra"

    def test_full_priority_chain(self):
        """完整优先级链：内置 > 运行时 > 仓库自定义 > 模板自定义 > stage default"""
        repo = make_repository(variable_overrides=[VariableDeclaration(name="VAR", value="repo")])
        self.template.variable_declarations = [
            VariableDeclaration(name="VAR", value="tmpl", source=VariableSource.TEMPLATE_CUSTOM)
        ]
        stage_decls = [VariableDeclaration(name="VAR", default="stage", source=VariableSource.TEMPLATE_STAGE)]

        # 无运行时覆盖：仓库自定义胜出
        r1 = self.resolver.build_runtime_variables(repo, self.template, "main", stage_declarations=stage_decls)
        assert r1["VAR"] == "repo"

        # 有运行时覆盖：运行时胜出
        r2 = self.resolver.build_runtime_variables(
            repo,
            self.template,
            "main",
            runtime_overrides={"VAR": "runtime"},
            stage_declarations=stage_decls,
        )
        assert r2["VAR"] == "runtime"

    def test_no_stage_declarations_does_not_fail(self):
        result = self.resolver.build_runtime_variables(self.repo, self.template, "main")
        assert "repository_id" in result


#  场景 4: sanitize_variable_overrides


class TestSanitizeVariableOverrides:
    def setup_method(self):
        self.resolver = VariableResolver()

    def test_empty_input_returns_empty(self):
        assert self.resolver.sanitize_variable_overrides(None) == []
        assert self.resolver.sanitize_variable_overrides([]) == []

    def test_template_custom_is_kept(self):
        var = VariableDeclaration(name="MY_VAR", value="v", source=VariableSource.TEMPLATE_CUSTOM)
        result = self.resolver.sanitize_variable_overrides([var])
        assert len(result) == 1
        assert result[0].name == "MY_VAR"

    def test_repository_custom_is_kept(self):
        var = VariableDeclaration(name="MY_VAR", value="v", source=VariableSource.REPOSITORY_CUSTOM)
        result = self.resolver.sanitize_variable_overrides([var])
        assert len(result) == 1

    def test_builtin_names_are_filtered_out(self):
        vars = [
            VariableDeclaration(name="repository_id", value="fake", source=VariableSource.TEMPLATE_CUSTOM),
            VariableDeclaration(name="template_id", value="fake", source=VariableSource.TEMPLATE_CUSTOM),
            VariableDeclaration(name="MY_VAR", value="ok", source=VariableSource.TEMPLATE_CUSTOM),
        ]
        result = self.resolver.sanitize_variable_overrides(vars)
        names = {v.name for v in result}
        assert "repository_id" not in names
        assert "template_id" not in names
        assert "MY_VAR" in names

    def test_template_stage_with_value_is_upgraded_to_template_custom(self):
        var = VariableDeclaration(
            name="working_dir", value="frontend", default=".", source=VariableSource.TEMPLATE_STAGE
        )
        result = self.resolver.sanitize_variable_overrides([var])
        assert len(result) == 1
        assert result[0].source == VariableSource.TEMPLATE_CUSTOM
        assert result[0].value == "frontend"
        assert result[0].default == "."  # default 字段保留

    def test_template_stage_without_value_is_dropped(self):
        """template_stage 变量没有 value（只有 default）时不持久化"""
        var = VariableDeclaration(name="working_dir", value=None, default=".", source=VariableSource.TEMPLATE_STAGE)
        result = self.resolver.sanitize_variable_overrides([var])
        assert result == []

    def test_global_and_template_source_are_filtered(self):
        vars = [
            VariableDeclaration(name="X", value="v", source=VariableSource.GLOBAL),
            VariableDeclaration(name="Y", value="v", source=VariableSource.TEMPLATE),
            VariableDeclaration(name="Z", value="v", source=VariableSource.REPOSITORY),
        ]
        result = self.resolver.sanitize_variable_overrides(vars)
        assert result == []

    def test_global_variable_with_custom_source_still_filtered_by_name(self):
        """即使 source 是 custom，如果名字是内置变量名也要过滤"""
        resolver = VariableResolver(global_variables={"CI_ENV": "prod"})
        var = VariableDeclaration(name="CI_ENV", value="dev", source=VariableSource.TEMPLATE_CUSTOM)
        result = resolver.sanitize_variable_overrides([var])
        assert result == []


#  场景 5: get_builtin_variable_names ─


class TestGetBuiltinVariableNames:
    def test_includes_repository_and_template_builtins(self):
        resolver = VariableResolver()
        names = resolver.get_builtin_variable_names()
        assert "repository_id" in names
        assert "repository_ref" in names
        assert "template_id" in names
        assert "template_version" in names

    def test_includes_global_variables(self):
        resolver = VariableResolver(global_variables={"CI_ENV": "prod", "REGISTRY": "docker.io"})
        names = resolver.get_builtin_variable_names()
        assert "CI_ENV" in names
        assert "REGISTRY" in names

    def test_no_global_variables_by_default(self):
        resolver = VariableResolver()
        names = resolver.get_builtin_variable_names()
        # 不包含全局变量（全局变量来自 global_variables 参数）
        assert "CI_ENV" not in names
        assert "REGISTRY" not in names
        # 包含仓库和模板内置变量
        assert "repository_id" in names
        assert "template_id" in names


#  辅助函数


def _make_stage(script: str) -> BuildStage:
    """创建测试用 BuildStage"""
    return BuildStage.create(name="test-stage", image="alpine", script=script)
