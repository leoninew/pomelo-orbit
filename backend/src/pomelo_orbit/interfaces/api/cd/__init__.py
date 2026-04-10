"""CD API 模块"""

from fastapi import APIRouter

from pomelo_orbit.interfaces.api.cd.application import router as application_router
from pomelo_orbit.interfaces.api.cd.deployment import router as deployment_router
from pomelo_orbit.interfaces.api.cd.route import router as route_router
from pomelo_orbit.interfaces.api.cd.traefik_route import router as traefik_route_router

# 创建主路由器，组合所有子路由
router = APIRouter(prefix="/cd", tags=["cd"])

# 注册子路由
router.include_router(application_router)
router.include_router(deployment_router)
router.include_router(route_router)
router.include_router(traefik_route_router)

__all__ = ["router"]
