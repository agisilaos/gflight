package cli

import (
	"testing"

	"github.com/agisilaos/gflight/internal/config"
)

func TestApplyAuthLogin(t *testing.T) {
	cfg := config.Config{Provider: "serpapi"}
	if err := applyAuthLogin(&cfg, "google", "abc123"); err != nil {
		t.Fatalf("apply auth login: %v", err)
	}
	if cfg.Provider != "google-url" {
		t.Fatalf("expected normalized provider, got %q", cfg.Provider)
	}
	if cfg.SerpAPIKey != "abc123" {
		t.Fatalf("expected api key to be set")
	}
}

func TestApplyAuthLoginRejectsInvalidProvider(t *testing.T) {
	cfg := config.Config{}
	if err := applyAuthLogin(&cfg, "invalid", ""); err == nil {
		t.Fatalf("expected invalid provider error")
	}
}

func TestAuthStatus(t *testing.T) {
	cfg := config.Config{
		Provider:     "serpapi",
		SerpAPIKey:   "secret",
		SMTPHost:     "smtp.example.com",
		SMTPUsername: "u",
		SMTPPassword: "p",
		SMTPSender:   "noreply@example.com",
	}
	status := authStatus(cfg)
	if status["provider"] != "serpapi" {
		t.Fatalf("unexpected provider")
	}
	if status["serpapi_key"] != true {
		t.Fatalf("expected serpapi key true")
	}
	if status["smtp_configured"] != true {
		t.Fatalf("expected smtp configured true")
	}
}
