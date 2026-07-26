#!/usr/bin/env python3
"""Rewrite flat pomeloorbit gen/proto imports to domain packages."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
GEN = ROOT / "internal" / "gen" / "proto" / "orbit" / "v1"
BASE = "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"

# Spec aliases
DOMAIN_ALIAS = {
    "auth": "authv1",
    "user": "userv1",
    "role": "rolev1",
    "project": "projectv1",
    "settings": "settingsv1",
    "task": "taskv1",
    "credential": "credentialv1",
    "repository": "repositoryv1",
    "pipeline": "pipelinev1",
    "pipeline_run": "pipelinerunv1",
    "application": "applicationv1",
    "service": "servicev1",
    "deployment": "deploymentv1",
    "gateway": "gatewayv1",
    "route": "routev1",
    "common": "commonv1",
}

# Symbol renames applied before domain mapping
RENAMES = {
    "BuildStageResp": "PipelineStageResp",
    "BuildStageCreateReq": "PipelineStageCreateReq",
    "BuildStageUpdateReq": "PipelineStageUpdateReq",
    "BuildStageDuplicateReq": "PipelineStageDuplicateReq",
    "BuildStagePaginatedResp": "PipelineStagePaginatedResp",
    "StageRunResp": "PipelineStageRunResp",
}


def build_type_map() -> dict[str, str]:
    type_to_domain: dict[str, str] = {}
    for pb in GEN.rglob("*.pb.go"):
        domain = pb.parent.name
        if domain not in DOMAIN_ALIAS:
            raise SystemExit(f"unknown domain directory: {pb}")
        text = pb.read_text(encoding="utf-8")
        for name in re.findall(r"^type (\w+) struct", text, re.M):
            if name in type_to_domain and type_to_domain[name] != domain:
                raise SystemExit(f"duplicate type {name} in {type_to_domain[name]} and {domain}")
            type_to_domain[name] = domain
    # map old names to domains of new names
    for old, new in RENAMES.items():
        if new not in type_to_domain:
            raise SystemExit(f"renamed type missing in gen: {new}")
        type_to_domain[old] = type_to_domain[new]
    return type_to_domain


FLAT_IMPORT_RE = re.compile(
    r'(?m)^(\t)(?:pomeloorbit\s+)?"' + re.escape(BASE) + r'"\s*$'
)
TYPE_REF_RE = re.compile(r"\bpomeloorbit\.(\w+)\b")


def rewrite_file(path: Path, type_to_domain: dict[str, str]) -> bool:
    original = path.read_text(encoding="utf-8")
    if BASE not in original and "pomeloorbit." not in original:
        return False

    text = original

    # Apply symbol renames on pomeloorbit.X refs first
    def rename_ref(m: re.Match[str]) -> str:
        name = m.group(1)
        return f"pomeloorbit.{RENAMES.get(name, name)}"

    text = TYPE_REF_RE.sub(rename_ref, text)

    # Collect used types after rename
    used = sorted(set(TYPE_REF_RE.findall(text)))
    if not used and f'"{BASE}"' not in text:
        return False

    domains_needed: set[str] = set()
    for name in used:
        domain = type_to_domain.get(name)
        if domain is None:
            raise SystemExit(f"{path}: unknown type pomeloorbit.{name}")
        domains_needed.add(domain)

    # If import present but no type refs (shouldn't happen), drop import later
    if not domains_needed and f'"{BASE}"' in text:
        # keep file if only import leftover - remove import
        text2 = re.sub(
            r'(?m)^\t(?:pomeloorbit\s+)?"' + re.escape(BASE) + r'"\n',
            "",
            text,
        )
        if text2 != original:
            path.write_text(text2, encoding="utf-8")
            return True
        return False

    # Replace type refs with domain aliases
    def replace_ref(m: re.Match[str]) -> str:
        name = m.group(1)
        domain = type_to_domain[name]
        return f"{DOMAIN_ALIAS[domain]}.{name}"

    text = TYPE_REF_RE.sub(replace_ref, text)

    # Build import lines
    import_lines = []
    for domain in sorted(domains_needed, key=lambda d: DOMAIN_ALIAS[d]):
        alias = DOMAIN_ALIAS[domain]
        import_lines.append(f'\t{alias} "{BASE}/{domain}"')
    import_block = "\n".join(import_lines)

    if FLAT_IMPORT_RE.search(text):
        text = FLAT_IMPORT_RE.sub(import_block, text, count=1)
        # remove any remaining flat imports
        text = re.sub(
            r'(?m)^\t(?:pomeloorbit\s+)?"' + re.escape(BASE) + r'"\n',
            "",
            text,
        )
    elif f'"{BASE}"' in text:
        # bare import without alias
        text = re.sub(
            r'(?m)^\t"' + re.escape(BASE) + r'"\s*$',
            import_block,
            text,
            count=1,
        )
        text = re.sub(
            r'(?m)^\t"' + re.escape(BASE) + r'"\n',
            "",
            text,
        )
    else:
        # no import line but refs changed — inject into import block
        m = re.search(r"(?m)^import \(\n", text)
        if not m:
            raise SystemExit(f"{path}: cannot find import block to inject domain imports")
        insert_at = m.end()
        text = text[:insert_at] + import_block + "\n" + text[insert_at:]

    if text != original:
        path.write_text(text, encoding="utf-8")
        return True
    return False


def main() -> int:
    type_to_domain = build_type_map()
    print(f"mapped {len(type_to_domain)} types across {len(DOMAIN_ALIAS)} domains")

    changed = 0
    scanned = 0
    for path in sorted((ROOT / "internal").rglob("*.go")):
        if "internal/gen/" in path.as_posix() or "\\internal\\gen\\" in str(path):
            continue
        scanned += 1
        content = path.read_text(encoding="utf-8")
        if BASE not in content and "pomeloorbit." not in content:
            continue
        if rewrite_file(path, type_to_domain):
            changed += 1
            print(f"updated {path.relative_to(ROOT)}")

    # also cmd if any
    for path in sorted((ROOT / "cmd").rglob("*.go")) if (ROOT / "cmd").exists() else []:
        content = path.read_text(encoding="utf-8")
        if BASE not in content and "pomeloorbit." not in content:
            continue
        if rewrite_file(path, type_to_domain):
            changed += 1
            print(f"updated {path.relative_to(ROOT)}")

    print(f"done: scanned={scanned} changed={changed}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
