"""Webhook 签名验证"""

import hashlib
import hmac
import logging

logger = logging.getLogger(__name__)


def verify_github_signature(payload: bytes, signature: str, secret: str) -> bool:
    """
    验证 GitHub webhook 签名

    Args:
        payload: 原始请求体
        signature: X-Hub-Signature-256 header 值
        secret: webhook secret

    Returns:
        签名是否有效
    """
    if not signature or not secret:
        logger.warning("GitHub signature verification failed: signature or secret is empty")
        return False

    expected = hmac.new(
        secret.encode("utf-8"), payload, hashlib.sha256
    ).hexdigest()

    expected_signature = f"sha256={expected}"
    is_valid = hmac.compare_digest(expected_signature, signature)

    if not is_valid:
        logger.warning("GitHub signature verification failed: signature mismatch")

    return is_valid


def verify_gitlab_signature(token: str, secret: str) -> bool:
    """
    验证 GitLab webhook 签名

    Args:
        token: X-Gitlab-Token header 值
        secret: webhook secret

    Returns:
        签名是否有效
    """
    if not token or not secret:
        logger.warning("GitLab signature verification failed: token or secret is empty")
        return False

    is_valid = hmac.compare_digest(token, secret)

    if not is_valid:
        logger.warning("GitLab signature verification failed: token mismatch")

    return is_valid
