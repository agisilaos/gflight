package cli

import (
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
	case "watch", "watch run":
		return watchRunHelpText()
	case "doctor":
		return doctorHelpText()
	default:
		return usageText()
	}
}

func watchRunHelpText() string {
	return `gflight watch run - Execute saved watch checks

USAGE:
  gflight watch run --all [--once] [--fail-on-provider-errors] [global flags]
  gflight watch run --id <watch-id> [--once] [--fail-on-provider-errors] [global flags]

RULES:
  - Exactly one selector is required: --all or --id
  - Default provider failure policy exits 4 only when all evaluated provider requests fail
  - --fail-on-provider-errors exits 4 on any provider failure

OUTPUT:
  - --json: emits summary object with evaluated/triggered/provider_failures/notify_failures/alerts
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
