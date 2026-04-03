"""CI 集成测试共用 fixtures"""

import pytest

from pomelo_orbit.infrastructure.ci.models import (  # noqa: F401 - 触发 CI 表注册
    CredentialModel,
    PipelineRunModel,
    PipelineSnapshotModel,
    PipelineTemplateModel,
    ProjectModel,
)


@pytest.fixture
def test_template(db_session):
    tmpl = PipelineTemplateModel(
        name="test-template",
        description="A test template",
        stages='[{"name": "build", "type": "checkout", "config": {"ref": "master"}}]',
        variable_declarations="[]",
        is_builtin=0,
    )
    db_session.add(tmpl)
    db_session.commit()
    db_session.refresh(tmpl)
    return tmpl


@pytest.fixture
def test_snapshot(db_session, test_template):
    snapshot = PipelineSnapshotModel(
        template_id=test_template.id,
        version=1,
        stages_snapshot=test_template.stages,
        variable_declarations_snapshot=test_template.variable_declarations,
    )
    db_session.add(snapshot)
    db_session.commit()
    db_session.refresh(snapshot)
    return snapshot


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
def test_project(db_session, test_snapshot, test_credential):
    project = ProjectModel(
        name="test-project",
        repository_url="https://github.com/test/repo.git",
        pipeline_snapshot_id=test_snapshot.id,
        git_credential_id=test_credential.id,
        variable_overrides="{}",
    )
    db_session.add(project)
    db_session.commit()
    db_session.refresh(project)
    return project
