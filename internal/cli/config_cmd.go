package cli

import (
	"fmt"

	"github.com/agisilaos/gflight/internal/config"
)

func (a App) cmdConfig(g globalFlags, args []string) error {
	if len(args) < 2 {
		return newExitError(ExitInvalidUsage, "usage: gflight config get <key> | gflight config set <key> <value>")
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	switch args[0] {
	case "get":
		if len(args) != 2 {
			return newExitError(ExitInvalidUsage, "usage: gflight config get <key>")
		}
		val, ok := configGet(cfg, args[1])
		if !ok {
			return newExitError(ExitInvalidUsage, "unknown key %q", args[1])
		}
		if g.JSON {
			return writeJSON(map[string]string{"key": args[1], "value": val})
		}
		if g.Plain {
			writePlainKV("key", args[1], "value", val)
			return nil
		}
		fmt.Println(val)
		return nil
	case "set":
		if len(args) != 3 {
			return newExitError(ExitInvalidUsage, "usage: gflight config set <key> <value>")
		}
		if err := configSet(&cfg, args[1], args[2]); err != nil {
			return newExitError(ExitInvalidUsage, "%v", err)
		}
		if err := config.Save(cfg); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		if g.Plain && !g.JSON {
			writePlainKV("ok", "true", "key", args[1])
			return nil
		}
		return writeMaybeJSON(g, map[string]string{"ok": "true", "key": args[1]})
	default:
		return newExitError(ExitInvalidUsage, "unknown config action %q", args[0])
	}
}
