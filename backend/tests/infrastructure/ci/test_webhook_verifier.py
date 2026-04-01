"""Webhook 签名验证测试"""

import hashlib
import hmac

from pomelo_orbit.infrastructure.ci.webhook_verifier import (
    verify_github_signature,
    verify_gitlab_signature,
)


class TestGitHubSignatureVerification:
    """GitHub 签名验证测试"""

    def test_verify_valid_signature(self):
        """测试验证有效签名"""
        payload = b'{"ref":"refs/heads/main"}'
        secret = "my-secret"

        # 生成正确的签名
        expected = hmac.new(secret.encode("utf-8"), payload, hashlib.sha256).hexdigest()
        signature = f"sha256={expected}"

        assert verify_github_signature(payload, signature, secret) is True

    def test_verify_invalid_signature(self):
        """测试验证无效签名"""
        payload = b'{"ref":"refs/heads/main"}'
        secret = "my-secret"
        signature = "sha256=invalid"

        assert verify_github_signature(payload, signature, secret) is False

    def test_verify_wrong_secret(self):
        """测试使用错误的 secret"""
        payload = b'{"ref":"refs/heads/main"}'
        secret = "my-secret"

        # 使用不同的 secret 生成签名
        wrong_secret = "wrong-secret"
        expected = hmac.new(wrong_secret.encode("utf-8"), payload, hashlib.sha256).hexdigest()
        signature = f"sha256={expected}"

        assert verify_github_signature(payload, signature, secret) is False

    def test_verify_empty_signature(self):
        """测试空签名"""
        payload = b'{"ref":"refs/heads/main"}'
        secret = "my-secret"

        assert verify_github_signature(payload, "", secret) is False

    def test_verify_empty_secret(self):
        """测试空 secret"""
        payload = b'{"ref":"refs/heads/main"}'
        signature = "sha256=something"

        assert verify_github_signature(payload, signature, "") is False

    def test_verify_missing_sha256_prefix(self):
        """测试缺少 sha256= 前缀"""
        payload = b'{"ref":"refs/heads/main"}'
        secret = "my-secret"

        expected = hmac.new(secret.encode("utf-8"), payload, hashlib.sha256).hexdigest()
        # 不加前缀
        signature = expected

        assert verify_github_signature(payload, signature, secret) is False


class TestGitLabSignatureVerification:
    """GitLab 签名验证测试"""

    def test_verify_valid_token(self):
        """测试验证有效 token"""
        secret = "my-secret"
        token = "my-secret"

        assert verify_gitlab_signature(token, secret) is True

    def test_verify_invalid_token(self):
        """测试验证无效 token"""
        secret = "my-secret"
        token = "wrong-token"

        assert verify_gitlab_signature(token, secret) is False

    def test_verify_empty_token(self):
        """测试空 token"""
        secret = "my-secret"

        assert verify_gitlab_signature("", secret) is False

    def test_verify_empty_secret(self):
        """测试空 secret"""
        token = "my-token"

        assert verify_gitlab_signature(token, "") is False
