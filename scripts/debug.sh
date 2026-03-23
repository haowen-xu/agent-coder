#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_PATH="${1:-config.yaml}"
LOG_DIR="${ROOT_DIR}/.debug"
SERVER_LOG="${LOG_DIR}/server.log"
WORKER_LOG="${LOG_DIR}/worker.log"

if ! command -v go >/dev/null 2>&1; then
  echo "[debug] missing command: go" >&2
  exit 1
fi
if ! command -v pnpm >/dev/null 2>&1; then
  echo "[debug] missing command: pnpm" >&2
  exit 1
fi

if [[ ! -f "${ROOT_DIR}/${CONFIG_PATH}" && ! -f "${CONFIG_PATH}" ]]; then
  echo "[debug] config file not found: ${CONFIG_PATH}" >&2
  exit 1
fi

if [[ -f "${CONFIG_PATH}" ]]; then
  CONFIG_FILE="${CONFIG_PATH}"
else
  CONFIG_FILE="${ROOT_DIR}/${CONFIG_PATH}"
fi

PORT="$(awk '
  /^server:/ {in_server=1; next}
  in_server && /^[^[:space:]]/ {in_server=0}
  in_server && $1=="port:" {print $2; exit}
' "${CONFIG_FILE}")"
if [[ -z "${PORT}" ]]; then
  PORT="25790"
fi

mkdir -p "${LOG_DIR}"
: > "${SERVER_LOG}"
: > "${WORKER_LOG}"

cleanup() {
  local code=$?
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${WORKER_PID:-}" ]] && kill -0 "${WORKER_PID}" >/dev/null 2>&1; then
    kill "${WORKER_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${TAIL_PID:-}" ]] && kill -0 "${TAIL_PID}" >/dev/null 2>&1; then
    kill "${TAIL_PID}" >/dev/null 2>&1 || true
  fi
  wait >/dev/null 2>&1 || true
  exit "${code}"
}
trap cleanup INT TERM EXIT

echo "[debug] building webui static assets"
(
  cd "${ROOT_DIR}/webui"
  pnpm build
)

echo "[debug] starting server with config: ${CONFIG_FILE}"
(
  cd "${ROOT_DIR}"
  go run ./cmds server --config "${CONFIG_FILE}"
) >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

echo "[debug] starting worker with config: ${CONFIG_FILE}"
(
  cd "${ROOT_DIR}"
  go run ./cmds worker --config "${CONFIG_FILE}"
) >"${WORKER_LOG}" 2>&1 &
WORKER_PID=$!

echo "[debug] server: http://127.0.0.1:${PORT}"
echo "[debug] logs: ${SERVER_LOG} | ${WORKER_LOG}"
echo "[debug] press Ctrl+C to stop"

tail -n 40 -f "${SERVER_LOG}" "${WORKER_LOG}" &
TAIL_PID=$!

while true; do
  if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    echo "[debug] server exited, stopping debug environment" >&2
    break
  fi
  if ! kill -0 "${WORKER_PID}" >/dev/null 2>&1; then
    echo "[debug] worker exited, stopping debug environment" >&2
    break
  fi
  sleep 1
done
