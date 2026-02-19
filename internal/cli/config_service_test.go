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
