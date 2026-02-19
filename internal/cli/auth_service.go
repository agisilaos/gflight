package cli

import (
	"fmt"
	"strings"

	"github.com/agisilaos/gflight/internal/config"
)

func authStatus(cfg config.Config) map[string]any {
	return map[string]any{
		"provider":        cfg.Provider,
		"serpapi_key":     cfg.SerpAPIKey != "",
		"smtp_configured": cfg.SMTPHost != "" && cfg.SMTPUsername != "" && cfg.SMTPPassword != "" && cfg.SMTPSender != "",
	}
}

func applyAuthLogin(cfg *config.Config, providerName, apiKey string) error {
	if providerName != "" {
		normalized, err := normalizeProvider(providerName)
		if err != nil {
			return err
		}
		cfg.Provider = normalized
	}
	if apiKey != "" {
		cfg.SerpAPIKey = apiKey
	}
	return nil
}

func normalizeProvider(providerName string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "serpapi":
		return "serpapi", nil
	case "google-url", "google":
		return "google-url", nil
	default:
		return "", fmt.Errorf("unsupported provider %q (expected serpapi|google-url)", providerName)
	}
}
