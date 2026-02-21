#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

log() {
  echo "[smoke-real-provider] $*"
}

err() {
  echo "[smoke-real-provider] error: $*" >&2
  exit 1
}

if [[ "${GFLIGHT_SMOKE_REAL:-0}" != "1" ]]; then
  log "skipped (set GFLIGHT_SMOKE_REAL=1 to enable network smoke test)"
  exit 0
fi

require_env() {
  local key="$1"
  if [[ -z "${!key:-}" ]]; then
    err "missing required env var: ${key}"
  fi
}

require_env GFLIGHT_SERPAPI_KEY
require_env GFLIGHT_FROM
require_env GFLIGHT_TO
require_env GFLIGHT_DEPART

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT
export XDG_CONFIG_HOME="$tmp_root/config"
export XDG_STATE_HOME="$tmp_root/state"
mkdir -p "$XDG_CONFIG_HOME" "$XDG_STATE_HOME"

log "auth login (serpapi)"
go run ./cmd/gflight auth login --provider serpapi --serpapi-key "$GFLIGHT_SERPAPI_KEY" >/dev/null

log "search smoke"
search_json="$tmp_root/search.json"
go run ./cmd/gflight --json search \
  --from "$GFLIGHT_FROM" \
  --to "$GFLIGHT_TO" \
  --depart "$GFLIGHT_DEPART" >"$search_json"
grep -q '"flights"' "$search_json" || err "search output missing flights field"
grep -q '"url"' "$search_json" || err "search output missing url field"

log "watch create/run/delete smoke"
watch_json="$tmp_root/watch.json"
go run ./cmd/gflight --json watch create \
  --name smoke-watch \
  --from "$GFLIGHT_FROM" \
  --to "$GFLIGHT_TO" \
  --depart "$GFLIGHT_DEPART" >"$watch_json"
watch_id="$(sed -n 's/.*"id": "\([^"]*\)".*/\1/p' "$watch_json" | head -n1)"
if [[ -z "$watch_id" ]]; then
  err "failed to parse watch id from watch create output"
fi

go run ./cmd/gflight --json watch run --id "$watch_id" --once >/dev/null
go run ./cmd/gflight --json watch delete --id "$watch_id" --force >/dev/null

log "passed"
