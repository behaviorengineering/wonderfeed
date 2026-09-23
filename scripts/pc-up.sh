#!/usr/bin/env bash
# Start the Wonderfeed provider stack under process-compose (after a timed Docker probe).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROBE_SECONDS="${WONDERFEED_DOCKER_PROBE_SECONDS:-3}"
PC_BIN="${PROCESS_COMPOSE_BIN:-process-compose}"

docker_ok() {
  if command -v timeout >/dev/null 2>&1; then
    timeout "${PROBE_SECONDS}" docker info >/dev/null 2>&1
    return $?
  fi
  if command -v gtimeout >/dev/null 2>&1; then
    gtimeout "${PROBE_SECONDS}" docker info >/dev/null 2>&1
    return $?
  fi
  docker info >/dev/null 2>&1 &
  local pid=$!
  local i=0
  while kill -0 "$pid" 2>/dev/null; do
    if (( i >= PROBE_SECONDS )); then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      return 1
    fi
    sleep 1
    i=$((i + 1))
  done
  wait "$pid"
}

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker CLI not found. Install Docker Desktop (or Engine) and retry." >&2
  exit 1
fi

if ! docker_ok; then
  echo "Docker daemon did not answer within ${PROBE_SECONDS}s." >&2
  echo "The provider stack requires Docker for PostgreSQL and YT Zero." >&2
  exit 1
fi

if ! command -v "$PC_BIN" >/dev/null 2>&1; then
  echo "process-compose not found on PATH (looked for: $PC_BIN)." >&2
  echo "Install process-compose or set PROCESS_COMPOSE_BIN." >&2
  echo "Detached alternative: make build && ./bin/wonderfeed provider up" >&2
  exit 1
fi

make -C "$ROOT" build
"$ROOT/bin/wonderfeed" provider prepare

export ROOT
exec "$PC_BIN" up -f "$ROOT/process-compose.yaml"
