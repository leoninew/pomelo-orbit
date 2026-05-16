"""Webhook API 集成测试"""

import hashlib
import hmac

from pomelo_orbit.infrastructure.ci.models import ProjectWebhookModel
from tests.integration.conftest import DEFAULT_CI_PROJECT_ID


class TestWebhookList:
    def test_returns_webhooks_for_project(self, auth_client, test_project, test_template):
        # 创建一个 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "test-webhook",
                "template_id": test_template.id,
                "secret": "test-secret",
            },
        )
        assert create_resp.status_code == 201

        # 列出 webhooks
        resp = auth_client.get(f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        data = resp.json()
        assert isinstance(data, list)
        assert len(data) >= 1
        assert any(w["name"] == "test-webhook" for w in data)


class TestWebhookCreate:
    def test_creates_webhook(self, auth_client, test_project, test_template):
        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "my-webhook",
                "template_id": test_template.id,
                "secret": "my-secret",
            },
        )
        assert resp.status_code == 201
        data = resp.json()
        assert data["name"] == "my-webhook"
        assert data["template_id"] == test_template.id
        assert "secret" not in data  # 密码不应在响应中返回

    def test_validates_required_fields(self, auth_client, test_project):
        resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={"name": "incomplete"},
        )
        assert resp.status_code == 422  # Validation error


class TestWebhookGet:
    def test_webhook_in_list_response(self, auth_client, test_project, test_template):
        """测试 webhook 信息包含在列表响应中（没有单独的 GET 端点）"""
        # 创建 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "get-test",
                "template_id": test_template.id,
                "secret": "secret",
            },
        )
        assert create_resp.status_code == 201
        webhook_data = create_resp.json()

        # 通过列表端点获取 webhook（没有单独的 GET 端点）
        resp = auth_client.get(f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}")
        assert resp.status_code == 200
        webhooks = resp.json()
        # 找到刚创建的 webhook
        webhook = next((w for w in webhooks if w["id"] == webhook_data["id"]), None)
        assert webhook is not None
        assert webhook["name"] == "get-test"


class TestWebhookUpdate:
    def test_updates_webhook(self, auth_client, test_project, test_template):
        # 创建 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "update-test",
                "template_id": test_template.id,
                "secret": "old-secret",
            },
        )
        webhook_id = create_resp.json()["id"]

        # 更新 webhook
        resp = auth_client.put(
            f"/api/ci/repository/{test_project.id}/webhook/{webhook_id}?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "updated-webhook",
                "template_id": test_template.id,
                "secret": "new-secret",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["name"] == "updated-webhook"


class TestWebhookDelete:
    def test_deletes_webhook(self, auth_client, db_session, test_project, test_template):
        # 创建 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "delete-test",
                "template_id": test_template.id,
                "secret": "secret",
            },
        )
        webhook_id = create_resp.json()["id"]

        # 删除 webhook
        resp = auth_client.delete(
            f"/api/ci/repository/{test_project.id}/webhook/{webhook_id}?project_id={DEFAULT_CI_PROJECT_ID}"
        )
        assert resp.status_code == 204

        # 验证已删除
        assert (
            db_session.query(ProjectWebhookModel).filter_by(repository_id=test_project.id, id=webhook_id).first()
            is None
        )


class TestWebhookVerification:
    def test_webhook_trigger_endpoint_exists(self, auth_client, test_project, test_template):
        """测试 webhook 触发端点"""
        secret = "test-secret"

        # 创建 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "verify-test",
                "template_id": test_template.id,
                "secret": secret,
            },
        )
        webhook_id = create_resp.json()["id"]

        # 测试触发端点存在（不测试实际触发，因为这会创建 pipeline run）
        payload = b'{"ref": "refs/heads/main"}'
        signature = hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()

        # 发送请求到全局 webhook 端点
        resp = auth_client.post(
            f"/api/ci/webhook/{webhook_id}",
            content=payload,
            headers={"X-Hub-Signature-256": f"sha256={signature}"},
        )
        # 应该接受请求（可能触发 pipeline，也可能返回测试响应，但不会是分支过滤）
        # 由于没有配置 branch_filter，会被分支过滤，但 HTTP 状态码应该是 200
        assert resp.status_code == 200

    def test_rejects_invalid_signature(self, auth_client, test_project, test_template):
        """测试拒绝无效签名"""
        secret = "test-secret"

        # 创建 webhook
        create_resp = auth_client.post(
            f"/api/ci/repository/{test_project.id}/webhook?project_id={DEFAULT_CI_PROJECT_ID}",
            json={
                "name": "reject-test",
                "template_id": test_template.id,
                "secret": secret,
            },
        )
        webhook_id = create_resp.json()["id"]

        # 发送错误签名的请求
        payload = b'{"ref": "main"}'
        resp = auth_client.post(
            f"/api/ci/webhook/{webhook_id}",
            content=payload,
            headers={"X-Hub-Signature-256": "sha256=invalid"},
        )
        # 签名验证失败应该返回 401 Unauthorized
        assert resp.status_code == 401
