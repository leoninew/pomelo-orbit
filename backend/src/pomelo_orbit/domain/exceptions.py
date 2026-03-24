"""
领域异常定义
"""


class BusinessError(Exception):
    """业务异常基类"""

    def __init__(self, message: str, status_code: int = 400):
        self.message = message
        self.status_code = status_code
        super().__init__(self.message)


class AuthenticationError(BusinessError):
    """认证失败异常（用户名密码错误等）"""

    def __init__(self, message: str = "用户名或密码错误"):
        super().__init__(message, status_code=400)


class AuthorizationError(BusinessError):
    """授权失败异常（token 无效或过期）"""

    def __init__(self, message: str = "登录已过期, 请重新登录"):
        super().__init__(message, status_code=401)
