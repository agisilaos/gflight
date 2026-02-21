package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agisilaos/gflight/internal/config"
)

var (
	errProviderUnsupported = errors.New("unsupported provider")
	errProviderAuthMissing = errors.New("provider auth missing")
	errSMTPIncomplete      = errors.New("smtp configuration incomplete")
	errWebhookMissing      = errors.New("webhook url missing")
)

func validateProviderRuntime(cfg config.Config) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "google-url", "google":
		return nil
	case "serpapi":
		if strings.TrimSpace(cfg.SerpAPIKey) == "" {
			return fmt.Errorf("%w: provider=serpapi requires serp_api_key", errProviderAuthMissing)
		}
		return nil
	default:
		return fmt.Errorf("%w: %q", errProviderUnsupported, cfg.Provider)
	}
}

func validateNotifyEmailRuntime(cfg config.Config, recipient string) error {
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		recipient = strings.TrimSpace(cfg.DefaultNotifyEmail)
	}
	if recipient == "" {
		return fmt.Errorf("%w: missing email recipient (set --to or notify_email)", errSMTPIncomplete)
	}
	missing := missingSMTPFields(cfg)
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing required smtp fields: %s", errSMTPIncomplete, strings.Join(missing, ", "))
	}
	return nil
}

func validateNotifyWebhookRuntime(cfg config.Config, url string) error {
	if strings.TrimSpace(url) == "" {
		url = cfg.WebhookURL
	}
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("%w: missing webhook url (set --url or webhook_url)", errWebhookMissing)
	}
	return nil
}

func configReadinessChecks(cfg config.Config) []doctorCheck {
	checks := make([]doctorCheck, 0, 3)

	if err := validateProviderRuntime(cfg); err != nil {
		checks = append(checks, doctorCheck{Name: "provider.auth", Status: "fail", Message: err.Error()})
	} else {
		if strings.EqualFold(cfg.Provider, "google-url") || strings.EqualFold(cfg.Provider, "google") {
			checks = append(checks, doctorCheck{Name: "provider.auth", Status: "ok", Message: "provider=google-url does not require API key"})
		} else {
			checks = append(checks, doctorCheck{Name: "provider.auth", Status: "ok", Message: "serpapi key present"})
		}
	}

	missing := missingSMTPFields(cfg)
	if len(missing) == 0 {
		if strings.TrimSpace(cfg.SMTPHost) == "" && strings.TrimSpace(cfg.SMTPUsername) == "" && strings.TrimSpace(cfg.SMTPPassword) == "" && strings.TrimSpace(cfg.SMTPSender) == "" {
			checks = append(checks, doctorCheck{Name: "notify.email", Status: "warn", Message: "smtp is not configured"})
		} else {
			checks = append(checks, doctorCheck{Name: "notify.email", Status: "ok", Message: "smtp configuration complete"})
		}
	} else {
		if len(missing) == 4 {
			checks = append(checks, doctorCheck{Name: "notify.email", Status: "warn", Message: "smtp is not configured"})
		} else {
			checks = append(checks, doctorCheck{Name: "notify.email", Status: "fail", Message: "missing required smtp fields: " + strings.Join(missing, ", ")})
		}
	}

	if strings.TrimSpace(cfg.WebhookURL) == "" {
		checks = append(checks, doctorCheck{Name: "notify.webhook", Status: "warn", Message: "webhook_url is not configured"})
	} else {
		checks = append(checks, doctorCheck{Name: "notify.webhook", Status: "ok", Message: "webhook_url configured"})
	}

	return checks
}

func missingSMTPFields(cfg config.Config) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		missing = append(missing, "smtp_host")
	}
	if strings.TrimSpace(cfg.SMTPUsername) == "" {
		missing = append(missing, "smtp_user")
	}
	if strings.TrimSpace(cfg.SMTPPassword) == "" {
		missing = append(missing, "smtp_pass")
	}
	if strings.TrimSpace(cfg.SMTPSender) == "" {
		missing = append(missing, "smtp_sender")
	}
	return missing
}
