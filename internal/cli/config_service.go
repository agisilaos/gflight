package cli

import (
	"fmt"
	"strconv"

	"github.com/agisilaos/gflight/internal/config"
)

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
	case "provider_timeout_seconds":
		return strconv.Itoa(cfg.ProviderTimeoutSec), true
	case "provider_retries":
		return strconv.Itoa(cfg.ProviderRetries), true
	case "provider_backoff_ms":
		return strconv.Itoa(cfg.ProviderBackoffMS), true
	default:
		return "", false
	}
}

func configSet(cfg *config.Config, key, value string) error {
	switch key {
	case "provider":
		normalized, err := normalizeProvider(value)
		if err != nil {
			return err
		}
		cfg.Provider = normalized
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
	case "provider_timeout_seconds":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("provider_timeout_seconds must be positive integer")
		}
		cfg.ProviderTimeoutSec = n
	case "provider_retries":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("provider_retries must be integer >= 0")
		}
		cfg.ProviderRetries = n
	case "provider_backoff_ms":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("provider_backoff_ms must be positive integer")
		}
		cfg.ProviderBackoffMS = n
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}
