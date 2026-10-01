# gflight Docs

## What Lives Here

- Product and architecture notes for `gflight`.
- Workflow and maintenance docs that do not belong in the top-level `README.md`.

## Release Process

- Primary release runbook is in `README.md` under `## Release`.
- Release automation scripts:
  - `scripts/release-check.sh`
  - `scripts/release.sh`
  - `scripts/smoke-real-provider.sh` (opt-in network smoke)
- [Interrupted release recovery](release-recovery.md) explains phase reports and retained artifacts.
- `scripts/release-config.sh` owns Gflight's names, version stamps and bundle fingerprint.
- `scripts/cli-shared/` contains the pinned local copy of the common process helpers.

## Verification and help

- `make verify` checks module metadata, formatting, vet, tests, docs and help.
- `make update-help` explicitly refreshes the snapshots; validation never updates them.
- `scripts/help-snapshots.txt` records four public help topics:
  - `--help` → `docs/help/root.txt`
  - `help watch run` → `docs/help/watch-run.txt`
  - `help doctor` → `docs/help/doctor.txt`
  - `help completion` → `docs/help/completion.txt`

## Policy

- Do not use `## [Unreleased]` in docs.
- Use concrete released versions (for example `v0.3.0`).
