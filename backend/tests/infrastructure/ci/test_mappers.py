"""测试 CI Mappers"""

import ulid

from pomelo_orbit.domain.ci.entities import Project
from pomelo_orbit.infrastructure.ci.mappers import ProjectMapper
from pomelo_orbit.infrastructure.ci.models import ProjectModel


class TestProjectMapper:
    def test_to_domain(self):
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides='{"KEY": "value"}',
            default_branch="main",
        )

        entity = ProjectMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.repository_url == orm.repository_url
        assert entity.variable_overrides == {"KEY": "value"}
        assert entity.default_branch == "main"

    def test_to_domain_defaults(self):
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=None,
            variable_overrides="{}",
        )

        entity = ProjectMapper.to_domain(orm)

        assert entity.git_credential_id is None
        assert entity.variable_overrides == {}

    def test_to_orm(self):
        entity = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
            default_branch="main",
        )

        orm = ProjectMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.repository_url == entity.repository_url
        assert orm.variable_overrides == '{"KEY": "value"}'
        assert orm.default_branch == "main"

    def test_to_orm_without_optional_fields(self):
        entity = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
        )

        orm = ProjectMapper.to_orm(entity)

        assert orm.git_credential_id is None

    def test_round_trip(self):
        original = Project.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
            default_branch="develop",
        )

        orm = ProjectMapper.to_orm(original)
        restored = ProjectMapper.to_domain(orm)

        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.repository_url == original.repository_url
        assert restored.variable_overrides == original.variable_overrides
        assert restored.default_branch == original.default_branch
