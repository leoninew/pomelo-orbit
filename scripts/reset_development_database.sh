#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")"
uv run --locked python reset_development_database.py "$@"
