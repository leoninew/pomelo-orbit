"""测试快照变量存储"""

import pytest


class TestSnapshotVariables:
    """测试快照中的变量存储"""

    @pytest.mark.skip(reason="Background task uses independent session, incompatible with test teardown")
    def test_snapshot_stores_complete_variable_list(self, auth_client, test_credential):
        """
        测试场景：快照应该存储完整的变量列表（内置 + stage + 自定义）

        步骤：
        1. 创建模板并添加 stage 和自定义变量
        2. 触发流水线（会创建快照）
        3. 验证快照中存储了完整的变量列表
        """
        # 1. 创建模板
        template_resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "Test Template",
                "description": "",
            },
        )
        assert template_resp.status_code == 201
        template_id = template_resp.json()["id"]

        # 2. 创建 Stage
        stage_resp = auth_client.post(
            "/api/ci/build-stage",
            json={
                "name": "build",
                "image": "alpine",
                "script": "echo {{ repository_url }} {{ template_id }}",
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

        # 4. 创建仓库
        repo_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
            },
        )
        assert repo_resp.status_code == 201
        repo_id = repo_resp.json()["id"]

        # 5. 触发流水线（会创建快照）
        trigger_resp = auth_client.post(
            f"/api/ci/repository/{repo_id}/trigger",
            json={
                "template_id": template_id,
                "trigger_ref": "main",
            },
        )
        assert trigger_resp.status_code == 201
        run = trigger_resp.json()
        snapshot_id = run["snapshot_id"]

        # 6. 获取快照
        snapshot_resp = auth_client.get(f"/api/ci/snapshot/{snapshot_id}")
        assert snapshot_resp.status_code == 200
        snapshot = snapshot_resp.json()

        # 7. 验证快照中的变量列表
        var_names = {v["name"] for v in snapshot["variables_snapshot"]}

        # 应该包含 stage 中用到的内置变量
        assert "repository_url" in var_names  # stage 脚本中用到
        assert "template_id" in var_names  # stage 脚本中用到

        # 不应该包含未在 stage 中使用的自定义变量
        assert "custom_var" not in var_names  # 虽然定义了，但 stage 中没用到

        # 不应该包含 stage 中没用到的内置变量
        assert "template_name" not in var_names
        assert "template_version" not in var_names

    @pytest.mark.skip(reason="Background task uses independent session, incompatible with test teardown")
    def test_snapshot_reuse_when_template_unchanged(self, auth_client, test_credential):
        """
        测试场景：模板未变更时应该复用快照

        步骤：
        1. 创建模板和仓库
        2. 第一次触发流水线（创建快照）
        3. 第二次触发流水线（应该复用快照）
        4. 验证两次运行使用同一个快照
        """
        # 1. 创建模板
        template_resp = auth_client.post(
            "/api/ci/template",
            json={
                "name": "Test Template",
                "description": "",
            },
        )
        assert template_resp.status_code == 201
        template_id = template_resp.json()["id"]

        # 2. 创建仓库
        repo_resp = auth_client.post(
            "/api/ci/repository",
            json={
                "name": "test-repo",
                "code": "test-repo",
                "repository_url": "https://github.com/test/repo.git",
                "git_credential_id": test_credential.id,
            },
        )
        assert repo_resp.status_code == 201
        repo_id = repo_resp.json()["id"]

        # 3. 第一次触发
        trigger1_resp = auth_client.post(
            f"/api/ci/repository/{repo_id}/trigger",
            json={
                "template_id": template_id,
                "trigger_ref": "main",
            },
        )
        assert trigger1_resp.status_code == 201
        run1 = trigger1_resp.json()
        snapshot_id_1 = run1["snapshot_id"]

        # 4. 第二次触发
        trigger2_resp = auth_client.post(
            f"/api/ci/repository/{repo_id}/trigger",
            json={
                "template_id": template_id,
                "trigger_ref": "main",
            },
        )
        assert trigger2_resp.status_code == 201
        run2 = trigger2_resp.json()
        snapshot_id_2 = run2["snapshot_id"]

        # 5. 验证使用同一个快照
        assert snapshot_id_1 == snapshot_id_2
