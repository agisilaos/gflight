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
		return writeMaybeJSON(g, map[string]string{"ok": "true", "key": args[1]})
	default:
		return newExitError(ExitInvalidUsage, "unknown config action %q", args[0])
	}
}

func configGet(cfg config.Config, key string) (string, bool) {
	switch key {
	case "provider":
		return cfg.Provider, true
	case "serp_api_key":
		if cfg.SerpAPIKey == "" {
			return "", true
		}
		return "***", true
	case "smtp_host":
		return cfg.SMTPHost, true
	case "smtp_user":
		return cfg.SMTPUsername, true
	case "smtp_sender":
		return cfg.SMTPSender, true
	case "notify_email":
		return cfg.DefaultNotifyEmail, true
	default:
		return "", false
	}
}

func configSet(cfg *config.Config, key, value string) error {
	switch key {
	case "provider":
		cfg.Provider = value
	case "serp_api_key":
		cfg.SerpAPIKey = value
	case "smtp_host":
		cfg.SMTPHost = value
	case "smtp_port":
		var p int
		_, err := fmt.Sscanf(value, "%d", &p)
		if err != nil || p <= 0 {
			return fmt.Errorf("smtp_port must be positive integer")
		}
		cfg.SMTPPort = p
	case "smtp_user":
		cfg.SMTPUsername = value
	case "smtp_pass":
		cfg.SMTPPassword = value
	case "smtp_sender":
		cfg.SMTPSender = value
	case "notify_email":
		cfg.DefaultNotifyEmail = value
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}
