"""Tests for RouteService."""

from unittest.mock import Mock

import pytest

from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.domain.cd.entities import CertType, Route


@pytest.fixture
def route_repo():
    """Mock route repository."""
    return Mock()


@pytest.fixture
def mock_settings():
    """Mock settings."""
    settings = Mock()
    settings.traefik.dynamic_route_dir = "traefik/dynamic"
    settings.traefik.cert_dir = "traefik/certs"
    settings.traefik.container_name = "traefik"
    settings.letsencrypt.enabled = True
    settings.letsencrypt.email = "test@example.com"
    return settings


@pytest.fixture
def route_service(route_repo, mock_settings):
    """Create RouteService with mocked dependencies."""
    traefik_manager = Mock()
    mkcert_service = Mock()
    return RouteService(
        route_repo=route_repo,
        traefik_manager=traefik_manager,
        mkcert_service=mkcert_service,
        settings=mock_settings,
    )


@pytest.fixture
def sample_route():
    """Create a sample route."""
    return Route(
        id="route-1",
        name="test-route",
        domain="example.com",
        path_prefix="/",
        target_url="http://localhost:8080",
        enabled=True,
        https_enabled=False,
        cert_type=CertType.MANUAL,
    )


class TestCreateRoute:
    """Tests for create_route method."""

    def test_saves_route_to_repository(self, route_service, route_repo, sample_route):
        """Should save route to repository."""
        route_service.create_route(
            name=sample_route.name,
            domain=sample_route.domain,
            path_prefix=sample_route.path_prefix,
            target_url=sample_route.target_url,
            enabled=sample_route.enabled,
        )
        route_repo.save.assert_called_once()

    def test_deploys_route_when_enabled(self, route_service, sample_route):
        """Should deploy route configuration when enabled."""
        route_service.create_route(
            name=sample_route.name,
            domain=sample_route.domain,
            path_prefix=sample_route.path_prefix,
            target_url=sample_route.target_url,
            enabled=True,
        )
        route_service.traefik_manager.deploy_route.assert_called_once()

    def test_skips_deploy_when_disabled(self, route_service, sample_route):
        """Should not deploy route when disabled."""
        route_service.create_route(
            name=sample_route.name,
            domain=sample_route.domain,
            path_prefix=sample_route.path_prefix,
            target_url=sample_route.target_url,
            enabled=False,
        )
        route_service.traefik_manager.deploy_route.assert_not_called()


class TestGetRoute:
    """Tests for get_route method."""

    def test_returns_route_from_repository(self, route_service, route_repo, sample_route):
        """Should return route from repository."""
        route_repo.find_by_id.return_value = sample_route
        result = route_service.get_route("route-1")
        assert result == sample_route
        route_repo.find_by_id.assert_called_once_with("route-1")

    def test_returns_none_when_not_found(self, route_service, route_repo):
        """Should raise BusinessError when route not found."""
        from pomelo_orbit.domain.exceptions import BusinessError

        route_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError, match="Route nonexistent not found"):
            route_service.get_route("nonexistent")


class TestUpdateRoute:
    """Tests for update_route method."""

    def test_raises_error_when_route_not_found(self, route_service, route_repo, sample_route):
        """Should raise BusinessError when route not found."""
        from pomelo_orbit.domain.exceptions import BusinessError

        route_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError, match="Route route-1 not found"):
            route_service.update_route("route-1", name="new-name")

    def test_saves_updated_route(self, route_service, route_repo, sample_route):
        """Should save updated route to repository."""
        route_repo.find_by_id.return_value = sample_route
        route_service.update_route("route-1", name="new-name")
        route_repo.save.assert_called_once()

    def test_deploys_route_when_enabled(self, route_service, route_repo, sample_route):
        """Should deploy route when enabled."""
        route_repo.find_by_id.return_value = sample_route
        sample_route.enabled = True
        route_service.update_route("route-1", name="new-name")
        route_service.traefik_manager.deploy_route.assert_called_once()

    def test_revokes_route_when_disabled(self, route_service, route_repo, sample_route):
        """Should revoke route when disabled."""
        route_repo.find_by_id.return_value = sample_route
        sample_route.enabled = False
        route_service.update_route("route-1", name="new-name")
        route_service.traefik_manager.revoke_route.assert_called_once()

    def test_deploys_cert_when_https_enabled_with_cert(self, route_service, route_repo, sample_route):
        """Should deploy certificate when HTTPS enabled with cert."""
        route_repo.find_by_id.return_value = sample_route
        sample_route.enabled = True
        sample_route.https_enabled = True
        sample_route.cert_pem = "cert-content"
        sample_route.cert_key = "key-content"
        route_service.update_route("route-1", name="new-name")
        # After update, the route name is "new-name"
        # Check that deploy_cert was called with the correct route name and cert content
        # (ignoring cert_dir parameter)
        assert route_service.traefik_manager.deploy_cert.called
        call_args = route_service.traefik_manager.deploy_cert.call_args
        assert call_args[0][0] == "new-name"  # route_name
        assert call_args[0][1] == "cert-content"  # cert_pem
        assert call_args[0][2] == "key-content"  # cert_key

    def test_revokes_cert_when_https_disabled(self, route_service, route_repo, sample_route):
        """Should revoke certificate when HTTPS disabled."""
        route_repo.find_by_id.return_value = sample_route
        sample_route.https_enabled = False
        route_service.update_route("route-1", name="new-name")
        # After update, the route name is "new-name"
        # Check that revoke_cert was called with the correct route name (ignoring cert_dir parameter)
        assert route_service.traefik_manager.revoke_cert.called
        call_args = route_service.traefik_manager.revoke_cert.call_args
        assert call_args[0][0] == "new-name"  # route_name


class TestEnableRoute:
    """Tests for enable_route method."""

    def test_raises_error_when_route_not_found(self, route_service, route_repo):
        """Should raise BusinessError when route not found."""
        from pomelo_orbit.domain.exceptions import BusinessError

        route_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError, match="Route route-1 not found"):
            route_service.enable_route("route-1")

    def test_enables_route_and_saves(self, route_service, route_repo, sample_route):
        """Should enable route and save to repository."""
        sample_route.enabled = False
        route_repo.find_by_id.return_value = sample_route
        route_service.enable_route("route-1")
        assert sample_route.enabled is True
        route_repo.save.assert_called_once_with(sample_route)

    def test_deploys_route_configuration(self, route_service, route_repo, sample_route):
        """Should deploy route configuration."""
        route_repo.find_by_id.return_value = sample_route
        route_service.enable_route("route-1")
        # Check that deploy_route was called with the route (ignoring config_dir and traefik_container)
        assert route_service.traefik_manager.deploy_route.called
        call_args = route_service.traefik_manager.deploy_route.call_args
        assert call_args[0][0] == sample_route  # route

    def test_deploys_cert_when_https_enabled(self, route_service, route_repo, sample_route):
        """Should deploy certificate when HTTPS enabled."""
        sample_route.https_enabled = True
        sample_route.cert_pem = "cert-content"
        sample_route.cert_key = "key-content"
        route_repo.find_by_id.return_value = sample_route
        route_service.enable_route("route-1")
        # Check that deploy_cert was called with correct parameters (ignoring cert_dir)
        assert route_service.traefik_manager.deploy_cert.called
        call_args = route_service.traefik_manager.deploy_cert.call_args
        assert call_args[0][0] == "test-route"  # route_name
        assert call_args[0][1] == "cert-content"  # cert_pem
        assert call_args[0][2] == "key-content"  # cert_key


class TestDisableRoute:
    """Tests for disable_route method."""

    def test_raises_error_when_route_not_found(self, route_service, route_repo):
        """Should raise BusinessError when route not found."""
        from pomelo_orbit.domain.exceptions import BusinessError

        route_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError, match="Route route-1 not found"):
            route_service.disable_route("route-1")

    def test_disables_route_and_saves(self, route_service, route_repo, sample_route):
        """Should disable route and save to repository."""
        sample_route.enabled = True
        route_repo.find_by_id.return_value = sample_route
        route_service.disable_route("route-1")
        assert sample_route.enabled is False
        route_repo.save.assert_called_once_with(sample_route)

    def test_revokes_route_configuration(self, route_service, route_repo, sample_route):
        """Should revoke route configuration."""
        route_repo.find_by_id.return_value = sample_route
        route_service.disable_route("route-1")
        # Check that revoke_route was called with the route (ignoring config_dir and traefik_container)
        assert route_service.traefik_manager.revoke_route.called
        call_args = route_service.traefik_manager.revoke_route.call_args
        assert call_args[0][0] == sample_route  # route
