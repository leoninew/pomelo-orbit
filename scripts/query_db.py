#!/usr/bin/env python3
import sqlite3
import sys

db_path = './backend/data/db/pomelo-orbit.db'
conn = sqlite3.connect(db_path)
cursor = conn.cursor()

# 获取所有表
cursor.execute("SELECT name FROM sqlite_master WHERE type='table'")
tables = [row[0] for row in cursor.fetchall()]

print("=== Tables ===")
for table in tables:
    print(f"\n{table}:")
    cursor.execute(f"SELECT * FROM {table} LIMIT 3")
    rows = cursor.fetchall()
    if rows:
        # 获取列名
        cursor.execute(f"PRAGMA table_info({table})")
        columns = [col[1] for col in cursor.fetchall()]
        print(f"  Columns: {', '.join(columns)}")
        print(f"  Sample rows: {len(rows)}")
        for row in rows:
            print(f"    {row}")
    else:
        print("  (empty)")

conn.close()
