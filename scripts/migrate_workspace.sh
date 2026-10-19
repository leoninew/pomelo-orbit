#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
script_path=$script_dir/migrate_workspace.py
ORBIT_MIGRATION_CALLER_CWD=$PWD

case "$(uname -s)" in
	CYGWIN*|MSYS*|MINGW*)
		script_path=$(cygpath -am -- "$script_path")
		ORBIT_MIGRATION_CALLER_CWD=$(cygpath -am -- "$PWD")
		ORBIT_MIGRATION_CYGPATH=$(cygpath -am -- /usr/bin/cygpath.exe)
		export ORBIT_MIGRATION_CYGPATH
		# Python knows the selected target's OS and converts Windows paths there.
		# A Linux SSH destination such as /srv/orbit must remain literal here.
		MSYS2_ARG_CONV_EXCL='*'
		MSYS_NO_PATHCONV=1
		export MSYS2_ARG_CONV_EXCL MSYS_NO_PATHCONV
		;;
esac

export ORBIT_MIGRATION_CALLER_CWD
cd "$script_dir/.."
POMELO_ORBIT_APP__ENV=${POMELO_ORBIT_APP__ENV:-development}
export POMELO_ORBIT_APP__ENV
exec "${PYTHON:-python}" "$script_path" "$@"
