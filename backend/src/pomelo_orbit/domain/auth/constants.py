"""OAuth Provider 常量定义"""


# OAuth 提供商
class OAuthProvider:
    """OAuth 提供商常量"""

    GOOGLE = "google"
    GITHUB = "github"  # 预留

    @classmethod
    def all(cls) -> list[str]:
        """返回所有支持的提供商"""
        return [cls.GOOGLE, cls.GITHUB]


# 认证来源
class AuthSource:
    """认证来源常量"""

    PASSWORD = "password"
    OAUTH = "oauth"
