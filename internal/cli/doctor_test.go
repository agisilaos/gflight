package cli

import (
	"testing"

	"github.com/agisilaos/gflight/internal/config"
)

func TestRunDoctorChecksSerpAPIMissingKeyFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()

	report := runDoctorChecks(config.Config{Provider: "serpapi"}, stateDir)
	if report.OK {
		t.Fatalf("expected doctor report to fail")
	}
	if report.Failures == 0 {
		t.Fatalf("expected at least one failure")
	}
}

func TestRunDoctorChecksGoogleURLPassesCoreChecks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()

	report := runDoctorChecks(config.Config{Provider: "google-url"}, stateDir)
	if !report.OK {
		t.Fatalf("expected doctor report ok, got failures=%d", report.Failures)
	}
	if report.Warnings == 0 {
		t.Fatalf("expected warning for missing smtp config")
	}
}

func TestCmdDoctorUsage(t *testing.T) {
	app := NewApp("test")
	err := app.Run([]string{"doctor", "extra"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage, got err=%v code=%d", err, ExitCode(err))
	}
}
