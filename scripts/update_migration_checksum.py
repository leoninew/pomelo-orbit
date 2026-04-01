#!/usr/bin/env python3
"""
更新 __migration_history 表中指定迁移文件的 checksum。

用法:
    python scripts/update_migration_checksum.py <sql_file> [datasource]

示例:
    python scripts/update_migration_checksum.py backend/migrations/v0.6.0__ci_tables.sql
    python scripts/update_migration_checksum.py backend/migrations/v0.6.0__ci_tables.sql orbit
"""

import hashlib
import subprocess
import sys
from pathlib import Path


def calculate_md5(file_path: Path) -> str:
    md5_hash = hashlib.md5()
    with file_path.open("rb") as f:
        for chunk in iter(lambda: f.read(4096), b""):
            md5_hash.update(chunk)
    return md5_hash.hexdigest()


def main() -> None:
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    sql_file = Path(sys.argv[1])
    datasource = sys.argv[2] if len(sys.argv) > 2 else "orbit"

    if not sql_file.exists():
        print(f"Error: file not found: {sql_file}")
        sys.exit(1)

    filename = sql_file.name
    checksum = calculate_md5(sql_file)
    print(f"file:       {filename}")
    print(f"checksum:   {checksum}")

    sql = f"UPDATE __migration_history SET checksum='{checksum}' WHERE filename='{filename}'"
    result = subprocess.run(
        ["pomelo-db", "-d", datasource, "-e", sql, "-w"],
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        print(f"Error: {result.stderr or result.stdout}")
        sys.exit(1)

    print(result.stdout.strip())


if __name__ == "__main__":
    main()
