"""Tests for route service."""

import tempfile
from pathlib import Path

import pytest
from sqlalchemy.orm import Session

from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.domain.cd.entities import Route
from pomelo_orbit.domain.cd.route_service import RouteDomainService
from pomelo_orbit.infrastructure.cd.repositories.route import RouteRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


@pytest.fixture
def temp_config_dir():
    """Create a temporary config directory."""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield Path(tmpdir)


@pytest.fixture
def route_service(db_session: Session, temp_config_dir: Path):
    """Create a route service instance."""
    from unittest.mock import Mock, patch

    from pomelo_orbit.infrastructure.cd.traefik.manager import TraefikManager
    from pomelo_orbit.infrastructure.config import get_settings

    route_repo = RouteRepositoryImpl(db_session)
    traefik_manager = TraefikManager()
    mkcert_service = Mock()
    settings = get_settings()
    svc = RouteService(route_repo, traefik_manager, mkcert_service, settings)

    cert_dir = temp_config_dir / "certs"
    with patch.object(svc, "_get_traefik_config", return_value=(temp_config_dir, cert_dir, "traefik")):
        yield svc


class TestRouteDomainService:
    """Test route domain service."""

    def test_generate_empty_config(self):
        """Test generating empty configuration when no routes exist."""
        route = Route(
            id="test-id",
            name="test-route",
            domain="test.example.com",
            path_prefix="/",
            target_url="http://test-app:80",
            enabled=False,
            https_enabled=False,
            created_at=utc_now(),
            updated_at=utc_now(),
        )
        config = RouteDomainService.generate_route_config(route)
        assert "http" in config
        assert "routers" in config["http"]
        assert "services" in config["http"]

    def test_generate_config_with_route(self):
        """Test generating configuration with a single route."""
        route = Route(
            id="test-id",
            name="test-route",
            domain="test.example.com",
            path_prefix="/",
            target_url="http://test-app:80",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        config = RouteDomainService.generate_route_config(route)

        assert len(config["http"]["routers"]) == 1
        assert len(config["http"]["services"]) == 1

        router_name = f"{route.name}-route"
        router = config["http"]["routers"][router_name]
        assert router["rule"] == "Host(`test.example.com`)"
        assert router["entryPoints"] == ["web"]

    def test_generate_config_with_path_prefix(self):
        """Test generating configuration with path prefix."""
        route = Route(
            id="test-id",
            name="test-route",
            domain="test.example.com",
            path_prefix="/api",
            target_url="http://test-app:80",
            enabled=True,
            https_enabled=False,
            created_at=utc_now(),
            updated_at=utc_now(),
        )

        config = RouteDomainService.generate_route_config(route)

        router_name = f"{route.name}-route"
        router = config["http"]["routers"][router_name]
        assert router["rule"] == "Host(`test.example.com`) && PathPrefix(`/api`)"
        assert router["entryPoints"] == ["web"]


class TestRouteApplicationService:
    """Test route application service."""

    def test_create_route_with_enabled(self, route_service: RouteService, db_session: Session):
        """Test creating a route with enabled=True."""
        created = route_service.create_route(
            name="test-route",
            domain="test.example.com",
            path_prefix="/",
            target_url="http://test-app:80",
            enabled=True,
        )
        assert created.enabled is True
        assert created.name == "test-route"

    def test_create_route_with_disabled(self, route_service: RouteService, db_session: Session):
        """Test creating a route with enabled=False."""
        created = route_service.create_route(
            name="test-route",
            domain="test.example.com",
            path_prefix="/",
            target_url="http://test-app:80",
            enabled=False,
        )
        assert created.enabled is False
        assert created.name == "test-route"
