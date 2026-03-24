"""
Webhook 集成测试
使用真实的 Github Webhook 请求进行测试
"""

import hashlib
import hmac
import json
from pathlib import Path
from typing import Any
from unittest.mock import patch

import pytest

from pomelo_orbit.infrastructure.persistence.models import ApplicationModel, CredentialModel, GitSourceModel

# 测试用的 webhook secret
TEST_WEBHOOK_SECRET = "test_webhook_secret_for_integration_testing"

# JSON 测试数据文件路径
TEST_DATA_FILE = Path(__file__).parent / "test_webhook_integration.json"


def load_test_data() -> Any:
    """从 JSON 文件加载测试数据"""
    if not TEST_DATA_FILE.exists():
        pytest.skip(f"测试数据文件不存在: {TEST_DATA_FILE}")
    return json.loads(TEST_DATA_FILE.read_text(encoding="utf-8"))


def compute_github_signature(payload: bytes, secret: str) -> str:
    """计算 Github Webhook 签名"""
    mac = hmac.new(secret.encode(), payload, hashlib.sha256)
    return f"sha256={mac.hexdigest()}"


@pytest.fixture
def test_project(db_session):
    """创建测试应用（匹配 JSON 数据中的仓库）"""
    data = load_test_data()
    repo = data["payload"]["repository"]

    app = ApplicationModel(
        name=repo["name"],
        code=f"{repo['name']}-code",
        enabled=True,
    )
    db_session.add(app)
    db_session.flush()

    # 创建凭据（属于应用）
    credential = CredentialModel(
        application_id=app.id,
        name="test_webhook_credential",
        type="webhook_secret",
        value_encrypted=TEST_WEBHOOK_SECRET,
        extra_data=json.dumps({"description": "Test webhook secret for integration testing"}),
    )
    db_session.add(credential)

    git_source = GitSourceModel(
        application_id=app.id,
        repository_url=repo["html_url"],
        deploy_branches="main,master",
        auto_deploy=True,
    )
    db_session.add(git_source)
    db_session.commit()
    db_session.refresh(app)
    return app


class TestGithubWebhookIntegration:
    """Github Webhook 集成测试"""

    @patch("pomelo_orbit.infrastructure.security.SecurityService.decrypt_value")
    def test_webhook_with_real_data(self, mock_decrypt, client, test_project):
        """使用真实 Github Webhook 数据测试"""
        # Mock decrypt_value 返回明文 secret
        mock_decrypt.return_value = TEST_WEBHOOK_SECRET

        data = load_test_data()

        # 复制 headers，避免修改原始数据
        headers = dict(data["headers"])
        payload = data["payload"]
        payload_bytes = json.dumps(payload, separators=(",", ":")).encode()

        # 计算签名（使用测试凭据中的 secret）并覆盖
        headers["X-Hub-Signature-256"] = compute_github_signature(payload_bytes, TEST_WEBHOOK_SECRET)

        response = client.post(
            "/api/hooks/github",
            content=payload_bytes,
            headers=headers,
        )

        # 兼容 header key 大小写
        event_type = None
        for key in headers:
            if key.lower() == "x-github-event":
                event_type = headers[key]
                break
        event_type = event_type or "unknown"

        if event_type == "ping":
            assert response.status_code == 200
            assert response.json()["message"] == "pong"
        else:
            # push, release 等事件
            assert response.status_code in [200, 202]


class TestSignatureVerification:
    """签名验证测试"""

    def test_signature_computation(self):
        """测试签名计算是否正确"""
        payload = b'{"test": "data"}'
        secret = "my-secret"

        # 使用我们的函数计算签名
        signature = compute_github_signature(payload, secret)

        # 手动计算预期签名
        expected_mac = hmac.new(secret.encode(), payload, hashlib.sha256)
        expected_signature = f"sha256={expected_mac.hexdigest()}"

        assert signature == expected_signature

    def test_signature_format(self):
        """测试签名格式"""
        payload = b'{"test": "data"}'
        secret = "my-secret"

        signature = compute_github_signature(payload, secret)

        # 签名应该以 sha256= 开头
        assert signature.startswith("sha256=")
        # sha256 后面应该是 64 位十六进制字符串
        hex_part = signature[7:]
        assert len(hex_part) == 64
        assert all(c in "0123456789abcdef" for c in hex_part)
