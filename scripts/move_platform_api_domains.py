#!/usr/bin/env python3
from __future__ import annotations

import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
API = ROOT / "web" / "src" / "api"
WEB = ROOT / "web" / "src"

MOVES = {
    "auth.ts": "auth/auth.ts",
    "user.ts": "user/user.ts",
    "role.ts": "role/role.ts",
    "project.ts": "project/project.ts",
    "settings.ts": "settings/settings.ts",
}

PATH_MAP = {
    "@/api/auth": "@/api/auth/auth",
    "@/api/user": "@/api/user/user",
    "@/api/role": "@/api/role/role",
    "@/api/project": "@/api/project/project",
    "@/api/settings": "@/api/settings/settings",
}


def main() -> None:
    for old, new in MOVES.items():
        src = API / old
        dst = API / new
        if not src.exists():
            if dst.exists():
                print(f"skip {old}")
                continue
            raise SystemExit(f"missing {src}")
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(src), str(dst))
        print(f"moved {old} -> {new}")

    for path in WEB.rglob("*"):
        if path.suffix not in {".ts", ".vue"}:
            continue
        original = path.read_text(encoding="utf-8")
        text = original
        for old, new in PATH_MAP.items():
            text = text.replace(f"from '{old}'", f"from '{new}'")
            text = text.replace(f'from "{old}"', f'from "{new}"')
        if text != original:
            path.write_text(text, encoding="utf-8", newline="\n")
            print(f"rewrote {path.relative_to(ROOT)}")
    print("done")


if __name__ == "__main__":
    main()
