#!/usr/bin/env python3
"""Rewrite frontend flat gen/proto imports to domain paths + type renames."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WEB_SRC = ROOT / "web" / "src"

# old entity file stem -> (domain, new stem)
ENTITY_MAP = {
    "application": ("application", "application"),
    "application_bundle": ("application", "application_bundle"),
    "version": ("application", "version"),  # Service* moved out; version still under application
    "auth": ("auth", "auth"),
    "common": ("common", "common"),
    "credential": ("credential", "credential"),
    "deployment": ("deployment", "deployment"),
    "environment": ("environment", "environment"),
    "gateway": ("gateway", "gateway"),
    "build_stage": ("pipeline", "pipeline_stage"),
    "snapshot": ("pipeline", "snapshot"),
    "template": ("pipeline", "template"),
    "artifact": ("pipeline_run", "artifact"),
    "pipeline_run": ("pipeline_run", "pipeline_run"),
    "pipeline_stage_run": ("pipeline_run", "pipeline_stage_run"),
    "project": ("project", "project"),
    "repository": ("repository", "repository"),
    "webhook": ("repository", "webhook"),
    "role": ("role", "role"),
    "route": ("route", "route"),
    "traefik": ("route", "traefik"),
    "service": ("service", "service"),
    "settings": ("settings", "settings"),
    "task": ("task", "task"),
    "user": ("user", "user"),
}

# Types that moved from version.ts to service.ts
SERVICE_TYPES = {
    "ServiceResp",
    "ServiceListResp",
    "ServicePaginatedResp",
}

TYPE_RENAMES = {
    "BuildStageCreateReq": "PipelineStageCreateReq",
    "BuildStageDuplicateReq": "PipelineStageDuplicateReq",
    "BuildStagePaginatedResp": "PipelineStagePaginatedResp",
    "BuildStageResp": "PipelineStageResp",
    "BuildStageUpdateReq": "PipelineStageUpdateReq",
    "StageRunResp": "PipelineStageRunResp",
}

IMPORT_RE = re.compile(
    r"""from\s+(['"])@/gen/proto/orbit/v1/([A-Za-z0-9_]+)\1"""
)
TYPE_IMPORT_BLOCK_RE = re.compile(
    r"""import\s+type\s*\{([^}]+)\}\s*from\s*(['"])@/gen/proto/orbit/v1/([A-Za-z0-9_]+)\2"""
)


def rewrite_content(text: str) -> str:
    # First: type-aware rewrite for Service* from version
    def rewrite_type_import(m: re.Match[str]) -> str:
        names_blob = m.group(1)
        quote = m.group(2)
        entity = m.group(3)
        names = [n.strip() for n in names_blob.split(",") if n.strip()]
        # rename types
        names = [TYPE_RENAMES.get(n, n) for n in names]

        if entity == "version":
            service_names = [n for n in names if n in SERVICE_TYPES]
            version_names = [n for n in names if n not in SERVICE_TYPES]
            parts = []
            if version_names:
                domain, stem = ENTITY_MAP["version"]
                parts.append(
                    f"import type {{ {', '.join(version_names)} }} from {quote}@/gen/proto/orbit/v1/{domain}/{stem}{quote}"
                )
            if service_names:
                parts.append(
                    f"import type {{ {', '.join(service_names)} }} from {quote}@/gen/proto/orbit/v1/service/service{quote}"
                )
            return "\n".join(parts)

        if entity == "pipeline_run":
            stage_run = [n for n in names if n == "PipelineStageRunResp"]
            run_names = [n for n in names if n != "PipelineStageRunResp"]
            # After rename StageRunResp -> PipelineStageRunResp
            parts = []
            if run_names:
                parts.append(
                    f"import type {{ {', '.join(run_names)} }} from {quote}@/gen/proto/orbit/v1/pipeline_run/pipeline_run{quote}"
                )
            if stage_run:
                parts.append(
                    f"import type {{ {', '.join(stage_run)} }} from {quote}@/gen/proto/orbit/v1/pipeline_run/pipeline_stage_run{quote}"
                )
            if parts:
                return "\n".join(parts)

        if entity not in ENTITY_MAP:
            raise SystemExit(f"unknown entity import: {entity}")
        domain, stem = ENTITY_MAP[entity]
        return f"import type {{ {', '.join(names)} }} from {quote}@/gen/proto/orbit/v1/{domain}/{stem}{quote}"

    text = TYPE_IMPORT_BLOCK_RE.sub(rewrite_type_import, text)

    # Non-type imports / remaining path-only (rare)
    def rewrite_path(m: re.Match[str]) -> str:
        quote = m.group(1)
        entity = m.group(2)
        if entity not in ENTITY_MAP:
            # already domain-prefixed path? skip if looks like domain
            return m.group(0)
        domain, stem = ENTITY_MAP[entity]
        return f"from {quote}@/gen/proto/orbit/v1/{domain}/{stem}{quote}"

    text = IMPORT_RE.sub(rewrite_path, text)

    # Standalone type renames in code (not only imports)
    for old, new in TYPE_RENAMES.items():
        text = re.sub(rf"\b{old}\b", new, text)

    return text


def main() -> int:
    changed = 0
    for path in sorted(WEB_SRC.rglob("*")):
        if path.suffix not in {".ts", ".vue"}:
            continue
        if "web/src/gen/" in path.as_posix() or "\\web\\src\\gen\\" in str(path):
            continue
        original = path.read_text(encoding="utf-8")
        if "@/gen/proto/orbit/v1/" not in original and "BuildStage" not in original and "StageRunResp" not in original:
            continue
        updated = rewrite_content(original)
        if updated != original:
            path.write_text(updated, encoding="utf-8", newline="\n")
            changed += 1
            print(f"updated {path.relative_to(ROOT)}")
    print(f"done changed={changed}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
