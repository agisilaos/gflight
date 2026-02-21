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
		status := authStatus(cfg)
		if g.Plain && !g.JSON {
			writePlainKV(
				"provider", cfg.Provider,
				"serpapi_key", boolToPlain(status["serpapi_key"]),
				"smtp_configured", boolToPlain(status["smtp_configured"]),
				"webhook_configured", boolToPlain(status["webhook_configured"]),
			)
			return nil
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
		if err := applyAuthLogin(&cfg, *providerName, *apiKey); err != nil {
			return newExitError(ExitInvalidUsage, "%v", err)
		}
		if err := config.Save(cfg); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		if g.Plain && !g.JSON {
			writePlainKV("ok", "true", "provider", cfg.Provider)
			return nil
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "provider": cfg.Provider})
	default:
		if s := suggestClosest(args[0], []string{"login", "status"}); s != "" {
			return newExitError(ExitInvalidUsage, "unknown auth subcommand %q (did you mean %q?)", args[0], s)
		}
		return newExitError(ExitInvalidUsage, "unknown auth subcommand %q", args[0])
	}
}
