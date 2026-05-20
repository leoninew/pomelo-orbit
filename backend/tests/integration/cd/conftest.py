"""CD 集成测试共用 fixtures"""

import pytest

from pomelo_orbit.domain.cd.entities import TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType, TaskStatus
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
    ApplicationServiceConfigModel,
    DeploymentModel,
    RouteModel,
)
from tests.integration.conftest import DEFAULT_CI_PROJECT_ID


@pytest.fixture
def test_app(db_session):
    app = ApplicationModel(
        project_id=DEFAULT_CI_PROJECT_ID,
        name="test-app",
        code="test-app",
        image_pull_policy="missing",
    )
    db_session.add(app)
    db_session.commit()
    db_session.refresh(app)
    return app


@pytest.fixture
def test_config_file(db_session, test_app):
    config_file = ApplicationConfigFileModel(
        application_id=test_app.id,
        path=".env",
        content="KEY=value",
    )
    db_session.add(config_file)
    db_session.commit()
    db_session.refresh(config_file)
    return config_file


@pytest.fixture
def test_compose_file(db_session, test_app):
    config_file = ApplicationConfigFileModel(
        application_id=test_app.id,
        path="docker-compose.yml",
        content='services:\n  web:\n    image: nginx:1.25\n    ports:\n      - "8080:80"\n',
    )
    db_session.add(config_file)
    db_session.commit()
    db_session.refresh(config_file)
    return config_file


@pytest.fixture
def test_service_config(db_session, test_app):
    service_config = ApplicationServiceConfigModel(
        application_id=test_app.id,
        service_name="web",
        image="nginx:1.27",
    )
    db_session.add(service_config)
    db_session.commit()
    db_session.refresh(service_config)
    return service_config


@pytest.fixture
def test_deployment(db_session, test_app):
    deployment = DeploymentModel(
        project_id=DEFAULT_CI_PROJECT_ID,
        application_id=test_app.id,
        application_name=test_app.name,
        operation_type=OperationType.DEPLOY,
        trigger_type=TriggerType.MANUAL,
        status=TaskStatus.RAN_TO_COMPLETION,
    )
    db_session.add(deployment)
    db_session.commit()
    db_session.refresh(deployment)
    return deployment


@pytest.fixture
def test_route(db_session):
    route = RouteModel(
        project_id=DEFAULT_CI_PROJECT_ID,
        name="test-route",
        domain="test.example.com",
        path_prefix="/api",
        target_url="http://backend:8000",
        enabled=True,
        https_enabled=False,
    )
    db_session.add(route)
    db_session.commit()
    db_session.refresh(route)
    return route


@pytest.fixture
def disabled_route(db_session):
    route = RouteModel(
        project_id=DEFAULT_CI_PROJECT_ID,
        name="disabled-route",
        domain="disabled.example.com",
        path_prefix="/",
        target_url="http://app:8080",
        enabled=False,
        https_enabled=False,
    )
    db_session.add(route)
    db_session.commit()
    db_session.refresh(route)
    return route
