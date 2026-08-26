#!/usr/bin/env sh
set -eu

POMELO_ORBIT_APP__ENV=release
export POMELO_ORBIT_APP__ENV

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

tool_bin=${HOME:-}/.local/bin
if [ -d "$tool_bin" ]; then
	PATH=$tool_bin:$PATH
	export PATH
fi

if [ -f ./pomelo-orbit.exe ]; then
	server_path=$script_dir/pomelo-orbit.exe
else
	server_path=$script_dir/pomelo-orbit
fi

find_running_pids() {
	ps -eo pid=,args= | while IFS=' ' read -r pid command; do
		case "$command" in
			"$server_path"|"$server_path "*) printf '%s\n' "$pid" ;;
		esac
	done
}

stop_server() {
	pids=$(find_running_pids || true)
	if [ -z "$pids" ]; then
		printf '%s\n' "pomelo-orbit is not running from $server_path."
		return 0
	fi

	printf '%s\n' "Stopping pomelo-orbit PID(s): $pids"
	# Request a normal termination before falling back to SIGKILL.
	# shellcheck disable=SC2086
	kill $pids 2>/dev/null || true
	sleep 3

	pids=$(find_running_pids || true)
	if [ -n "$pids" ]; then
		printf '%s\n' "Forcing pomelo-orbit PID(s): $pids"
		# shellcheck disable=SC2086
		kill -KILL $pids 2>/dev/null || true
	fi

	pids=$(find_running_pids || true)
	if [ -n "$pids" ]; then
		printf '%s\n' "Failed to stop pomelo-orbit PID(s): $pids" >&2
		return 1
	fi

	printf '%s\n' 'pomelo-orbit stopped.'
}

start_server() {
	exec "$server_path" "$@"
}

case "${1:-start}" in
	start)
		if [ "${1:-}" = start ]; then shift; fi
		start_server "$@"
		;;
	stop)
		stop_server
		;;
	restart)
		stop_server
		start_server
		;;
	status)
		pids=$(find_running_pids || true)
		if [ -z "$pids" ]; then
			printf '%s\n' "pomelo-orbit is not running from $server_path."
			exit 3
		fi
		printf '%s\n' "pomelo-orbit is running from $server_path with PID(s): $pids"
		;;
	*)
		start_server "$@"
		;;
esac
