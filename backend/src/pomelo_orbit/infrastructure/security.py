"""
安全模块：JWT 认证 + 凭据加密 + Webhook 签名验证
"""

import hashlib
import hmac
import logging
from datetime import timedelta
from typing import Any, cast

import bcrypt
from cryptography.fernet import Fernet
from dynaconf import Dynaconf
from jose import JWTError, jwt

from pomelo_orbit.infrastructure.time_utils import add_minutes, utc_now

logger = logging.getLogger(__name__)


def hash_password(password: str) -> str:
    """生成密码哈希"""
    salt = bcrypt.gensalt()
    return bcrypt.hashpw(password.encode(), salt).decode()


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """验证密码"""
    return bcrypt.checkpw(plain_password.encode(), hashed_password.encode())


def validate_security_config(settings: Dynaconf) -> None:
    """验证安全配置"""
    secret_key = settings.jwt.secret_key
    assert secret_key, "jwt.secret_key 未配置, 请设置 POMELO_ORBIT_JWT__SECRET_KEY 环境变量"

    # 验证 Fernet 格式
    try:
        Fernet(secret_key.encode() if isinstance(secret_key, str) else secret_key)
    except Exception as e:
        raise AssertionError(
            "jwt.secret_key 不是有效的 Fernet 格式。请参考 .env.example 文件中的说明生成有效的密钥。"
        ) from e


class SecurityService:
    """安全服务 - 处理 JWT、加密、签名验证"""

    def __init__(self, settings: Dynaconf):
        self.settings = settings
        validate_security_config(settings)

    def create_access_token(self, data: dict[str, Any], expires_delta: timedelta | None = None) -> str:
        """创建 JWT 访问令牌"""
        secret_key = self.settings.jwt.secret_key

        to_encode = data.copy()
        if expires_delta:
            expire = add_minutes(utc_now(), int(expires_delta.total_seconds() / 60))
        else:
            expire = add_minutes(utc_now(), self.settings.jwt.expire_minutes)
        to_encode.update({"exp": expire})
        encoded_jwt = jwt.encode(
            to_encode,
            secret_key,
            algorithm=self.settings.jwt.algorithm,
        )
        return cast("str", encoded_jwt)

    def decode_access_token(self, token: str) -> dict[str, Any] | None:
        """解码 JWT 访问令牌"""
        secret_key = self.settings.jwt.secret_key

        try:
            payload = jwt.decode(
                token,
                secret_key,
                algorithms=[self.settings.jwt.algorithm],
            )
            return cast("dict[str, Any]", payload)
        except JWTError:
            return None

    def encrypt_value(self, plain_value: str) -> str:
        """加密敏感值"""
        key = self.settings.jwt.secret_key
        fernet = Fernet(key.encode() if isinstance(key, str) else key)
        return fernet.encrypt(plain_value.encode()).decode()

    def decrypt_value(self, encrypted_value: str) -> str:
        """解密敏感值"""
        key = self.settings.jwt.secret_key
        fernet = Fernet(key.encode() if isinstance(key, str) else key)
        return fernet.decrypt(encrypted_value.encode()).decode()

    @staticmethod
    def verify_github_signature(payload: bytes, signature: str, secret: str) -> bool:
        """
        验证 Github Webhook 签名

        Args:
            payload: 原始请求体
            signature: X-Hub-Signature-256 header 值 (格式: sha256=xxx)
            secret: Webhook secret（从项目凭据中获取）

        Returns:
            签名是否有效
        """
        if not secret:
            return False

        if not signature.startswith("sha256="):
            return False

        expected_signature = signature[7:]  # 移除 "sha256=" 前缀
        mac = hmac.new(secret.encode(), payload, hashlib.sha256)
        computed_signature = mac.hexdigest()

        return hmac.compare_digest(computed_signature, expected_signature)


# ============================================================================
# 向后兼容的函数接口（使用全局 get_settings）
# ============================================================================


def verify_github_signature(payload: bytes, signature: str, secret: str) -> bool:
    """验证 Github Webhook 签名（向后兼容）"""
    return SecurityService.verify_github_signature(payload, signature, secret)


__all__ = [
    "SecurityService",
    "hash_password",
    "validate_security_config",
    "verify_github_signature",
    "verify_password",
]
