#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   GFLIGHT_BIN=/absolute/path/to/gflight \
#   GFLIGHT_CONFIG_HOME=/absolute/path/config \
#   GFLIGHT_STATE_HOME=/absolute/path/state \
#   ./scripts/run-watch-once.sh

if [[ -z "${GFLIGHT_BIN:-}" ]]; then
  echo "error: GFLIGHT_BIN is required (absolute path to gflight binary)" >&2
  exit 2
fi

if [[ ! -x "$GFLIGHT_BIN" ]]; then
  echo "error: GFLIGHT_BIN is not executable: $GFLIGHT_BIN" >&2
  exit 2
fi

if [[ -n "${GFLIGHT_CONFIG_HOME:-}" ]]; then
  export XDG_CONFIG_HOME="$GFLIGHT_CONFIG_HOME"
fi

if [[ -n "${GFLIGHT_STATE_HOME:-}" ]]; then
  export XDG_STATE_HOME="$GFLIGHT_STATE_HOME"
fi

echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] running gflight watch pass"
"$GFLIGHT_BIN" --json watch run --all --once

