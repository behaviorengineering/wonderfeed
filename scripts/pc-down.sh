#!/usr/bin/env bash
# Stop the Wonderfeed provider stack without removing volumes.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PC_BIN="${PROCESS_COMPOSE_BIN:-process-compose}"

if command -v "$PC_BIN" >/dev/null 2>&1; then
  "$PC_BIN" down -f "$ROOT/process-compose.yaml" 2>/dev/null || true
fi

if [[ -x "$ROOT/bin/wonderfeed" ]]; then
  "$ROOT/bin/wonderfeed" provider down
else
  make -C "$ROOT" build
  "$ROOT/bin/wonderfeed" provider down
fi
