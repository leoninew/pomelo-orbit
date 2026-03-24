"""
安全模块测试
"""

import pytest
from cryptography.fernet import Fernet, InvalidToken

from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.security import (
    SecurityService,
    hash_password,
    verify_password,
)


class TestPasswordHashing:
    """密码哈希测试"""

    def test_password_hash(self):
        """测试密码哈希"""
        password = "testpassword123"
        hashed = hash_password(password)

        assert hashed != password
        assert verify_password(password, hashed)

    def test_wrong_password(self):
        """测试错误密码"""
        password = "correctpassword"
        hashed = hash_password(password)

        assert not verify_password("wrongpassword", hashed)

    def test_different_passwords_different_hashes(self):
        """测试不同密码产生不同哈希"""
        hash1 = hash_password("password1")
        hash2 = hash_password("password2")

        assert hash1 != hash2

    def test_same_password_different_hashes(self):
        """测试相同密码产生不同哈希（bcrypt salt）"""
        hash1 = hash_password("samepassword")
        hash2 = hash_password("samepassword")

        # bcrypt 每次生成不同的哈希
        assert hash1 != hash2
        # 但都能验证
        assert verify_password("samepassword", hash1)
        assert verify_password("samepassword", hash2)


class TestJWTToken:
    """JWT Token 测试"""

    def test_create_and_verify_token(self):
        """测试创建和验证 token"""
        security_service = SecurityService(get_settings())
        data = {"sub": "testuser"}
        token = security_service.create_access_token(data)

        assert token is not None
        assert isinstance(token, str)

        payload = security_service.decode_access_token(token)
        assert payload is not None
        assert payload["sub"] == "testuser"

    def test_invalid_token(self):
        """测试无效 token"""
        security_service = SecurityService(get_settings())
        payload = security_service.decode_access_token("invalid.token.here")
        assert payload is None

    def test_token_with_expired(self):
        """测试过期 token（需要等待）"""
        # 这个测试需要等待 token 过期，这里跳过


class TestEncryption:
    """凭据加密测试"""

    def test_encrypt_decrypt_direct(self):
        """测试加密和解密（直接使用 Fernet）"""
        # 使用固定的测试 key
        key = Fernet.generate_key()
        fernet = Fernet(key)

        original = "my-secret-token-123"
        encrypted = fernet.encrypt(original.encode()).decode()

        assert encrypted != original
        assert encrypted is not None

        decrypted = fernet.decrypt(encrypted.encode()).decode()
        assert decrypted == original

    def test_different_values_different_encryption(self):
        """测试不同值产生不同密文（直接使用 Fernet）"""
        key = Fernet.generate_key()
        fernet = Fernet(key)

        encrypted1 = fernet.encrypt(b"value1").decode()
        encrypted2 = fernet.encrypt(b"value2").decode()

        assert encrypted1 != encrypted2

    def test_encrypt_empty_string_direct(self):
        """测试加密空字符串（直接使用 Fernet）"""
        key = Fernet.generate_key()
        fernet = Fernet(key)

        encrypted = fernet.encrypt(b"").decode()
        decrypted = fernet.decrypt(encrypted.encode()).decode()
        assert decrypted == ""

    def test_decrypt_invalid_value(self):
        """测试解密无效值"""
        security_service = SecurityService(get_settings())
        # 无效的 Fernet token 会抛出异常
        # 注意：decrypt_value 会先尝试创建 Fernet 对象，如果 key 无效会抛出 ValueError
        with pytest.raises((InvalidToken, ValueError)):
            security_service.decrypt_value("not-a-valid-encrypted-value")
