"""
安全模块单元测试 - 加密解密功能
"""

import re

import pytest
from dynaconf import Dynaconf

from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.security import SecurityService, validate_security_config


class TestValidateSecurityConfig:
    """安全配置校验测试"""

    def test_validates_missing_secret_key(self):
        """缺失密钥时报错"""
        settings = Dynaconf(settings_files=[], environments=False)
        settings.set("jwt", {"secret_key": ""})

        with pytest.raises(AssertionError, match=re.escape("jwt.secret_key 未配置")):
            validate_security_config(settings)

    def test_validates_fernet_secret_key_format(self):
        """非法 Fernet 密钥格式时报错"""
        settings = Dynaconf(settings_files=[], environments=False)
        settings.set("jwt", {"secret_key": "00000000000000000000000000000000000000000000"})

        with pytest.raises(AssertionError, match=re.escape("jwt.secret_key 不是有效的 Fernet 格式")):
            validate_security_config(settings)


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
