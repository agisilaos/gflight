package cli

import (
	"fmt"
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
	Timeout  string
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
		return a.help(rest)
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
		return a.help(argv)
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
	case "completion":
		return a.cmdCompletion(g, argv)
	case "doctor":
		return a.cmdDoctor(g, argv)
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
  completion         Generate shell completion script
  doctor             Run automation preflight checks

GLOBAL FLAGS:
  --json             JSON output
  --plain            Stable plain output
  -q, --quiet        Suppress non-essential text
  -v, --verbose      Extra diagnostics to stderr
  --no-input         Disable prompts
  --timeout DUR      Provider timeout override (e.g. 10s)
  --state-dir PATH   Override state directory
  --version          Print version
  -h, --help         Show help
`
}

func parseGlobal(args []string) (globalFlags, []string, error) {
	var g globalFlags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			g.Help = true
		case "--version":
			g.Version = true
		case "--json":
			g.JSON = true
		case "--plain":
			g.Plain = true
		case "-q", "--quiet":
			g.Quiet = true
		case "-v", "--verbose":
			g.Verbose = true
		case "--no-input":
			g.NoInput = true
		case "--no-color":
			g.NoColor = true
		case "--state-dir":
			if i+1 >= len(args) {
				return g, nil, newExitError(ExitInvalidUsage, "--state-dir requires a value")
			}
			i++
			g.StateDir = args[i]
		case "--timeout":
			if i+1 >= len(args) {
				return g, nil, newExitError(ExitInvalidUsage, "--timeout requires a value like 10s")
			}
			i++
			g.Timeout = args[i]
		default:
			rest = append(rest, a)
		}
	}
	return g, rest, nil
}
