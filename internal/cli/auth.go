package cli

import (
	"flag"
	"os"

	"github.com/agisilaos/gflight/internal/config"
)

func (a App) cmdAuth(g globalFlags, args []string) error {
	if len(args) == 0 {
		return newExitError(ExitInvalidUsage, "auth requires subcommand: login|status")
	}
	switch args[0] {
	case "status":
		cfg, err := config.Load()
		if err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		status := map[string]any{
			"provider":        cfg.Provider,
			"serpapi_key":     cfg.SerpAPIKey != "",
			"smtp_configured": cfg.SMTPHost != "" && cfg.SMTPUsername != "" && cfg.SMTPPassword != "" && cfg.SMTPSender != "",
		}
		return writeMaybeJSON(g, status)
	case "login":
		cfg, err := config.Load()
		if err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		apiKey := fs.String("serpapi-key", "", "SerpAPI key")
		providerName := fs.String("provider", "", "Provider: serpapi|google-url")
		if err := fs.Parse(args[1:]); err != nil {
			return newExitError(ExitInvalidUsage, "%v", err)
		}
		if *providerName != "" {
			cfg.Provider = *providerName
		}
		if *apiKey != "" {
			cfg.SerpAPIKey = *apiKey
		}
		if err := config.Save(cfg); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "provider": cfg.Provider})
	default:
		return newExitError(ExitInvalidUsage, "unknown auth subcommand %q", args[0])
	}
}
