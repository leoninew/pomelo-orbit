"""_parse_data_json_file 单元测试"""

import json
from pathlib import Path

import pytest

from pomelo_orbit.infrastructure.migration.migrator import MigrationError, _parse_data_json_file


def make_json_file(tmp_path: Path, name: str, content: list) -> Path:
    f = tmp_path / name
    f.write_text(json.dumps(content), encoding="utf-8")
    return f


class TestInsert:
    def test_single_row(self, tmp_path):
        f = make_json_file(
            tmp_path,
            "v1.data.json",
            [{"type": "insert", "table": "user", "data": [{"username": "admin", "role": "admin"}]}],
        )
        results = _parse_data_json_file(f)
        assert len(results) == 1
        sql, params, comment = results[0]
        assert sql == 'INSERT INTO "user" ("username", "role") VALUES (:r0_username, :r0_role)'
        assert params == {"r0_username": "admin", "r0_role": "admin"}
        assert comment is None

    def test_multi_row(self, tmp_path):
        f = make_json_file(
            tmp_path, "v1.data.json", [{"type": "insert", "table": "tag", "data": [{"name": "a"}, {"name": "b"}]}]
        )
        results = _parse_data_json_file(f)
        sql, params, _ = results[0]
        assert sql == 'INSERT INTO "tag" ("name") VALUES (:r0_name), (:r1_name)'
        assert params == {"r0_name": "a", "r1_name": "b"}

    def test_with_comment(self, tmp_path):
        f = make_json_file(
            tmp_path,
            "v1.data.json",
            [{"type": "insert", "table": "user", "data": [{"name": "admin"}], "comment": "Create admin user"}],
        )
        results = _parse_data_json_file(f)
        _, _, comment = results[0]
        assert comment == "Create admin user"

    def test_batch_operations(self, tmp_path):
        f = make_json_file(
            tmp_path,
            "v1.data.json",
            [
                {"type": "insert", "table": "user", "data": [{"name": "a"}]},
                {"type": "insert", "table": "role", "data": [{"name": "admin"}]},
            ],
        )
        results = _parse_data_json_file(f)
        assert len(results) == 2
        sql1, params1, _ = results[0]
        sql2, params2, _ = results[1]
        assert "user" in sql1
        assert "role" in sql2
        assert params1 == {"r0_name": "a"}
        assert params2 == {"op1_r0_name": "admin"}  # op1_ prefix for second operation

    def test_missing_data_raises(self, tmp_path):
        f = make_json_file(tmp_path, "v1.data.json", [{"type": "insert", "table": "t"}])
        with pytest.raises(MigrationError, match="non-empty list"):
            _parse_data_json_file(f)


class TestUpdate:
    def test_basic(self, tmp_path):
        f = make_json_file(
            tmp_path,
            "v1.data.json",
            [{"type": "update", "table": "user", "data": {"role": "superadmin"}, "where": {"username": "admin"}}],
        )
        results = _parse_data_json_file(f)
        sql, params, _ = results[0]
        assert sql == 'UPDATE "user" SET "role" = :set_role WHERE "username" = :where_username'
        assert params == {"set_role": "superadmin", "where_username": "admin"}


class TestDelete:
    def test_basic(self, tmp_path):
        f = make_json_file(
            tmp_path, "v1.data.json", [{"type": "delete", "table": "session", "where": {"token": "abc"}}]
        )
        results = _parse_data_json_file(f)
        sql, params, _ = results[0]
        assert sql == 'DELETE FROM "session" WHERE "token" = :where_token'
        assert params == {"where_token": "abc"}


class TestValidation:
    def test_non_array_raises(self, tmp_path):
        f = tmp_path / "v1.data.json"
        f.write_text('{"type": "insert", "table": "t", "data": []}', encoding="utf-8")
        with pytest.raises(MigrationError, match="must be an array"):
            _parse_data_json_file(f)

    def test_invalid_type_raises(self, tmp_path):
        f = make_json_file(tmp_path, "v1.data.json", [{"type": "upsert", "table": "t"}])
        with pytest.raises(MigrationError, match=r"insert \| update \| delete"):
            _parse_data_json_file(f)


def get_backend_root():
    """获取 backend 根目录的绝对路径"""
    current = Path(__file__).resolve()
    return current.parents[4]


class TestMigrationIntegration:
    """迁移系统集成测试 - 验证所有迁移文件能在内存数据库中正确执行"""

    def test_all_migrations_run_successfully(self):
        """测试所有迁移文件能够成功执行"""
        from sqlalchemy import create_engine

        from pomelo_orbit.infrastructure.migration.migrator import run_migrations

        backend_root = get_backend_root()
        migrations_dir = backend_root / "migrations"

        # 在内存数据库中运行所有迁移
        engine = create_engine("sqlite:///:memory:")
        with engine.begin():
            run_migrations(engine, migrations_dir=str(migrations_dir))
        # 如果有异常会自动回滚
