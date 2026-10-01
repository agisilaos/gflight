#!/usr/bin/env bash
# Repository-owned settings; shared helpers are pinned to this reviewed bundle.
CLI_NAME="gflight"
FORMULA_NAME="gflight"
ARTIFACT_NAME="gflight"
DEFAULT_BRANCH="main"
DEFAULT_HOMEBREW_DESC="gflight command-line tool"
DEFAULT_HOMEBREW_LICENSE="MIT"
DEFAULT_HOMEBREW_TEST_ARG="--version"
DEFAULT_FORMULA_PATH="Formula/gflight.rb"
DEFAULT_BUILD_PKG="./cmd/gflight"
RELEASE_LDFLAGS_TEMPLATE='-s -w -X main.version={{VERSION}} -X main.commit={{COMMIT}} -X main.date={{DATE}}'
RELEASE_VERSION_TEMPLATE='gflight {{VERSION}} (commit {{COMMIT}}, built {{DATE}})'
RELEASE_CGO_ENABLED=0
RELEASE_INCLUDE_LICENSE=0
CLI_TEMPLATE_FINGERPRINT="816f217b3a5c95477b24e3fb8ba1f3db5470654441b71b16e5385c9cb5b73290"
