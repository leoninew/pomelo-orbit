"""测试 CI Mappers"""

import ulid

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.domain.ci.value_objects import VariableDeclaration
from pomelo_orbit.infrastructure.ci.mappers import RepositoryMapper
from pomelo_orbit.infrastructure.ci.models import ProjectModel


class TestProjectMapper:
    def test_to_domain(self):
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides='[{"name": "KEY", "value": "value"}]',
            default_branch="main",
        )

        entity = RepositoryMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.repository_url == orm.repository_url
        assert len(entity.variable_overrides) == 1
        assert entity.variable_overrides[0].name == "KEY"
        assert entity.variable_overrides[0].value == "value"
        assert entity.default_branch == "main"

    def test_to_domain_defaults(self):
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=None,
            variable_overrides="[]",
        )

        entity = RepositoryMapper.to_domain(orm)

        assert entity.git_credential_id is None
        assert entity.variable_overrides == []

    def test_to_orm(self):
        entity = Repository.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides=[VariableDeclaration(name="KEY", value="value")],
            default_branch="main",
        )

        orm = RepositoryMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.repository_url == entity.repository_url
        assert '"name": "KEY"' in orm.variable_overrides
        assert '"value": "value"' in orm.variable_overrides
        assert orm.default_branch == "main"

    def test_to_orm_without_optional_fields(self):
        entity = Repository.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
        )

        orm = RepositoryMapper.to_orm(entity)

        assert orm.git_credential_id is None
        assert orm.variable_overrides == "[]"

    def test_round_trip(self):
        original = Repository.create(
            name="test-project",
            code="test-project",
            repository_url="https://github.com/test/repo.git",
            git_credential_id=str(ulid.ULID()),
            variable_overrides=[VariableDeclaration(name="KEY", value="value", description="test var")],
            default_branch="develop",
        )

        orm = RepositoryMapper.to_orm(original)
        restored = RepositoryMapper.to_domain(orm)

        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.repository_url == original.repository_url
        assert len(restored.variable_overrides) == len(original.variable_overrides)
        assert restored.variable_overrides[0].name == original.variable_overrides[0].name
        assert restored.variable_overrides[0].value == original.variable_overrides[0].value
        assert restored.variable_overrides[0].description == original.variable_overrides[0].description
        assert restored.default_branch == original.default_branch
