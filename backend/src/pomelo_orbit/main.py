"""
FastAPI Application Entry Point
"""

import argparse
import logging
import sys
from contextlib import asynccontextmanager
from pathlib import Path

import uvicorn
from dishka.integrations.fastapi import setup_dishka
from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.infrastructure import get_cors_config, get_settings
from pomelo_orbit.infrastructure.container import create_container
from pomelo_orbit.infrastructure.logging import LOGGING_CONFIG, RequestLoggingMiddleware
from pomelo_orbit.infrastructure.migration.migrator import run_migrations
from pomelo_orbit.infrastructure.persistence.database import get_engine
from pomelo_orbit.interfaces.api.auth.router import router as auth_router
from pomelo_orbit.interfaces.api.cd import router as cd_router
from pomelo_orbit.interfaces.api.ci import router as ci_router
from pomelo_orbit.interfaces.api.settings.router import router as settings_router

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan events"""
    logger.info("Running database migrations...")
    run_migrations(get_engine())
    logger.info("Database migrations completed successfully!")

    yield

    await app.state.dishka_container.close()
    logger.info("Shutting down...")


settings = get_settings()
container = create_container()

app = FastAPI(
    title=settings.app.name,
    description="Pomelo Orbit API",
    version=settings.app.version,
    debug=settings.app.debug,
    lifespan=lifespan,
)

# Configure CORS
cors_config = get_cors_config()
app.add_middleware(CORSMiddleware, **cors_config)
app.add_middleware(RequestLoggingMiddleware)

setup_dishka(container=container, app=app)

# Include routers
app.include_router(auth_router, prefix="/api")
app.include_router(cd_router, prefix="/api")
app.include_router(settings_router, prefix="/api")
app.include_router(ci_router, prefix="/api")


@app.exception_handler(BusinessError)
async def business_exception_handler(request: Request, exc: BusinessError) -> JSONResponse:
    """Handle business exceptions"""
    logger.warning(
        f"Business error: {request.method} {request.url.path}, status={exc.status_code}, message={exc.message}"
    )
    return JSONResponse(
        status_code=exc.status_code,
        content={"detail": exc.message},
    )


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError) -> JSONResponse:
    """Handle request validation errors (422)"""
    errors = "; ".join(f"{'.'.join(str(loc_part) for loc_part in e['loc'])}: {e['msg']}" for e in exc.errors())
    logger.warning(f"Validation error: {request.method} {request.url.path}, errors={errors}")
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_CONTENT,
        content={"detail": errors},
    )


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception) -> JSONResponse:
    """Global exception handler"""
    logger.error(f"Unhandled exception: {request.method} {request.url.path}", exc_info=True)
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={"detail": "Internal server error"},
    )


@app.get("/docs")
async def root():
    """Root endpoint"""
    return {
        "message": settings.app.name,
        "version": settings.app.version,
        "docs": "/docs",
    }


@app.get("/api/health")
async def health():
    """Health check endpoint"""
    return {"status": "ok"}


# Catch-all for undefined API routes - must be registered AFTER all business routes
@app.api_route("/api/{full_path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"])
async def api_not_found(full_path: str):
    """Prevent API paths from falling through to the SPA fallback."""
    return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"detail": "Not Found"})


# Mount static files for frontend (if exists)
static_dir = Path("static")
if static_dir.exists() and static_dir.is_dir():
    # SPA fallback - serve index.html for non-API routes, or static files if they exist
    @app.get("/{full_path:path}")
    async def serve_spa(full_path: str):
        file_path = static_dir / full_path

        # If the file exists (favicon, logo, assets/*, etc.), serve it
        if file_path.is_file():
            return FileResponse(file_path)

        # Otherwise, serve index.html for SPA routing
        return FileResponse(static_dir / "index.html")


def main():
    parser = argparse.ArgumentParser(description="Start Pomelo Orbit FastAPI Service")
    parser.add_argument("--host", default="localhost", help="Server host (default: localhost)")
    parser.add_argument("--port", type=int, default=80, help="Server port (default: 80)")
    parser.add_argument("--reload", action="store_true", help="Enable auto-reload for development")
    args = parser.parse_args()

    # 开发模式下静默跳过静态目录检查
    if not args.reload and not static_dir.exists():
        logger.warning(f"Static directory not found: path={static_dir.absolute()}")

    # 创建日志目录
    log_dir = Path("logs")
    log_dir.mkdir(parents=True, exist_ok=True)

    uvicorn_config = {
        "app": "pomelo_orbit.main:app",
        "host": args.host,
        "port": args.port,
        "log_config": LOGGING_CONFIG,
    }

    # 开发模式：启用 reload
    if args.reload:
        uvicorn_config.update(
            {
                "reload": True,
                "reload_dirs": ["src"],
            }
        )

    uvicorn.run(**uvicorn_config)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(1)
