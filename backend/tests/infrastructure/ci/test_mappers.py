"""测试 CI Mappers"""

import ulid

from pomelo_orbit.domain.ci.entities import Project
from pomelo_orbit.infrastructure.ci.mappers import ProjectMapper
from pomelo_orbit.infrastructure.ci.models import ProjectModel


class TestProjectMapper:
    """测试 ProjectMapper"""

    def test_to_domain(self):
        """测试 ORM 转领域实体"""
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            variable_overrides='{"KEY": "value"}',
            webhook_secret="my-secret",
            branch_filter="main,develop",
        )

        entity = ProjectMapper.to_domain(orm)

        assert entity.id == orm.id
        assert entity.name == orm.name
        assert entity.repository_url == orm.repository_url
        assert entity.variable_overrides == {"KEY": "value"}
        assert entity.webhook_secret == "my-secret"
        assert entity.branch_filter == "main,develop"

    def test_to_domain_without_webhook_fields(self):
        """测试 ORM 转领域实体（无 webhook 字段）"""
        orm = ProjectModel(
            id=str(ulid.ULID()),
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            variable_overrides='{}',
            webhook_secret=None,
            branch_filter=None,
        )

        entity = ProjectMapper.to_domain(orm)

        assert entity.webhook_secret is None
        assert entity.branch_filter is None

    def test_to_orm(self):
        """测试领域实体转 ORM"""
        entity = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
            webhook_secret="my-secret",
            branch_filter="main,develop",
        )

        orm = ProjectMapper.to_orm(entity)

        assert orm.id == entity.id
        assert orm.name == entity.name
        assert orm.repository_url == entity.repository_url
        assert orm.variable_overrides == '{"KEY": "value"}'
        assert orm.webhook_secret == "my-secret"
        assert orm.branch_filter == "main,develop"

    def test_to_orm_without_webhook_fields(self):
        """测试领域实体转 ORM（无 webhook 字段）"""
        entity = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
        )

        orm = ProjectMapper.to_orm(entity)

        assert orm.webhook_secret is None
        assert orm.branch_filter is None

    def test_round_trip(self):
        """测试往返转换"""
        original = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            variable_overrides={"KEY": "value"},
            webhook_secret="my-secret",
            branch_filter="main,develop",
        )

        orm = ProjectMapper.to_orm(original)
        restored = ProjectMapper.to_domain(orm)

        assert restored.id == original.id
        assert restored.name == original.name
        assert restored.repository_url == original.repository_url
        assert restored.variable_overrides == original.variable_overrides
        assert restored.webhook_secret == original.webhook_secret
        assert restored.branch_filter == original.branch_filter
