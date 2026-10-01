#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
source ./scripts/release-config.sh
export GOTOOLCHAIN="$RELEASE_GO_TOOLCHAIN"
source ./scripts/cli-shared/release-core.sh
cli_release_preflight "$@"
make verify

if [[ "${RUN_REAL_PROVIDER_SMOKE:-0}" == "1" ]]; then
  echo "[release-check] running opt-in real provider smoke test"
  make smoke-real-provider
else
  echo "[release-check] skipping real provider smoke test (set RUN_REAL_PROVIDER_SMOKE=1 to enable)"
fi

cli_release_build_check "$version"
