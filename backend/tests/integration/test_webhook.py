"""
Webhook 相关测试
"""

from pomelo_orbit.application.webhook_parser import parse_github_payload


class TestPayloadParsing:
    """Payload 解析测试"""

    def test_parse_github_push(self):
        """测试 Github push 事件解析"""
        payload = {
            "ref": "refs/heads/main",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "testuser"},
        }
        result = parse_github_payload("push", payload)

        assert result is not None
        assert result.event_type == "push"
        assert result.repository_name == "owner/repo"
        assert result.branch == "main"
        assert result.sender == "testuser"

    def test_parse_github_ping(self):
        """测试 Github ping 事件解析"""
        payload = {"zen": "Keep it simple", "hook_id": 123}
        result = parse_github_payload("ping", payload)

        assert result is not None
        assert result.event_type == "ping"

    def test_parse_github_release(self):
        """测试 Github release 事件解析"""
        payload = {
            "action": "published",
            "release": {"tag_name": "v1.0.0", "target_commitish": "main"},
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "testuser"},
        }
        result = parse_github_payload("release", payload)

        assert result is not None
        assert result.event_type == "release"
        assert result.branch == "main"  # target_commitish

    def test_parse_github_unsupported(self):
        """测试不支持的事件类型"""
        result = parse_github_payload("unsupported", {})
        assert result is None
