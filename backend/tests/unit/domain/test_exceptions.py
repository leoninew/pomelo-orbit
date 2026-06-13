"""
领域异常测试
"""

import pytest

from pomelo_orbit.domain.exceptions import AuthenticationError, AuthorizationError, BusinessError


class TestBusinessError:
    """BusinessError 基类测试"""

    def test_create_with_default_status_code(self):
        """测试创建业务异常时使用默认状态码"""
        error = BusinessError("测试错误")
        assert error.message == "测试错误"
        assert error.status_code == 400
        assert str(error) == "测试错误"

    def test_create_with_custom_status_code(self):
        """测试创建业务异常时使用自定义状态码"""
        error = BusinessError("测试错误", status_code=500)
        assert error.message == "测试错误"
        assert error.status_code == 500

    def test_is_exception_instance(self):
        """测试业务异常是 Exception 的实例"""
        error = BusinessError("测试错误")
        assert isinstance(error, Exception)

    def test_can_be_raised(self):
        """测试业务异常可以被抛出和捕获"""
        with pytest.raises(BusinessError) as exc_info:
            raise BusinessError("测试错误")
        assert exc_info.value.message == "测试错误"


class TestAuthenticationError:
    """AuthenticationError 测试"""

    def test_create_with_default_message(self):
        """测试创建认证异常时使用默认消息"""
        error = AuthenticationError()
        assert error.message == "用户名或密码错误"
        assert error.status_code == 400

    def test_create_with_custom_message(self):
        """测试创建认证异常时使用自定义消息"""
        error = AuthenticationError("无效的凭据")
        assert error.message == "无效的凭据"
        assert error.status_code == 400

    def test_is_business_error_subclass(self):
        """测试认证异常是 BusinessError 的子类"""
        error = AuthenticationError()
        assert isinstance(error, BusinessError)
        assert isinstance(error, Exception)

    def test_can_be_caught_as_business_error(self):
        """测试认证异常可以作为 BusinessError 被捕获"""
        with pytest.raises(BusinessError) as exc_info:
            raise AuthenticationError("测试")
        assert exc_info.value.message == "测试"


class TestAuthorizationError:
    """AuthorizationError 测试"""

    def test_create_with_default_message(self):
        """测试创建授权异常时使用默认消息"""
        error = AuthorizationError()
        assert error.message == "登录已过期, 请重新登录"
        assert error.status_code == 401

    def test_create_with_custom_message(self):
        """测试创建授权异常时使用自定义消息"""
        error = AuthorizationError("Token 已过期")
        assert error.message == "Token 已过期"
        assert error.status_code == 401

    def test_is_business_error_subclass(self):
        """测试授权异常是 BusinessError 的子类"""
        error = AuthorizationError()
        assert isinstance(error, BusinessError)
        assert isinstance(error, Exception)

    def test_status_code_differs_from_authentication(self):
        """测试授权异常的状态码与认证异常不同"""
        auth_error = AuthenticationError()
        authz_error = AuthorizationError()
        assert auth_error.status_code == 400
        assert authz_error.status_code == 401
