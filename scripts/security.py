#!/usr/bin/env python3
"""
密码哈希工具

用于生成和验证密码哈希值，方便创建测试用户或管理员账号。

使用方法:
    # 生成密码哈希
    python scripts/security.py hash <password>

    # 交互式输入密码（不显示明文）
    python scripts/security.py hash

    # 验证密码
    python scripts/security.py verify <password> <hash>

    # 交互式验证
    python scripts/security.py verify

示例:
    python scripts/security.py hash mypassword123
    python scripts/security.py verify mypassword123 '$2b$12$...'
"""

import argparse
import getpass
import sys
from pathlib import Path

# ruff: noqa: E402

# 添加项目根目录到 Python 路径
project_root = Path(__file__).parent.parent / "backend"
sys.path.insert(0, str(project_root / "src"))

from pomelo_orbit.infrastructure import (  # type: ignore[import-untyped]
    hash_password,
    verify_password,
)


def hash_command(password: str | None = None) -> None:
    """生成密码哈希"""
    if password is None:
        # 交互式输入（不显示明文）
        password = getpass.getpass("请输入密码: ")
        if not password:
            print("错误: 密码不能为空", file=sys.stderr)
            sys.exit(1)

    # 生成哈希
    hashed = hash_password(password)

    print("\n密码哈希生成成功:")
    print("-" * 80)
    print(hashed)
    print("-" * 80)
    print("\n可以将此哈希值用于数据库中的 password_hash 字段")


def verify_command(password: str | None = None, hashed: str | None = None) -> None:
    """验证密码哈希"""
    if password is None:
        password = getpass.getpass("请输入密码: ")
        if not password:
            print("错误: 密码不能为空", file=sys.stderr)
            sys.exit(1)

    if hashed is None:
        hashed = input("请输入哈希值: ").strip()
        if not hashed:
            print("错误: 哈希值不能为空", file=sys.stderr)
            sys.exit(1)

    # 验证
    is_valid = verify_password(password, hashed)

    print("\n验证结果:")
    print("-" * 80)
    if is_valid:
        print("✓ 密码匹配")
    else:
        print("✗ 密码不匹配")
    print("-" * 80)

    sys.exit(0 if is_valid else 1)


def main():
    parser = argparse.ArgumentParser(
        description="密码哈希工具",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 生成密码哈希
  python scripts/security.py hash mypassword123

  # 交互式输入密码（推荐，不显示明文）
  python scripts/security.py hash

  # 验证密码
  python scripts/security.py verify mypassword123 '$2b$12$...'

  # 交互式验证
  python scripts/security.py verify
        """,
    )

    subparsers = parser.add_subparsers(dest="command", help="子命令")

    # hash 子命令
    hash_parser = subparsers.add_parser("hash", help="生成密码哈希")
    hash_parser.add_argument(
        "password", nargs="?", help="密码（可选，不提供则交互式输入）"
    )

    # verify 子命令
    verify_parser = subparsers.add_parser("verify", help="验证密码哈希")
    verify_parser.add_argument(
        "password", nargs="?", help="密码（可选，不提供则交互式输入）"
    )
    verify_parser.add_argument(
        "hash", nargs="?", help="哈希值（可选，不提供则交互式输入）"
    )

    args = parser.parse_args()

    if args.command == "hash":
        hash_command(args.password)
    elif args.command == "verify":
        verify_command(args.password, args.hash)
    else:
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
