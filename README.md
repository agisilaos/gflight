# gflight

`gflight` is a macOS-focused CLI to search Google Flights results, create price watches, and notify via terminal and email.

## Install

```bash
go build ./cmd/gflight
```

## Usage

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
gflight doctor --json
gflight doctor --strict
gflight completion zsh > ~/.zsh/completions/_gflight
```

## Watch Commands

- `gflight watch create ...` create a saved watch.
  - `--plain` output: `watch_id=<id>`
  - Supports `--notify-webhook` and optional `--webhook-url`.
- `gflight watch list` list existing watches.
  - `--plain` output header: `id	name	enabled	target_price	from	to	depart`
- `gflight watch enable --id <watch-id>` enable a watch.
  - `--plain` output: `watch_id=<id>\tenabled=true`
- `gflight watch disable --id <watch-id>` disable a watch.
  - `--plain` output: `watch_id=<id>\tenabled=false`
- `gflight watch delete --id <watch-id> --force` delete a watch.
  - `--plain` output: `deleted_id=<id>`
  - Safety: requires `--force` or `--confirm <watch-id>`.
- `gflight watch run --all --once` executes selected watches and prints a summary.
  - Requires exactly one selector: `--all` or `--id <watch-id>`.
  - Exit behavior for provider failures:
    - default: exits `4` only when all evaluated provider requests fail
    - strict mode: `--fail-on-provider-errors` exits `4` on any provider failure
  - Human mode summary: `evaluated`, `triggered`, `provider_failures`, `notify_failures`.
  - JSON mode returns:
    - `evaluated`
    - `triggered`
    - `provider_failures`
    - `notify_failures`
    - `alerts` (triggered alert objects)

## Agent-Friendly Contract

- `--json` for deterministic structured output.
- `stdout` carries primary output; `stderr` carries diagnostics/alerts.
- `--no-input` avoids prompts.
- `--plain` emits stable line-based output for shell pipelines.
  - For mutation commands, plain output uses stable `key=value` fields.
- `--timeout` overrides provider request timeout per command (`search`, `watch run`).
- `doctor --json` provides preflight checks for provider auth, writable paths, and notification config.
- `doctor --strict` treats warnings as failures (CI/agent preflight mode).
- Query objects in JSON now use normalized `snake_case` keys (for example `query.from`, `query.depart`, `query.sort_by`).
- `gflight help <command>` provides command-specific help (for example `gflight help watch run`, `gflight help doctor`).
- Errors now include actionable `next:` hints on `stderr` when a known remediation exists.
- Unknown commands/subcommands include typo suggestions when a close match exists (for example `did you mean "watch"?`).

## Exit Codes

- `0` success
- `1` generic/runtime failure
- `2` invalid usage/validation
- `3` auth required/missing credentials
- `4` provider/upstream failure
- `6` notification delivery failure

## Config

Config path: `$XDG_CONFIG_HOME/gflight/config.json` (fallback `~/.config/gflight/config.json`)

State path: `$XDG_STATE_HOME/gflight` (fallback `~/.local/state/gflight`)

Supported config keys:

- `provider` (`serpapi` or `google-url`)
- `serp_api_key`
- `provider_timeout_seconds`
- `provider_retries`
- `provider_backoff_ms`
- `webhook_url`
- `smtp_host`
- `smtp_port`
- `smtp_user`
- `smtp_pass`
- `smtp_sender`
- `notify_email`

Related environment variables:

- `GFLIGHT_PROVIDER_TIMEOUT_SECONDS`
- `GFLIGHT_PROVIDER_RETRIES`
- `GFLIGHT_PROVIDER_BACKOFF_MS`
- `GFLIGHT_WEBHOOK_URL`

Notification channel test examples:

```bash
gflight notify test --channel terminal
gflight notify test --channel email --to you@example.com
gflight notify test --channel webhook --url https://example.com/hook
```

Webhook error hints:

- DNS issues are reported as `webhook dns lookup failed`.
- Timeouts are reported as `webhook timeout`.
- HTTP `429` is reported as `webhook endpoint rate limited`.
- HTTP `5xx` is reported as `webhook endpoint server error`.

## Architecture

- `internal/cli`: command handlers and CLI-facing validation/output.
- `internal/cli/watch_cmd_mutation.go`: watch create/list/enable/disable/delete command handlers.
- `internal/cli/watch_cmd_run.go`: watch run/test command handlers.
- `internal/cli/watch_service.go`: watch evaluation/selection/run logic (pure service helpers, unit-tested).
- `internal/cli/auth_service.go`: auth status + login mutation/validation helpers.
- `internal/cli/config_service.go`: config key get/set mutation/validation helpers.
- `internal/cli/config_validate.go`: shared runtime/docter config readiness validation.
- `internal/cli/notify_dispatcher.go`: notification abstraction boundary used by CLI orchestration.
- `internal/cli/errors.go`: centralized exit-code/error taxonomy mapping.
- `internal/cli/cli_integration_test.go`: table-driven CLI integration harness for agent flows.
- `internal/provider`: flight data providers (`serpapi`, `google-url`).
- `internal/notify`: terminal and SMTP notification delivery.
- `internal/watcher`: watch persistence store.

## Scheduled Runs (No Daemon)

Use your scheduler to run:

```bash
gflight --json watch run --all --once
```

### macOS launchd (recommended)

1. Build and choose stable absolute paths:

```bash
cd /Users/agis/projects/gflight
go build -o /Users/agis/bin/gflight ./cmd/gflight
```

2. Create `~/Library/LaunchAgents/com.agis.gflight.watch.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>com.agis.gflight.watch</string>

    <key>ProgramArguments</key>
    <array>
      <string>/bin/zsh</string>
      <string>-lc</string>
      <string>GFLIGHT_BIN=/Users/agis/bin/gflight GFLIGHT_CONFIG_HOME=/Users/agis/.config GFLIGHT_STATE_HOME=/Users/agis/.local/state /Users/agis/projects/gflight/scripts/run-watch-once.sh</string>
    </array>

    <key>StartInterval</key>
    <integer>900</integer>

    <key>RunAtLoad</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/tmp/gflight.watch.out.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/gflight.watch.err.log</string>
  </dict>
</plist>
```

3. Load and verify:

```bash
launchctl unload ~/Library/LaunchAgents/com.agis.gflight.watch.plist 2>/dev/null || true
launchctl load ~/Library/LaunchAgents/com.agis.gflight.watch.plist
launchctl list | grep gflight
tail -f /tmp/gflight.watch.out.log /tmp/gflight.watch.err.log
```

### cron (alternative)

Every 15 minutes:

```cron
*/15 * * * * GFLIGHT_BIN=/Users/agis/bin/gflight GFLIGHT_CONFIG_HOME=/Users/agis/.config GFLIGHT_STATE_HOME=/Users/agis/.local/state /Users/agis/projects/gflight/scripts/run-watch-once.sh >> /tmp/gflight.watch.cron.log 2>&1
```

## Release

1. `make release-check VERSION=vX.Y.Z`
2. `make release-dry-run VERSION=vX.Y.Z`
3. `make release VERSION=vX.Y.Z`

Release scripts:

- `scripts/release-check.sh`
- `scripts/release.sh`

## Docs

- `docs/README.md` overview of project docs.
