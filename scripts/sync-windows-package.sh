#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
version=$(cat "$repo_root/VERSION")
source_dir="$repo_root/dist/package/pomelo-orbit-v${version}-windows-x64/"
target_dir="$repo_root/dist/pomelo-orbit/"

is_runtime_path() {
	case "$1" in
	.env.release|data|logs) return 0 ;;
	esac
	return 1
}

if [ ! -d "$source_dir" ]; then
	printf 'source package does not exist: %s\n' "$source_dir" >&2
	exit 1
fi

mkdir -p "$target_dir"
if command -v rsync >/dev/null 2>&1; then
	rsync -av --delete --exclude='/.env.release' --exclude='/data/' --exclude='/logs/' "$source_dir" "$target_dir"
else
	for target_path in "$target_dir"* "$target_dir".[!.]*; do
		[ -e "$target_path" ] || continue
		name=${target_path##*/}
		is_runtime_path "$name" && continue
		[ -e "$source_dir/$name" ] || rm -rf "$target_path"
	done

	for source_path in "$source_dir"* "$source_dir".[!.]*; do
		[ -e "$source_path" ] || continue
		name=${source_path##*/}
		is_runtime_path "$name" && continue
		[ -d "$source_path" ] && rm -rf "$target_dir/$name"
		cp -a "$source_path" "$target_dir"
	done
fi

if [ ! -e "$target_dir/.env.release" ]; then
	cp -a "$source_dir/.env.release" "$target_dir/.env.release"
fi
