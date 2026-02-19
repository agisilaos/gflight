# gflight

`gflight` is a macOS-focused CLI to search Google Flights results, create price watches, and notify via terminal and email.

## Install / Build

```bash
go build ./cmd/gflight
```

## Quick Start

1. Configure provider auth (SerpAPI key for Google Flights data):

```bash
gflight auth login --provider serpapi --serpapi-key "$GFLIGHT_SERPAPI_KEY"
```

2. One-shot search:

```bash
gflight search --from SFO --to ATH --depart 2026-06-10 --return 2026-06-24 --json
```

3. Create a watch and run it:

```bash
gflight watch create --name summer-athens --from SFO --to ATH --depart 2026-06-10 --return 2026-06-24 --target-price 700 --notify-terminal --notify-email --email-to you@example.com
gflight watch list
gflight watch disable --id w_123
gflight watch enable --id w_123
gflight watch delete --id w_123 --force
gflight watch run --all --once
```

## Watch Commands

- `gflight watch create ...` create a saved watch.
- `gflight watch list` list existing watches.
- `gflight watch enable --id <watch-id>` enable a watch.
- `gflight watch disable --id <watch-id>` disable a watch.
- `gflight watch delete --id <watch-id> --force` delete a watch.
  - Safety: requires `--force` or `--confirm <watch-id>`.

## Agent-Friendly Contract

- `--json` for deterministic structured output.
- `stdout` carries primary output; `stderr` carries diagnostics.
- `--no-input` avoids prompts.
- `--plain` emits stable line-based output for shell pipelines.

## Config

Config path: `$XDG_CONFIG_HOME/gflight/config.json` (fallback `~/.config/gflight/config.json`)

State path: `$XDG_STATE_HOME/gflight` (fallback `~/.local/state/gflight`)

Supported config keys:

- `provider` (`serpapi` or `google-url`)
- `serp_api_key`
- `smtp_host`
- `smtp_port`
- `smtp_user`
- `smtp_pass`
- `smtp_sender`
- `notify_email`

## Release

1. `make release-check VERSION=vX.Y.Z`
2. `make release-dry-run VERSION=vX.Y.Z`
3. `make release VERSION=vX.Y.Z`
