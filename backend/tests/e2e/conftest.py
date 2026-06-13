"""E2E test fixtures using the real app lifespan and migrated schema."""

from collections.abc import Iterator
from pathlib import Path
from typing import Any

import pytest
from fastapi.testclient import TestClient
from jose import jwt
from sqlalchemy import create_engine
from sqlalchemy.orm import Session, sessionmaker

from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.persistence.models import (
    PermissionModel,
    ProjectMemberModel,
    ProjectModel,
    RoleModel,
    RolePermissionModel,
    UserModel,
    UserRoleModel,
)
from pomelo_orbit.infrastructure.security import hash_password

E2E_PROJECT_ID = "01E2E3NDEKTSV4RRFFQ69G5FAV"
E2E_JWT_SECRET = "mNGu9ayJZ54VVgsuJzcU7gJLXwHnWRjnfbg8ISzvqkQ="


@pytest.fixture
def e2e_client(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> Iterator[TestClient]:
    """Create a real app instance backed by a migrated file SQLite database."""
    db_path = tmp_path / "e2e.sqlite"
    traefik_dir = tmp_path / "traefik" / "dynamic"
    cert_dir = tmp_path / "traefik" / "certs"

    monkeypatch.setenv("POMELO_ORBIT_DATABASE__TYPE", "sqlite")
    monkeypatch.setenv("POMELO_ORBIT_DATABASE__SQLITE__PATH", str(db_path))
    monkeypatch.setenv("POMELO_ORBIT_JWT__SECRET_KEY", E2E_JWT_SECRET)
    monkeypatch.setenv("POMELO_ORBIT_TRAEFIK__DYNAMIC_ROUTE_DIR", str(traefik_dir))
    monkeypatch.setenv("POMELO_ORBIT_TRAEFIK__CERT_DIR", str(cert_dir))

    from pomelo_orbit.infrastructure.config import get_settings
    from pomelo_orbit.infrastructure.persistence.database import get_engine, get_session_factory

    get_settings.cache_clear()
    get_engine.cache_clear()
    get_session_factory.cache_clear()

    import pomelo_orbit.infrastructure.container as container_module
    import pomelo_orbit.infrastructure.persistence.database as db_module
    import pomelo_orbit.infrastructure.persistence.di as persistence_di_module
    import pomelo_orbit.main as main_module

    test_engine = create_engine(f"sqlite:///{db_path}", pool_pre_ping=True)
    test_session_factory = sessionmaker(autocommit=False, autoflush=True, bind=test_engine)
    monkeypatch.setattr(db_module, "get_engine", lambda: test_engine)
    monkeypatch.setattr(db_module, "get_session_factory", lambda: test_session_factory)
    monkeypatch.setattr(main_module, "get_engine", lambda: test_engine)
    monkeypatch.setattr(container_module, "get_session_factory", lambda: test_session_factory)
    monkeypatch.setattr(persistence_di_module, "get_session_factory", lambda: test_session_factory)

    import pomelo_orbit.infrastructure.ci.workspace as workspace_module

    create_app = main_module.create_app

    data_dir = tmp_path / "data"
    monkeypatch.setattr(workspace_module, "_ci_root", lambda: data_dir / "ci")
    monkeypatch.setattr(workspace_module, "_get_physical_data_dir", lambda: data_dir)

    app = create_app()
    with TestClient(app) as client:
        yield client

    get_settings.cache_clear()
    get_engine.cache_clear()
    get_session_factory.cache_clear()
    test_engine.dispose()


@pytest.fixture
def e2e_db(e2e_client: TestClient) -> Iterator[Session]:
    """Open a real database session against the E2E app database."""
    from pomelo_orbit.infrastructure.persistence.database import get_session_factory

    session = get_session_factory()()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()


@pytest.fixture(autouse=True)
def fake_container_executor(monkeypatch: pytest.MonkeyPatch) -> None:
    """Avoid Docker while keeping the real pipeline execution service path."""

    async def fake_run(
        self: ContainerExecutor,
        image: str,
        commands: list[str] | None,
        volumes: list[Any],
        environment: dict[str, str] | None,
        entrypoint: str = "sh",
        log_file: Any = None,
    ) -> tuple[int, str]:
        output = "\n".join(commands or [f"echo e2e {image}"])
        artifacts_dir = next((Path(volume.host_path) for volume in volumes if volume.container_path == "/artifacts"), None)
        if artifacts_dir is not None:
            artifact_path = artifacts_dir / "dist" / "app.tar.gz"
            artifact_path.parent.mkdir(parents=True, exist_ok=True)
            artifact_path.write_text("e2e artifact", encoding="utf-8")
        if log_file is not None:
            log_file.write(output)
            log_file.flush()
        return 0, output

    monkeypatch.setattr(ContainerExecutor, "run", fake_run)


def seed_user(
    session: Session,
    username: str = "e2euser",
    password: str = "testpass123",
    user_id: str = "e2e-user-id",
    permission_codes: list[str] | None = None,
    project_id: str = E2E_PROJECT_ID,
) -> UserModel:
    user = session.get(UserModel, user_id)
    if user is None:
        user = UserModel(
            id=user_id,
            username=username,
            password_hash=hash_password(password),
            status="enabled",
            oauth_provider="",
            oauth_provider_id="",
            auth_source="password",
        )
        session.add(user)
    else:
        user.username = username
        user.password_hash = hash_password(password)
        user.status = "enabled"

    project = session.get(ProjectModel, project_id)
    if project is None:
        project_code = "e2e" if project_id == E2E_PROJECT_ID else f"e2e-{project_id[-6:].lower()}"
        project = ProjectModel(id=project_id, name="E2E Project", code=project_code, is_active=True)
        session.add(project)

    if session.get(ProjectMemberModel, (project_id, user_id)) is None:
        session.add(ProjectMemberModel(project_id=project_id, user_id=user_id))

    seed_permissions(session, user_id, permission_codes or [])
    session.commit()
    session.refresh(user)
    return user


def seed_permissions(session: Session, user_id: str, permission_codes: list[str]) -> None:
    if not permission_codes:
        return

    role_id = f"role-{user_id}"
    role = session.get(RoleModel, role_id)
    if role is None:
        role = RoleModel(id=role_id, code=f"role-{user_id}", name=f"Role {user_id}")
        session.add(role)
    session.flush()

    for code in permission_codes:
        permission = session.query(PermissionModel).filter(PermissionModel.code == code).one_or_none()
        if permission is None:
            permission = PermissionModel(id=f"perm-{code.replace(':', '-')}", code=code, name=code)
            session.add(permission)
            session.flush()
        if session.get(RolePermissionModel, (role.id, permission.id)) is None:
            session.add(RolePermissionModel(role_id=role.id, permission_id=permission.id))

    if session.get(UserRoleModel, (user_id, role.id)) is None:
        session.add(UserRoleModel(user_id=user_id, role_id=role.id))


def auth_headers(client: TestClient, username: str = "e2euser", password: str = "testpass123") -> dict[str, str]:
    token = login(client, username=username, password=password)
    return {"Authorization": f"Bearer {token}"}


def login(client: TestClient, username: str = "e2euser", password: str = "testpass123") -> str:
    csrf_token = client.get("/api/auth/csrf-token").json()["token"]
    captcha_token = client.get("/api/auth/captcha").json()["token"]
    captcha_answer = decode_captcha_answer(captcha_token)
    response = client.post(
        "/api/auth/login",
        json={
            "username": username,
            "password": password,
            "csrf_token": csrf_token,
            "captcha_token": captcha_token,
            "captcha_answer": captcha_answer,
        },
    )
    assert response.status_code == 200, response.text
    token = response.json()["access_token"]
    assert isinstance(token, str)
    return token


def decode_captcha_answer(captcha_token: str) -> str:
    payload = jwt.decode(captcha_token, E2E_JWT_SECRET, algorithms=["HS256"])
    answer: str = payload["answer"]
    return answer
