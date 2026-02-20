package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agisilaos/gflight/internal/config"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type doctorReport struct {
	OK       bool          `json:"ok"`
	Failures int           `json:"failures"`
	Warnings int           `json:"warnings"`
	Checks   []doctorCheck `json:"checks"`
}

func (a App) cmdDoctor(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	strict := fs.Bool("strict", false, "Treat warnings as failures")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if len(fs.Args()) != 0 {
		return newExitError(ExitInvalidUsage, "usage: gflight doctor [--strict]")
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	report := runDoctorChecks(cfg, g.StateDir)
	effectiveFailures := report.Failures
	if *strict {
		effectiveFailures += report.Warnings
	}
	if g.JSON {
		if err := writeJSON(report); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
	} else {
		for _, c := range report.Checks {
			fmt.Printf("%s\t%s\t%s\n", strings.ToUpper(c.Status), c.Name, c.Message)
		}
		fmt.Printf("summary\tfailures=%d\twarnings=%d\n", report.Failures, report.Warnings)
	}
	if effectiveFailures > 0 {
		if *strict && report.Warnings > 0 && report.Failures == 0 {
			return newExitError(ExitGenericFailure, "doctor strict mode found %d warning(s)", report.Warnings)
		}
		return newExitError(ExitGenericFailure, "doctor found %d failing check(s)", report.Failures)
	}
	return nil
}

func runDoctorChecks(cfg config.Config, stateOverride string) doctorReport {
	checks := []doctorCheck{}
	add := func(name, status, message string) {
		checks = append(checks, doctorCheck{Name: name, Status: status, Message: message})
	}

	switch strings.ToLower(cfg.Provider) {
	case "serpapi":
		if cfg.SerpAPIKey == "" {
			add("provider.auth", "fail", "provider=serpapi but serp_api_key is missing")
		} else {
			add("provider.auth", "ok", "serpapi key present")
		}
	case "google-url", "google":
		add("provider.auth", "ok", "provider=google-url does not require API key")
	default:
		add("provider.auth", "fail", fmt.Sprintf("unsupported provider %q", cfg.Provider))
	}

	if dir, err := config.ConfigDir(); err != nil {
		add("paths.config", "fail", err.Error())
	} else if err := ensureWritableDir(dir); err != nil {
		add("paths.config", "fail", err.Error())
	} else {
		add("paths.config", "ok", dir)
	}

	if dir, err := config.StateDir(stateOverride); err != nil {
		add("paths.state", "fail", err.Error())
	} else if err := ensureWritableDir(dir); err != nil {
		add("paths.state", "fail", err.Error())
	} else {
		add("paths.state", "ok", dir)
	}

	if cfg.SMTPHost == "" && cfg.SMTPUsername == "" && cfg.SMTPPassword == "" && cfg.SMTPSender == "" {
		add("notify.email", "warn", "smtp is not configured")
	} else {
		missing := []string{}
		if cfg.SMTPHost == "" {
			missing = append(missing, "smtp_host")
		}
		if cfg.SMTPUsername == "" {
			missing = append(missing, "smtp_user")
		}
		if cfg.SMTPPassword == "" {
			missing = append(missing, "smtp_pass")
		}
		if cfg.SMTPSender == "" {
			missing = append(missing, "smtp_sender")
		}
		if len(missing) > 0 {
			add("notify.email", "fail", "missing required smtp fields: "+strings.Join(missing, ", "))
		} else {
			add("notify.email", "ok", "smtp configuration complete")
		}
	}

	if strings.TrimSpace(cfg.WebhookURL) == "" {
		add("notify.webhook", "warn", "webhook_url is not configured")
	} else {
		add("notify.webhook", "ok", "webhook_url configured")
	}

	report := doctorReport{Checks: checks}
	for _, c := range checks {
		switch c.Status {
		case "fail":
			report.Failures++
		case "warn":
			report.Warnings++
		}
	}
	report.OK = report.Failures == 0
	return report
}

func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".gflight-write-test")
	if err := os.WriteFile(probe, []byte("ok\n"), 0o600); err != nil {
		return err
	}
	if err := os.Remove(probe); err != nil {
		return err
	}
	return nil
}
