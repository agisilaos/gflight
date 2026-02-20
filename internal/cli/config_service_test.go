package cli

import (
	"testing"

	"github.com/agisilaos/gflight/internal/config"
)

func TestConfigSetProviderNormalization(t *testing.T) {
	cfg := config.Config{}
	if err := configSet(&cfg, "provider", "google"); err != nil {
		t.Fatalf("configSet provider: %v", err)
	}
	if cfg.Provider != "google-url" {
		t.Fatalf("expected normalized provider, got %q", cfg.Provider)
	}
}

func TestConfigSetSMTPPortValidation(t *testing.T) {
	cfg := config.Config{}
	if err := configSet(&cfg, "smtp_port", "-1"); err == nil {
		t.Fatalf("expected invalid smtp_port error")
	}
	if err := configSet(&cfg, "smtp_port", "587"); err != nil {
		t.Fatalf("expected valid smtp_port: %v", err)
	}
	if cfg.SMTPPort != 587 {
		t.Fatalf("expected smtp port set")
	}
}

func TestConfigGetMasksSerpKey(t *testing.T) {
	cfg := config.Config{SerpAPIKey: "secret"}
	v, ok := configGet(cfg, "serp_api_key")
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if v != "***" {
		t.Fatalf("expected masked key, got %q", v)
	}
}

func TestConfigSetProviderReliabilityKeys(t *testing.T) {
	cfg := config.Config{}
	if err := configSet(&cfg, "provider_timeout_seconds", "15"); err != nil {
		t.Fatalf("set timeout: %v", err)
	}
	if err := configSet(&cfg, "provider_retries", "3"); err != nil {
		t.Fatalf("set retries: %v", err)
	}
	if err := configSet(&cfg, "provider_backoff_ms", "250"); err != nil {
		t.Fatalf("set backoff: %v", err)
	}
	if cfg.ProviderTimeoutSec != 15 || cfg.ProviderRetries != 3 || cfg.ProviderBackoffMS != 250 {
		t.Fatalf("unexpected reliability config values: %+v", cfg)
	}
}

func TestConfigSetWebhookURL(t *testing.T) {
	cfg := config.Config{}
	url := "https://example.com/hook"
	if err := configSet(&cfg, "webhook_url", url); err != nil {
		t.Fatalf("set webhook_url: %v", err)
	}
	if cfg.WebhookURL != url {
		t.Fatalf("expected webhook url %q, got %q", url, cfg.WebhookURL)
	}
	v, ok := configGet(cfg, "webhook_url")
	if !ok || v != url {
		t.Fatalf("expected webhook_url from configGet, got ok=%t v=%q", ok, v)
	}
}
