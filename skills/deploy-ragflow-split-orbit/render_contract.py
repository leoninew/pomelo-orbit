"""Render and check generated artifacts for the split RAGFlow contract."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Any

SKILLS_DIRECTORY = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SKILLS_DIRECTORY))

from _ragflow.contract_renderer import (  # noqa: E402
    ContractError,
    check_outputs,
    load_contract,
    render_all,
    write_outputs,
)


CONTRACT_PATH = Path(__file__).with_name("deployment-contract.json")
RENDERER_PATH = "skills/deploy-ragflow-split-orbit/render_contract.py"


def render_current_contract() -> tuple[dict[str, Any], dict[Path, str]]:
    contract = load_contract(CONTRACT_PATH)
    return contract, render_all(contract, RENDERER_PATH)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--write", action="store_true", help="write split generated artifacts")
    action.add_argument("--check", action="store_true", help="fail when a split generated artifact has drifted")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        _, rendered = render_current_contract()
    except ContractError as error:
        print(f"split deployment contract error: {error}", file=sys.stderr)
        return 2

    if args.write:
        write_outputs(rendered)
        return 0

    drifted = check_outputs(rendered)
    if drifted:
        print("generated split RAGFlow artifacts have drifted:", file=sys.stderr)
        for path in drifted:
            print(f"  {path.as_posix()}", file=sys.stderr)
        print(f"run: python {RENDERER_PATH} --write", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
