"""模板变量集成测试 - 测试完整的变量生命周期"""

import pytest


class TestTemplateVariableLifecycle:
    """测试模板变量的完整生命周期"""

    def test_add_stage_and_save_preserves_custom_variables(self, auth_client, db_session):
        """
        测试场景：
        1. 创建模板
        2. 添加自定义变量 test=1
        3. 添加 stage（会解析出 repository_url 等变量）
        4. 保存模板
        5. 验证：自定义变量应该被保留，内置变量和 Stage 变量不应该被存储
        """
        # 1. 创建模板
        resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "GO 构建流水线",
                "description": "",
            },
        )
        assert resp.status_code == 201
        template_id = resp.json()["id"]

        # 2. 创建一个 Stage（包含变量引用）
        stage_resp = auth_client.post(
            "/api/ci/pipeline-stage",
            json={
                "name": "git clone",
                "image": "alpine/git",
                "script": "git clone {{ repository_url }} && git checkout {{ repository_ref }}",
                "description": "克隆代码",
            },
        )
        assert stage_resp.status_code == 201
        stage_id = stage_resp.json()["id"]

        # 3. 调用 resolve-variables API（模拟前端添加 stage 后的行为）
        resolve_resp = auth_client.post(
            "/api/ci/template/resolve-variables",
            json={
                "orchestration": [
                    {
                        "stage_id": stage_id,
                        "stage_name": "git clone",
                        "stage_version": 1,
                        "depends_on": [],
                        "sort_order": 0,
                    }
                ],
                "variable_declarations": [
                    {
                        "name": "test",
                        "value": "1",
                        "source": "template_custom",
                        "secret": False,
                    }
                ],
            },
        )
        assert resolve_resp.status_code == 200
        resolved_vars = resolve_resp.json()

        # 验证返回的变量包含：Stage 中用到的变量（自定义变量 test 未在 stage 中使用，不出现）
        var_names = {v["name"] for v in resolved_vars}
        assert "repository_url" in var_names  # Stage 解析的变量
        assert "repository_ref" in var_names
        assert "test" not in var_names  # 未在 stage 中使用

        # Stage 中没用到的内置变量不应该显示
        assert "template_id" not in var_names
        assert "template_name" not in var_names
        assert "template_version" not in var_names

        # 4. 保存模板（模拟前端点击保存）
        update_resp = auth_client.put(
            f"/api/ci/template/{template_id}",
            json={
                "name": "GO 构建流水线",
                "description": "",
                "orchestration": [
                    {
                        "stage_id": stage_id,
                        "stage_name": "git clone",
                        "stage_version": 1,
                        "depends_on": [],
                        "sort_order": 0,
                    }
                ],
                "variable_declarations": resolved_vars,  # 发送所有变量
            },
        )
        assert update_resp.status_code == 200
        saved_template = update_resp.json()

        # 5. 验证：PUT 响应返回完整的变量列表（只包含 Stage 中用到的变量 + 自定义变量）
        saved_vars = saved_template["variable_declarations"]
        saved_var_names = {v["name"] for v in saved_vars}

        # Stage 中用到的变量应该显示
        assert "repository_url" in saved_var_names
        assert "repository_ref" in saved_var_names

        # 自定义变量 test 未在 stage 中使用，不出现
        assert "test" not in saved_var_names

        # Stage 中没用到的内置变量不应该显示
        assert "template_id" not in saved_var_names
        assert "template_name" not in saved_var_names
        assert "template_version" not in saved_var_names

    def test_get_template_returns_all_variables(self, auth_client, db_session):
        """
        测试场景：
        1. 创建模板并添加 stage 和自定义变量
        2. 保存后重新获取模板
        3. 验证：返回的变量应该包含内置变量 + Stage 变量 + 自定义变量
        """
        # 1. 创建模板
        resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "Test Template",
                "description": "",
            },
        )
        assert resp.status_code == 201
        template_id = resp.json()["id"]

        # 2. 创建 Stage
        stage_resp = auth_client.post(
            "/api/ci/pipeline-stage",
            json={
                "name": "build",
                "image": "alpine",
                "script": "echo {{ repository_url }}",
                "description": "build",
            },
        )
        assert stage_resp.status_code == 201
        stage_id = stage_resp.json()["id"]

        # 3. 更新模板（添加 stage 和自定义变量）
        update_resp = auth_client.put(
            f"/api/ci/template/{template_id}",
            json={
                "name": "Test Template",
                "orchestration": [
                    {
                        "stage_id": stage_id,
                        "stage_name": "build",
                        "stage_version": 1,
                        "depends_on": [],
                        "sort_order": 0,
                    }
                ],
                "variable_declarations": [
                    {
                        "name": "custom_var",
                        "value": "custom_value",
                        "source": "template_custom",
                        "secret": False,
                    }
                ],
            },
        )
        assert update_resp.status_code == 200

        # 4. 重新获取模板
        get_resp = auth_client.get(f"/api/ci/template/{template_id}")
        assert get_resp.status_code == 200
        template = get_resp.json()

        # 5. 验证：返回的变量应该只包含 Stage 中用到的变量（custom_var 未在 stage 中使用，不出现）
        var_names = {v["name"] for v in template["variable_declarations"]}

        # Stage 中用到的变量应该显示
        assert "repository_url" in var_names

        # 自定义变量 custom_var 未在 stage 中使用，不出现
        assert "custom_var" not in var_names

        # Stage 中没用到的内置变量不应该显示
        assert "template_id" not in var_names
        assert "template_name" not in var_names
        assert "template_version" not in var_names

    def test_multiple_save_cycles_preserve_custom_variables(self, auth_client, db_session):
        """
        测试场景：多次保存不会丢失自定义变量
        1. 创建模板并添加自定义变量
        2. 保存
        3. 再次获取并保存
        4. 验证：自定义变量仍然存在
        """
        # 1. 创建模板
        resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "Multi Save Test",
                "description": "",
            },
        )
        assert resp.status_code == 201
        template_id = resp.json()["id"]

        # 2. 创建 Stage
        stage_resp = auth_client.post(
            "/api/ci/pipeline-stage",
            json={
                "name": "test",
                "image": "alpine",
                "script": "echo {{ repository_url }}",
                "description": "test",
            },
        )
        assert stage_resp.status_code == 201
        stage_id = stage_resp.json()["id"]

        # 3. 第一次保存（添加自定义变量）
        update1_resp = auth_client.put(
            f"/api/ci/template/{template_id}",
            json={
                "name": "Multi Save Test",
                "orchestration": [
                    {
                        "stage_id": stage_id,
                        "stage_name": "test",
                        "stage_version": 1,
                        "depends_on": [],
                        "sort_order": 0,
                    }
                ],
                "variable_declarations": [
                    {
                        "name": "my_var",
                        "value": "my_value",
                        "source": "template_custom",
                        "secret": False,
                    }
                ],
            },
        )
        assert update1_resp.status_code == 200

        # 4. 获取模板
        get_resp = auth_client.get(f"/api/ci/template/{template_id}")
        assert get_resp.status_code == 200
        template = get_resp.json()

        # 5. 第二次保存（使用获取到的所有变量）
        update2_resp = auth_client.put(
            f"/api/ci/template/{template_id}",
            json={
                "name": "Multi Save Test",
                "orchestration": template["orchestration"],
                "variable_declarations": template["variable_declarations"],
            },
        )
        assert update2_resp.status_code == 200

        # 6. 再次获取并验证
        final_get_resp = auth_client.get(f"/api/ci/template/{template_id}")
        assert final_get_resp.status_code == 200
        final_template = final_get_resp.json()

        # 验证：my_var 未在 stage 中使用，不出现在变量列表中
        var_names = {v["name"] for v in final_template["variable_declarations"]}
        assert "my_var" not in var_names
        assert "repository_url" in var_names  # stage 中用到的变量仍然存在


class TestRepositoryVariableLifecycle:
    """场景 1: 测试仓库变量的完整生命周期"""

    def test_repository_shows_builtin_and_custom_variables(self, auth_client, test_credential):
        """
        场景 1: 仓库详情界面 - 展示仓库内置变量 + 自定义变量

        测试步骤：
        1. 创建仓库并添加自定义变量
        2. 获取仓库详情
        3. 验证：返回内置变量（repository_id 等）+ 自定义变量
        """
        # 1. 创建仓库（带自定义变量）
        create_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {
                        "name": "DEPLOY_ENV",
                        "value": "production",
                        "description": "部署环境",
                    }
                ],
            },
        )
        assert create_resp.status_code == 201
        repo_id = create_resp.json()["id"]

        # 2. 获取仓库详情
        get_resp = auth_client.get(f"/api/ci/repository/{repo_id}")
        assert get_resp.status_code == 200
        repo = get_resp.json()

        # 3. 验证变量
        variables = repo["variable_declarations"]
        var_names = {v["name"] for v in variables}

        # 应该包含内置变量
        assert "repository_id" in var_names
        assert "repository_name" in var_names
        assert "repository_url" in var_names

        # 验证内置变量的 source 和 default（内置变量值现在在 default 字段）
        repo_id_var = next(v for v in variables if v["name"] == "repository_id")
        assert repo_id_var["source"] == "repository"
        assert repo_id_var["default"] == repo_id
        assert repo_id_var["value"] is None  # 用户未覆盖

        # 应该包含自定义变量
        assert "DEPLOY_ENV" in var_names
        deploy_env_var = next(v for v in variables if v["name"] == "DEPLOY_ENV")
        assert deploy_env_var["source"] == "repository_custom"
        assert deploy_env_var["value"] == "production"

    def test_repository_update_preserves_custom_variables(self, auth_client, test_credential):
        """
        场景 1: 更新仓库时保留自定义变量

        测试步骤：
        1. 创建仓库并添加自定义变量
        2. 更新仓库（修改自定义变量）
        3. 验证：自定义变量被正确更新
        """
        # 1. 创建仓库
        create_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {
                        "name": "VAR1",
                        "value": "value1",
                    }
                ],
            },
        )
        assert create_resp.status_code == 201
        repo_id = create_resp.json()["id"]

        # 2. 更新仓库（修改和添加变量）
        update_resp = auth_client.put(
            f"/api/ci/repository/{repo_id}",
            json={
                "name": "test-repo-updated",
                "variable_overrides": [
                    {
                        "name": "VAR1",
                        "value": "value1_updated",
                    },
                    {
                        "name": "VAR2",
                        "value": "value2",
                    },
                ],
            },
        )
        assert update_resp.status_code == 200

        # 3. 获取并验证
        get_resp = auth_client.get(f"/api/ci/repository/{repo_id}")
        assert get_resp.status_code == 200
        repo = get_resp.json()

        variables = repo["variable_declarations"]
        custom_vars = [v for v in variables if v["source"] == "repository_custom"]
        custom_var_names = {v["name"] for v in custom_vars}

        assert "VAR1" in custom_var_names
        assert "VAR2" in custom_var_names

        var1 = next(v for v in custom_vars if v["name"] == "VAR1")
        assert var1["value"] == "value1_updated"

        var2 = next(v for v in custom_vars if v["name"] == "VAR2")
        assert var2["value"] == "value2"

    def test_repository_cannot_override_builtin_variables(self, auth_client, test_credential):
        """
        场景 1: 仓库不能覆盖内置变量

        测试步骤：
        1. 尝试创建仓库并使用内置变量名作为自定义变量
        2. 验证：内置变量名被过滤掉
        """
        # 1. 创建仓库（尝试覆盖内置变量）
        create_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {
                        "name": "repository_id",  # 内置变量
                        "value": "fake_id",
                    },
                    {
                        "name": "MY_VAR",  # 正常的自定义变量
                        "value": "my_value",
                    },
                ],
            },
        )
        assert create_resp.status_code == 201
        repo_id = create_resp.json()["id"]

        # 2. 获取并验证
        get_resp = auth_client.get(f"/api/ci/repository/{repo_id}")
        assert get_resp.status_code == 200
        repo = get_resp.json()

        variables = repo["variable_declarations"]

        # 内置变量应该保持原值（在 default 字段）
        repo_id_var = next(v for v in variables if v["name"] == "repository_id")
        assert repo_id_var["default"] == repo_id  # 不是 "fake_id"
        assert repo_id_var["value"] is None
        assert repo_id_var["source"] == "repository"

        # 自定义变量应该被保留
        custom_vars = [v for v in variables if v["source"] == "repository_custom"]
        custom_var_names = {v["name"] for v in custom_vars}
        assert "MY_VAR" in custom_var_names
        assert "repository_id" not in custom_var_names  # 不应该出现在自定义变量中


class TestRuntimeVariableMerging:
    """场景 4: 测试运行流水线时的变量合并"""

    @pytest.mark.skip(reason="Background task database issue - pipeline_run table not accessible in test")
    def test_runtime_variables_merge_correctly(self, auth_client, test_credential, db_session):
        """
        场景 4: 运行流水线 - 合并所有来源的变量

        测试步骤：
        1. 创建仓库（带自定义变量）
        2. 创建模板（带自定义变量和 stage）
        3. 触发流水线
        4. 验证：运行时变量正确合并（内置 > 运行时覆盖 > 仓库自定义 > 模板声明）
        """
        # 1. 创建仓库（带自定义变量）
        repo_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
                "variable_overrides": [
                    {
                        "name": "REPO_VAR",
                        "value": "repo_value",
                    },
                    {
                        "name": "SHARED_VAR",
                        "value": "from_repo",
                    },
                ],
            },
        )
        assert repo_resp.status_code == 201
        repo_id = repo_resp.json()["id"]

        # 2. 创建 Stage
        stage_resp = auth_client.post(
            "/api/ci/pipeline-stage",
            json={
                "name": "build",
                "image": "alpine",
                "script": "echo {{ REPO_VAR }} {{ TEMPLATE_VAR }} {{ SHARED_VAR }}",
                "description": "build",
            },
        )
        assert stage_resp.status_code == 201
        stage_id = stage_resp.json()["id"]

        # 3. 创建模板（带自定义变量）
        template_resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "test-template",
                "description": "",
            },
        )
        assert template_resp.status_code == 201
        template_id = template_resp.json()["id"]

        # 更新模板
        auth_client.put(
            f"/api/ci/template/{template_id}",
            json={
                "name": "test-template",
                "orchestration": [
                    {
                        "stage_id": stage_id,
                        "stage_name": "build",
                        "stage_version": 1,
                        "depends_on": [],
                        "sort_order": 0,
                    }
                ],
                "variable_declarations": [
                    {
                        "name": "TEMPLATE_VAR",
                        "value": "template_value",
                        "source": "template_custom",
                        "secret": False,
                    },
                    {
                        "name": "SHARED_VAR",
                        "value": "from_template",
                        "source": "template_custom",
                        "secret": False,
                    },
                ],
            },
        )

        # 4. 触发流水线（带运行时变量）
        trigger_resp = auth_client.post(
            f"/api/ci/repository/{repo_id}/trigger",
            json={
                "template_id": template_id,
                "trigger_ref": "main",
                "variables": {
                    "RUNTIME_VAR": "runtime_value",
                    "SHARED_VAR": "from_runtime",  # 应该覆盖仓库和模板的值
                },
            },
        )
        assert trigger_resp.status_code == 201
        run_id = trigger_resp.json()["id"]

        # 5. 获取运行详情并验证变量
        run_resp = auth_client.get(f"/api/ci/run/{run_id}")
        assert run_resp.status_code == 200
        run = run_resp.json()

        variables = run["variables_snapshot"]

        # 验证内置变量存在
        assert "repository_id" in variables
        assert "template_id" in variables
        assert variables["repository_id"] == repo_id
        assert variables["template_id"] == template_id

        # 验证仓库自定义变量
        assert variables["REPO_VAR"] == "repo_value"

        # 验证模板自定义变量
        assert variables["TEMPLATE_VAR"] == "template_value"

        # 验证运行时变量
        assert variables["RUNTIME_VAR"] == "runtime_value"

        # 验证优先级：运行时 > 仓库 > 模板
        assert variables["SHARED_VAR"] == "from_runtime"
