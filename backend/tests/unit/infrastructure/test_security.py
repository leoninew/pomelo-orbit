"""
安全模块单元测试 - 加密解密功能
"""

from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.security import SecurityService


class TestEncryptDecrypt:
    """加密解密测试"""

    def test_encrypt_value(self):
        """测试加密"""
        security_service = SecurityService(get_settings())
        value = "glpat-xxxxxxxxxxxxxxxxxxxx"
        encrypted = security_service.encrypt_value(value)
        assert encrypted != value
        assert isinstance(encrypted, str)

    def test_decrypt_value(self):
        """测试解密"""
        security_service = SecurityService(get_settings())
        value = "glpat-xxxxxxxxxxxxxxxxxxxx"
        encrypted = security_service.encrypt_value(value)
        decrypted = security_service.decrypt_value(encrypted)
        assert decrypted == value
