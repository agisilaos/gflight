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

func TestDoctorStrictFailsOnWarnings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	app := NewApp("test")
	if err := app.Run([]string{"auth", "login", "--provider", "google-url"}); err != nil {
		t.Fatalf("auth login: %v", err)
	}
	err := app.Run([]string{"doctor", "--strict"})
	if ExitCode(err) != ExitGenericFailure {
		t.Fatalf("expected strict doctor failure, got err=%v code=%d", err, ExitCode(err))
	}
}
