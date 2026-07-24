#!/usr/bin/env python3
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "web" / "src"


def main() -> None:
    for path in ROOT.rglob("*"):
        if path.suffix not in {".ts", ".vue"}:
            continue
        if "gen" in path.parts:
            continue
        original = path.read_text(encoding="utf-8")
        text = original
        text = re.sub(
            r"(from '@/gen/proto/orbit/v1/[^']+')(?!\s*;)",
            r"\1;",
            text,
        )
        # indent bare import type lines that follow an indented import in vue SFC
        text = re.sub(
            r"(  import type \{[^\n]+\} from '@/gen/proto/orbit/v1/[^']+';)\n(import type )",
            r"\1\n  \2",
            text,
        )
        if text != original:
            path.write_text(text, encoding="utf-8", newline="\n")
            print(f"fixed {path}")
    print("done")


if __name__ == "__main__":
    main()
