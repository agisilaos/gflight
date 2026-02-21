package cli

import (
	"errors"
	"testing"

	"github.com/agisilaos/gflight/internal/config"
)

func TestValidateProviderRuntime(t *testing.T) {
	if err := validateProviderRuntime(config.Config{Provider: "google-url"}); err != nil {
		t.Fatalf("google-url should pass: %v", err)
	}
	err := validateProviderRuntime(config.Config{Provider: "serpapi"})
	if err == nil || !errors.Is(err, errProviderAuthMissing) {
		t.Fatalf("expected provider auth missing error, got: %v", err)
	}
	err = validateProviderRuntime(config.Config{Provider: "unknown"})
	if err == nil || !errors.Is(err, errProviderUnsupported) {
		t.Fatalf("expected provider unsupported error, got: %v", err)
	}
}

func TestValidateNotifyRuntime(t *testing.T) {
	err := validateNotifyEmailRuntime(config.Config{}, "")
	if err == nil || !errors.Is(err, errSMTPIncomplete) {
		t.Fatalf("expected smtp incomplete error, got: %v", err)
	}
	cfg := config.Config{SMTPHost: "h", SMTPUsername: "u", SMTPPassword: "p", SMTPSender: "s", DefaultNotifyEmail: "to@example.com"}
	if err := validateNotifyEmailRuntime(cfg, ""); err != nil {
		t.Fatalf("expected email validation pass, got: %v", err)
	}

	err = validateNotifyWebhookRuntime(config.Config{}, "")
	if err == nil || !errors.Is(err, errWebhookMissing) {
		t.Fatalf("expected webhook missing error, got: %v", err)
	}
	if err := validateNotifyWebhookRuntime(config.Config{}, "https://example.com/h"); err != nil {
		t.Fatalf("expected webhook validation pass, got: %v", err)
	}
}
