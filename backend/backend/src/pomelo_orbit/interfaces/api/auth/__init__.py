"""认证 API 模块"""

from pomelo_orbit.interfaces.api.auth.router import router

__all__ = ["router"]

# 为了避免循环导入，get_current_user 需要从 router 模块直接导入
# from pomelo_orbit.interfaces.api.auth.router import get_current_user
