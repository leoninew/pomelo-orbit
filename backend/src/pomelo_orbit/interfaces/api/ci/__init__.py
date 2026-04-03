"""CI API 模块"""

from fastapi import APIRouter

from pomelo_orbit.interfaces.api.ci.credentials import router as credentials_router
from pomelo_orbit.interfaces.api.ci.jobs import router as jobs_router
from pomelo_orbit.interfaces.api.ci.projects import router as projects_router
from pomelo_orbit.interfaces.api.ci.runs import router as runs_router
from pomelo_orbit.interfaces.api.ci.snapshots import router as snapshots_router
from pomelo_orbit.interfaces.api.ci.templates import router as templates_router
from pomelo_orbit.interfaces.api.ci.webhooks import router as webhooks_router

router = APIRouter(prefix="/ci", tags=["ci"])

router.include_router(credentials_router)
router.include_router(templates_router)
router.include_router(snapshots_router)
router.include_router(projects_router)
router.include_router(runs_router)
router.include_router(jobs_router)
router.include_router(webhooks_router)

__all__ = ["router"]
