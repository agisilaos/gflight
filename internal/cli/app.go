package cli

import (
	"fmt"
	"strings"
)

type App struct {
	Version string
}

type globalFlags struct {
	JSON     bool
	Plain    bool
	Quiet    bool
	Verbose  bool
	NoInput  bool
	NoColor  bool
	StateDir string
	Help     bool
	Version  bool
}

func NewApp(version string) App {
	return App{Version: version}
}

func (a App) Run(args []string) error {
	g, rest, err := parseGlobal(args)
	if err != nil {
		return err
	}
	if g.Help {
		return a.help(nil)
	}
	if g.Version {
		fmt.Println(a.Version)
		return nil
	}
	if len(rest) == 0 {
		return a.help(nil)
	}
	cmd := rest[0]
	argv := rest[1:]

	switch cmd {
	case "help", "-h", "--help":
		return a.help(nil)
	case "--version", "version":
		fmt.Println(a.Version)
		return nil
	case "search":
		return a.cmdSearch(g, argv)
	case "watch":
		return a.cmdWatch(g, argv)
	case "notify":
		return a.cmdNotify(g, argv)
	case "auth":
		return a.cmdAuth(g, argv)
	case "config":
		return a.cmdConfig(g, argv)
	default:
		return newExitError(ExitInvalidUsage, "unknown command %q\n\n%s", cmd, usageText())
	}
}

func usageText() string {
	return `gflight - Search Google Flights and run price alerts

USAGE:
  gflight [global flags] <command> [args]

COMMANDS:
  search             One-shot flight search
  watch create       Create a watch
  watch list         List watches
  watch enable       Enable a watch
  watch disable      Disable a watch
  watch delete       Delete a watch
  watch run          Execute watches and emit notifications
  watch test         Simulate a watch alert
  notify test        Test notification channels
  auth login         Store API key interactively
  auth status        Show auth/config status
  config get/set     Read/write config values

GLOBAL FLAGS:
  --json             JSON output
  --plain            Stable plain output
  -q, --quiet        Suppress non-essential text
  -v, --verbose      Extra diagnostics to stderr
  --no-input         Disable prompts
  --state-dir PATH   Override state directory
  --version          Print version
  -h, --help         Show help
`
}

func parseGlobal(args []string) (globalFlags, []string, error) {
	var g globalFlags
	for len(args) > 0 {
		a := args[0]
		switch a {
		case "-h", "--help":
			g.Help = true
			args = args[1:]
		case "--version":
			g.Version = true
			args = args[1:]
		case "--json":
			g.JSON = true
			args = args[1:]
		case "--plain":
			g.Plain = true
			args = args[1:]
		case "-q", "--quiet":
			g.Quiet = true
			args = args[1:]
		case "-v", "--verbose":
			g.Verbose = true
			args = args[1:]
		case "--no-input":
			g.NoInput = true
			args = args[1:]
		case "--no-color":
			g.NoColor = true
			args = args[1:]
		case "--state-dir":
			if len(args) < 2 {
				return g, nil, newExitError(ExitInvalidUsage, "--state-dir requires a value")
			}
			g.StateDir = args[1]
			args = args[2:]
		default:
			if strings.HasPrefix(a, "-") {
				return g, nil, newExitError(ExitInvalidUsage, "unknown global flag %q", a)
			}
			return g, args, nil
		}
	}
	return g, args, nil
}
