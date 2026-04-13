"""CI API 模块"""

from fastapi import APIRouter

from pomelo_orbit.interfaces.api.ci.artifact import router as artifact_router
from pomelo_orbit.interfaces.api.ci.build_stage import router as build_stage_router
from pomelo_orbit.interfaces.api.ci.credential import router as credential_router
from pomelo_orbit.interfaces.api.ci.repository import router as repository_router
from pomelo_orbit.interfaces.api.ci.run import router as run_router
from pomelo_orbit.interfaces.api.ci.snapshot import router as snapshot_router
from pomelo_orbit.interfaces.api.ci.template import router as template_router
from pomelo_orbit.interfaces.api.ci.webhook import router as webhook_router

# /api/ci/* 路由（需认证）
router = APIRouter(prefix="/ci", tags=["ci"])
router.include_router(artifact_router)
router.include_router(build_stage_router)
router.include_router(credential_router)
router.include_router(template_router)
router.include_router(snapshot_router)
router.include_router(repository_router)
router.include_router(run_router)
router.include_router(webhook_router)

__all__ = ["router"]
