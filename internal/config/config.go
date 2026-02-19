package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Provider           string `json:"provider"`
	SerpAPIKey         string `json:"serp_api_key,omitempty"`
	SMTPHost           string `json:"smtp_host,omitempty"`
	SMTPPort           int    `json:"smtp_port,omitempty"`
	SMTPUsername       string `json:"smtp_username,omitempty"`
	SMTPPassword       string `json:"smtp_password,omitempty"`
	SMTPSender         string `json:"smtp_sender,omitempty"`
	DefaultNotifyEmail string `json:"default_notify_email,omitempty"`
}

func ConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gflight"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gflight"), nil
}

func StateDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if env := os.Getenv("GFLIGHT_STATE_DIR"); env != "" {
		return env, nil
	}
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "gflight"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "gflight"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	cfg := Config{
		Provider: "serpapi",
		SMTPPort: 587,
	}
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			applyEnv(&cfg)
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	applyEnv(&cfg)
	if cfg.Provider == "" {
		cfg.Provider = "serpapi"
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("GFLIGHT_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("GFLIGHT_SERPAPI_KEY"); v != "" {
		cfg.SerpAPIKey = v
	}
	if v := os.Getenv("GFLIGHT_SMTP_HOST"); v != "" {
		cfg.SMTPHost = v
	}
	if v := os.Getenv("GFLIGHT_SMTP_USER"); v != "" {
		cfg.SMTPUsername = v
	}
	if v := os.Getenv("GFLIGHT_SMTP_PASS"); v != "" {
		cfg.SMTPPassword = v
	}
	if v := os.Getenv("GFLIGHT_SMTP_SENDER"); v != "" {
		cfg.SMTPSender = v
	}
	if v := os.Getenv("GFLIGHT_NOTIFY_EMAIL"); v != "" {
		cfg.DefaultNotifyEmail = v
	}
}
