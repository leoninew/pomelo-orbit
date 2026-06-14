#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_PID=""
FRONTEND_PID=""

cleanup() {
  trap - INT TERM EXIT

  if [[ -n "${FRONTEND_PID}" ]] && kill -0 "${FRONTEND_PID}" 2>/dev/null; then
    echo "停止前端服务器..."
    kill "${FRONTEND_PID}" 2>/dev/null || true
  fi

  if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
    echo "停止后端服务器..."
    kill "${BACKEND_PID}" 2>/dev/null || true
  fi

  wait 2>/dev/null || true
}

trap cleanup INT TERM EXIT

echo "启动后端服务器 (端口 9001)..."
(
  cd "${ROOT_DIR}/backend"
  PYTHONUTF8=1 uv run python -m pomelo_orbit.main --port 9001 --reload
) &
BACKEND_PID=$!

echo "启动前端服务器 (端口 9002)..."
(
  cd "${ROOT_DIR}/frontend"
  yarn dev
) &
FRONTEND_PID=$!

echo "前后端已启动，按 Ctrl+C 退出。"
wait -n "${BACKEND_PID}" "${FRONTEND_PID}"
