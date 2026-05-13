"""
CSRF Token 生成和验证
"""

from datetime import timedelta

from jose import JWTError, jwt

from pomelo_orbit.infrastructure.time_utils import utc_now


def generate_csrf_token(secret_key: str, ttl_minutes: int = 10) -> str:
    """
    生成 CSRF Token

    Args:
        secret_key: JWT 密钥
        ttl_minutes: 有效期（分钟）

    Returns:
        CSRF token 字符串
    """
    payload = {
        "exp": utc_now() + timedelta(minutes=ttl_minutes),
        "type": "csrf",
    }
    token: str = jwt.encode(payload, secret_key, algorithm="HS256")
    return token


def verify_csrf_token(token: str, secret_key: str) -> bool:
    """
    验证 CSRF Token

    Args:
        token: CSRF token
        secret_key: JWT 密钥

    Returns:
        是否有效
    """
    try:
        payload = jwt.decode(token, secret_key, algorithms=["HS256"])
        is_valid: bool = payload.get("type") == "csrf"
        return is_valid
    except JWTError:
        return False


__all__ = ["generate_csrf_token", "verify_csrf_token"]
