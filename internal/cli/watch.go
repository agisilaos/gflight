package cli

import (
	"path/filepath"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/watcher"
)

func (a App) watcherStore(stateOverride string) (watcher.Store, error) {
	dir, err := config.StateDir(stateOverride)
	if err != nil {
		return watcher.Store{}, err
	}
	return watcher.Store{Path: filepath.Join(dir, "watches.json")}, nil
}

func (a App) cmdWatch(g globalFlags, args []string) error {
	if len(args) == 0 {
		return newExitError(ExitInvalidUsage, "watch requires subcommand: create|list|enable|disable|delete|run|test")
	}
	sub := args[0]
	argv := args[1:]
	switch sub {
	case "create":
		return a.cmdWatchCreate(g, argv)
	case "list":
		return a.cmdWatchList(g, argv)
	case "enable":
		return a.cmdWatchSetEnabled(g, argv, true)
	case "disable":
		return a.cmdWatchSetEnabled(g, argv, false)
	case "delete":
		return a.cmdWatchDelete(g, argv)
	case "run":
		return a.cmdWatchRun(g, argv)
	case "test":
		return a.cmdWatchTest(g, argv)
	default:
		if s := suggestClosest(sub, []string{"create", "list", "enable", "disable", "delete", "run", "test"}); s != "" {
			return newExitError(ExitInvalidUsage, "unknown watch subcommand %q (did you mean %q?)", sub, s)
		}
		return newExitError(ExitInvalidUsage, "unknown watch subcommand %q", sub)
	}
}
