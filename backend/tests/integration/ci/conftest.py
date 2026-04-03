"""CI 集成测试共用 fixtures"""

import pytest

from pomelo_orbit.infrastructure.ci.models import (  # noqa: F401 - 触发 CI 表注册
    CredentialModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)


@pytest.fixture
def test_template(db_session):
    tmpl = PipelineTemplateModel(
        name="test-template",
        description="A test template",
        content="version: v1\nsteps:\n  - name: build\n    image: alpine\n    commands:\n      - echo hello",
        variable_declarations="[]",
        is_builtin=0,
    )
    db_session.add(tmpl)
    db_session.commit()
    db_session.refresh(tmpl)
    return tmpl


@pytest.fixture
def test_credential(db_session):
    cred = CredentialModel(
        name="test-cred",
        type="git_token",
        encrypted_data="encrypted-token",
    )
    db_session.add(cred)
    db_session.commit()
    db_session.refresh(cred)
    return cred


@pytest.fixture
def test_project(db_session, test_template, test_credential):
    project = ProjectModel(
        name="test-project",
        repository_url="https://github.com/test/repo.git",
        pipeline_template_id=test_template.id,
        git_credential_id=test_credential.id,
        variable_overrides="{}",
        webhook_secret="test-secret",
    )
    db_session.add(project)
    db_session.commit()
    db_session.refresh(project)
    return project
