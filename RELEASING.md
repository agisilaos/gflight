# Releasing

## Go toolchain

Release checks, dry runs and publication select Go 1.27.1 through
`RELEASE_GO_TOOLCHAIN` in `scripts/release-config.sh`. Release CI uses the same
version. Go downloads and verifies it if needed. Ordinary verification retains
the caller's toolchain, subject to the Go 1.27.1 requirement in `go.mod`.

Releases are prepared by an agent, reviewed by a human, and published from a clean macOS checkout of the default branch.

## Local verification

Run `make verify` while developing. It checks the pinned local helper bundle,
module metadata, formatting, static analysis, tests, documentation and help.
Module validation leaves `go.mod` and `go.sum` unchanged, including on errors.
The real-provider smoke remains opt-in through `RUN_REAL_PROVIDER_SMOKE=1`.

## Prepare the changelog

Ask an agent to prepare `vX.Y.Z`. The agent must start from the repository evidence:

```bash
make changelog-context VERSION=vX.Y.Z
```

The agent updates only the new top section of `CHANGELOG.md` and must:

- describe user-visible outcomes rather than copy commit subjects;
- group related implementation commits into one useful bullet;
- use clear headings such as `Added`, `Changed`, `Fixed`, or `Removed` when they help;
- link every bullet to its verified merged GitHub pull request, or to a GitHub commit when no pull request exists;
- call out breaking changes explicitly;
- preserve all existing release sections.

Use commit messages, changed-file evidence, and PR metadata to understand impact. Never infer a PR association without evidence. Review the generated section, then commit it before running release checks.

## Validate and publish

Run these commands in order:

```bash
make release-check VERSION=vX.Y.Z
make release-dry-run VERSION=vX.Y.Z
make release VERSION=vX.Y.Z
```

`release-check` validates the clean worktree, version and changelog, runs `make verify`, then checks the version-stamped binary. `release-dry-run` builds both macOS archives and checksums, extracts the approved changelog section as release notes, and renders and syntax-checks the Homebrew formula without remote writes. Publication requires the `main` branch and validates the selected existing tap branch before creating a tag.

The final command creates and pushes the tag, publishes the GitHub Release with the approved changelog section, and updates the configured Homebrew tap.

If publication stops, follow [release recovery](docs/release-recovery.md) using the retained original artifacts and reported phase outcomes. Inspect remote state before attempting a missing step.

## Changelog policy

- Keep concrete release headings in the form `## [vX.Y.Z] - YYYY-MM-DD`.
- Do not add an `Unreleased` section.
- Treat the reviewed changelog section as the source of truth for GitHub release notes.
