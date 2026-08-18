#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
version=$(cat "$repo_root/VERSION")
source_dir="$repo_root/dist/package/pomelo-orbit-v${version}-windows-x64/"
target_dir="$repo_root/dist/package/pomelo-orbit-v${version}/"

if [ ! -d "$source_dir" ]; then
	printf 'source package does not exist: %s\n' "$source_dir" >&2
	exit 1
fi

mkdir -p "$target_dir"
if command -v rsync >/dev/null 2>&1; then
	rsync -av --exclude='/.env' "$source_dir" "$target_dir"
	exit 0
fi

for source_path in "$source_dir"* "$source_dir".[!.]*; do
	[ -e "$source_path" ] || continue
	[ "${source_path##*/}" = ".env" ] && continue
	cp -a "$source_path" "$target_dir"
done
