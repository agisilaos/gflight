package cli

import (
	"flag"
	"fmt"
	"strings"
)

func (a App) help(args []string) error {
	fmt.Print(helpText(args))
	return nil
}

func helpText(args []string) string {
	if len(args) == 0 {
		return usageText()
	}
	k := strings.ToLower(strings.Join(args, " "))
	switch k {
	case "search":
		fs, _ := newSearchFlagSet("search")
		return focusedFlagHelp(fs, "Search flight options", `Required: --from, --to, --depart (YYYY-MM-DD).
Optional --return must be on or after departure. Without it, the trip is one-way.
Search queries the configured provider; there is no search --dry-run flag.
Use --json or --plain for automation; see gflight --help for global flags.`, "gflight search --from SFO --to ATH --depart 2026-12-01")
	case "watch create":
		fs, _, _ := newWatchCreateFlagSet()
		return focusedFlagHelp(fs, "Save a price watch", `Required: --from, --to, --depart (YYYY-MM-DD).
Optional --return must be on or after departure. Without it, the trip is one-way.
Watches start enabled. Creation does not search or send notifications.
--dry-run reads configuration defaults and previews without saving a watch.
Disable terminal alerts explicitly with --notify-terminal=false.
Use --json for a structured preview; see gflight --help for global flags.`, "gflight watch create --from SFO --to ATH --depart 2026-12-01 --target-price 700 --dry-run")
	case "watch delete":
		fs, _ := newWatchDeleteFlagSet()
		return focusedFlagHelp(fs, "Delete a saved watch", `Required: --id plus either --force or --confirm with the same watch ID.
There is no interactive prompt or delete --dry-run flag.
--no-input requires --force even when --confirm matches.
Find IDs with gflight watch list. See gflight --help for global flags.`, "gflight watch delete --id w_123 --confirm w_123")
	case "watch", "watch run":
		return watchRunHelpText()
	case "completion":
		return completionHelpText()
	case "doctor":
		return doctorHelpText()
	default:
		return usageText()
	}
}

func watchRunHelpText() string {
	return `gflight watch run - Execute saved watch checks

USAGE:
  gflight watch run --all [--once] [--fail-on-provider-errors] [--retry-uncertain] [global flags]
  gflight watch run --id <watch-id> [--once] [--fail-on-provider-errors] [--retry-uncertain] [global flags]

RULES:
  - Exactly one selector is required: --all or --id
  - Default provider failure policy exits 4 only when all evaluated provider requests fail
  - --fail-on-provider-errors exits 4 on any provider failure
  - Pending delivery is retained per channel; successful channels are not resent
  - Inspect ambiguous delivery before --retry-uncertain (may duplicate notifications)

OUTPUT:
  - --json: emits summary object with evaluated/triggered/provider_failures/notify_failures/pending/uncertain/recovered/alerts
  - human: emits summary line and any alert notifications
`
}

func doctorHelpText() string {
	return `gflight doctor - Run preflight checks for automation readiness

USAGE:
  gflight doctor [--strict] [global flags]

CHECKS:
  - provider authentication readiness
  - config/state path writability
  - email/webhook notification readiness

BEHAVIOR:
  - default: warnings do not fail command
  - --strict: warnings are treated as failures
`
}

func completionHelpText() string {
	return `gflight completion - Generate shell completion script

USAGE:
  gflight completion <bash|zsh|fish>
  gflight completion path <bash|zsh|fish>

EXAMPLES:
  gflight completion zsh > ~/.zsh/completions/_gflight
  gflight completion bash > /usr/local/etc/bash_completion.d/gflight
  gflight completion fish > ~/.config/fish/completions/gflight.fish
  gflight completion path zsh
`
}

func focusedFlagHelp(fs *flag.FlagSet, summary, rules, example string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "gflight %s - %s\n\nUSAGE:\n  gflight %s [flags] [global flags]\n\nRULES:\n%s\n\nFLAGS:\n", fs.Name(), summary, fs.Name(), rules)
	fs.SetOutput(&out)
	fs.PrintDefaults()
	fmt.Fprintf(&out, "\nEXAMPLE:\n  %s\n", example)
	return out.String()
}
