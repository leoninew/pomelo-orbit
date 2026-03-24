"""Traefik infrastructure module"""

from pomelo_orbit.infrastructure.traefik.client import TraefikAPIClient
from pomelo_orbit.infrastructure.traefik.di import get_traefik_manager
from pomelo_orbit.infrastructure.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.traefik.models import TraefikRouter

__all__ = ["TraefikAPIClient", "TraefikManager", "TraefikRouter", "get_traefik_manager"]
