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

## Policy

- Do not use `## [Unreleased]` in docs.
- Use concrete released versions (for example `v0.3.0`).
