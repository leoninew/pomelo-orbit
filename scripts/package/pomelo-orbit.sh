#!/usr/bin/env sh
set -eu

POMELO_ORBIT_APP__ENV=release
export POMELO_ORBIT_APP__ENV

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

if [ -f ./pomelo-orbit.exe ]; then
	exec ./pomelo-orbit.exe "$@"
fi
exec ./pomelo-orbit "$@"
