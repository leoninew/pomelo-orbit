#!/usr/bin/env python3
"""Walk git history from the first commit and derive x.y.z.

Rules (x is fixed at 0):
  * y, z start at 0
  * a commit whose subject starts with "feat"  -> y += 1, z = 0
  * any other commit                          -> z += 1
  * print one line every time y or z changes:
        <commit-date>  <sha8>  <subject-first-50-chars>  <x>.<y>.<z>

After calculation, optionally apply the final version to:
  * VERSION                         (package version source)
  * configs/config.yaml             (app.version)
  * .env.example                    (app version example)
  * web/package.json                ("version" field)

When applying from a clean worktree, --apply creates the matching lightweight
Git tag on the current HEAD. A dirty worktree still receives the version-file
updates, but skips tag creation with a warning.

--apply-amend is the release-commit workflow: it requires an unpushed HEAD,
writes and stages the version metadata, amends HEAD without changing its
message, then creates the matching lightweight Git tag.

Run from any directory inside the target git repository:

    uv run --project scripts python scripts/version-calc.py
    uv run --project scripts python scripts/version-calc.py --apply
    uv run --project scripts python scripts/version-calc.py --quiet --apply
    uv run --project scripts python scripts/version-calc.py --quiet --apply-amend
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterator, Tuple

REPO_ROOT = Path(__file__).resolve().parent.parent
VERSION_FILE = REPO_ROOT / "VERSION"
CONFIG_FILE = REPO_ROOT / "configs" / "config.yaml"
ENV_EXAMPLE_FILE = REPO_ROOT / ".env.example"
PACKAGE_JSON = REPO_ROOT / "web" / "package.json"

APP_VERSION_RE = re.compile(
    rb"^(app:\r?\n(?:^[ \t]+[^\r\n]*\r?\n)*?^[ \t]+version:[ \t]*)([^\s#\r\n]+)",
    re.MULTILINE,
)
PACKAGE_JSON_VERSION_RE = re.compile(
    rb'^(\s*"version"\s*:\s*")([^"]*)(")',
    re.MULTILINE,
)
ENV_APP_VERSION_RE = re.compile(
    rb"^(#\s*POMELO_ORBIT_APP__VERSION=)([^\r\n]*)",
    re.MULTILINE,
)


def version_files() -> Tuple[Path, ...]:
    return (VERSION_FILE, CONFIG_FILE, ENV_EXAMPLE_FILE, PACKAGE_JSON)


def run_git(*args: str) -> str:
    proc = subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        text=True,
    )
    return proc.stdout


def is_feature(subject: str) -> bool:
    return subject.lstrip().lower().startswith("feat")


def iter_commits() -> Iterator[Tuple[str, str, str]]:
    """Yield (full_hash, iso_date, subject) from oldest to newest."""
    log = run_git(
        "log",
        "--reverse",
        "--pretty=format:%H%x1f%aI%x1f%s",
    )
    for line in log.splitlines():
        parts = line.split("\x1f", 2)
        if len(parts) != 3:
            continue
        full_hash, date, subject = parts
        yield full_hash, date, subject


def calculate_version(*, print_history: bool = True) -> str:
    """Walk history and return final x.y.z. Optionally print each step."""
    x = 0
    y = 0
    z = 0
    saw_commit = False

    for full_hash, date, subject in iter_commits():
        saw_commit = True
        short = full_hash[:8]
        if is_feature(subject):
            y += 1
            z = 0
        else:
            z += 1
        if print_history:
            headline = subject.split("\n", 1)[0][:50]
            print(f"{date}  {short}  {headline}  {x}.{y}.{z}")

    if not saw_commit:
        raise RuntimeError("no commits found; cannot derive version")

    return f"{x}.{y}.{z}"


def apply_version(version: str) -> None:
    """Write the release version to every tracked consumer."""
    previous = (
        VERSION_FILE.read_text(encoding="utf-8").strip()
        if VERSION_FILE.exists()
        else ""
    )
    replacements = (
        _prepare_version_replacement(
            CONFIG_FILE, APP_VERSION_RE, version, "app.version"
        ),
        _prepare_version_replacement(
            ENV_EXAMPLE_FILE,
            ENV_APP_VERSION_RE,
            version,
            "app version example",
        ),
        _prepare_version_replacement(
            PACKAGE_JSON, PACKAGE_JSON_VERSION_RE, version, "package version"
        ),
    )

    # Read and validate every target before changing any of them.
    VERSION_FILE.write_text(version + "\n", encoding="utf-8")
    print(f"updated {VERSION_FILE.relative_to(REPO_ROOT)}: {previous} -> {version}")
    for path, previous, updated, label in replacements:
        if updated == path.read_bytes():
            print(f"unchanged {path.relative_to(REPO_ROOT)} -> {label} = {version!r}")
            continue
        path.write_bytes(updated)
        print(
            f"updated {path.relative_to(REPO_ROOT)} -> "
            f"{label} {previous!r} -> {version!r}"
        )


def is_worktree_clean() -> bool:
    """Return whether the index and worktree, including untracked files, are clean."""
    return not run_git("status", "--porcelain=v1", "--untracked-files=all").strip()


def require_unpushed_head() -> None:
    """Require HEAD to be ahead of its configured upstream branch."""
    try:
        upstream = run_git(
            "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"
        ).strip()
    except subprocess.CalledProcessError as exc:
        raise RuntimeError(
            "--apply-amend requires the current branch to track an upstream"
        ) from exc
    if not upstream:
        raise RuntimeError(
            "--apply-amend requires the current branch to track an upstream"
        )
    ahead = int(run_git("rev-list", "--count", f"{upstream}..HEAD").strip())
    if ahead == 0:
        raise RuntimeError(
            f"HEAD is already present on upstream {upstream}; --apply-amend requires an unpushed HEAD"
        )
    print(f"HEAD is {ahead} commit(s) ahead of {upstream}")


def create_version_tag(version: str) -> None:
    """Create the lightweight release tag for the current HEAD."""
    tag = f"v{version}"
    run_git("tag", tag)
    print(f"created tag: {tag}")


def version_tag_is_on_head(version: str) -> bool:
    """Reject conflicting tags and report whether the tag currently names HEAD."""
    tag = f"v{version}"
    if not run_git("tag", "--list", tag).strip():
        return False
    tag_head = run_git("rev-list", "-n", "1", tag).strip()
    head = run_git("rev-parse", "HEAD").strip()
    if tag_head != head:
        raise RuntimeError(
            f"release tag {tag} already points to {tag_head[:8]}, not {head[:8]}"
        )
    return True


def stage_version_files() -> bool:
    """Stage version metadata and report whether it changed the index."""
    paths = tuple(path.relative_to(REPO_ROOT).as_posix() for path in version_files())
    run_git("add", "--", *paths)
    return bool(run_git("diff", "--cached", "--name-only", "--", *paths).strip())


def amend_head_with_version() -> None:
    """Amend only version metadata while retaining HEAD's message and parent."""
    paths = tuple(path.relative_to(REPO_ROOT).as_posix() for path in version_files())
    run_git("commit", "--amend", "--no-edit", "--only", "--", *paths)
    print("amended HEAD with version metadata")


def move_version_tag_to_head(version: str) -> None:
    """Move an existing local release tag to the amended HEAD."""
    tag = f"v{version}"
    run_git("tag", "--force", tag)
    print(f"moved tag to amended HEAD: {tag}")


def is_version_tag_creation_needed(version: str) -> bool:
    """Reject conflicting release tags and skip tags already on the current HEAD."""
    tag = f"v{version}"
    if not run_git("tag", "--list", tag).strip():
        return True

    tag_head = run_git("rev-list", "-n", "1", tag).strip()
    head = run_git("rev-parse", "HEAD").strip()
    if tag_head != head:
        raise RuntimeError(
            f"release tag {tag} already points to {tag_head[:8]}, not {head[:8]}"
        )
    print(f"tag already exists on HEAD: {tag}")
    return False


def _prepare_version_replacement(
    path: Path,
    pattern: re.Pattern[bytes],
    version: str,
    label: str,
) -> Tuple[Path, str, bytes, str]:
    """Prepare one replacement while preserving the source encoding and newlines."""
    raw = path.read_bytes()
    match = pattern.search(raw)
    if match is None:
        raise RuntimeError(f"could not find {label} in {path}")
    previous = match.group(2).decode("utf-8")
    version_bytes = version.encode("utf-8")
    updated = raw[: match.start(2)] + version_bytes + raw[match.end(2) :]
    return path, previous, updated, label


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Derive x.y.z from git history")
    apply_group = parser.add_mutually_exclusive_group()
    apply_group.add_argument(
        "--apply",
        action="store_true",
        help="write final version and tag HEAD when the worktree is clean",
    )
    apply_group.add_argument(
        "--apply-amend",
        action="store_true",
        help="write, stage, and amend final version into an unpushed HEAD before tagging it",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="do not print per-commit history lines",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    version = calculate_version(print_history=not args.quiet)
    if not args.quiet:
        print()
    print(f"version: {version}")
    if args.apply:
        worktree_was_clean = is_worktree_clean()
        tag_creation_needed = worktree_was_clean and is_version_tag_creation_needed(
            version
        )
        apply_version(version)
        if tag_creation_needed:
            create_version_tag(version)
        else:
            if not worktree_was_clean:
                print(
                    "warning: worktree was not clean before --apply; "
                    f"skipped tag v{version}",
                    file=sys.stderr,
                )
    elif args.apply_amend:
        require_unpushed_head()
        tag_was_on_head = version_tag_is_on_head(version)
        apply_version(version)
        if stage_version_files():
            amend_head_with_version()
            if tag_was_on_head:
                move_version_tag_to_head(version)
            else:
                create_version_tag(version)
        else:
            print(
                "version metadata already matches the calculated version; skipped amend"
            )
            if tag_was_on_head:
                print(f"tag already exists on HEAD: v{version}")
            else:
                create_version_tag(version)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as exc:
        sys.stderr.write(f"git failed: {exc.stderr.strip() or exc}\n")
        sys.exit(1)
    except (OSError, RuntimeError, ValueError, re.error) as exc:
        sys.stderr.write(f"{exc}\n")
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
