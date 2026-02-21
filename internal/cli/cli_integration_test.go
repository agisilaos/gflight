package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agisilaos/gflight/internal/watcher"
)

func TestCLIIntegrationCoreAgentFlows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	cases := []struct {
		name     string
		args     []string
		exitCode int
		wantJSON bool
		errHas   string
	}{
		{
			name:     "auth login google-url",
			args:     []string{"--json", "auth", "login", "--provider", "google-url"},
			exitCode: ExitSuccess,
			wantJSON: true,
		},
		{
			name:     "doctor json",
			args:     []string{"--json", "doctor"},
			exitCode: ExitSuccess,
			wantJSON: true,
		},
		{
			name:     "search json",
			args:     []string{"--json", "search", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"},
			exitCode: ExitSuccess,
			wantJSON: true,
		},
		{
			name:     "notify webhook missing",
			args:     []string{"--json", "notify", "test", "--channel", "webhook"},
			exitCode: ExitNotifyFailure,
			errHas:   "missing webhook url",
		},
		{
			name:     "watch run selector required",
			args:     []string{"--json", "watch", "run"},
			exitCode: ExitInvalidUsage,
			errHas:   "exactly one of --all or --id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--state-dir", stateDir}, tc.args...)
			stdout, stderr, code, errText := runCLIWithCapture(t, app, args)
			if code != tc.exitCode {
				t.Fatalf("exit code=%d want=%d\nstdout=%s\nstderr=%s", code, tc.exitCode, stdout, stderr)
			}
			if tc.wantJSON {
				var v any
				if err := json.Unmarshal([]byte(stdout), &v); err != nil {
					t.Fatalf("stdout not valid json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
				}
			}
			if tc.errHas != "" && !strings.Contains(errText, tc.errHas) {
				t.Fatalf("error missing %q\nerr=%s", tc.errHas, errText)
			}
		})
	}
}

func TestCLIIntegrationWatchLifecycle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	_, _, code, _ := runCLIWithCapture(t, app, []string{"--state-dir", stateDir, "--json", "auth", "login", "--provider", "google-url"})
	if code != ExitSuccess {
		t.Fatalf("auth login failed code=%d", code)
	}

	stdout, stderr, code, _ := runCLIWithCapture(t, app, []string{"--state-dir", stateDir, "--json", "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"})
	if code != ExitSuccess {
		t.Fatalf("watch create failed code=%d stderr=%s", code, stderr)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("watch create json parse: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("missing watch id in create output: %v", created)
	}

	_, _, code, _ = runCLIWithCapture(t, app, []string{"--state-dir", stateDir, "--json", "watch", "run", "--id", id, "--once"})
	if code != ExitSuccess {
		t.Fatalf("watch run failed code=%d", code)
	}

	_, _, code, _ = runCLIWithCapture(t, app, []string{"--state-dir", stateDir, "--json", "watch", "delete", "--id", id, "--force"})
	if code != ExitSuccess {
		t.Fatalf("watch delete failed code=%d", code)
	}

	store := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws, err := store.Load()
	if err != nil {
		t.Fatalf("load watch store: %v", err)
	}
	if len(ws.Watches) != 0 {
		t.Fatalf("expected no watches after delete, got %d", len(ws.Watches))
	}
}

func runCLIWithCapture(t *testing.T, app App, args []string) (stdout string, stderr string, code int, errText string) {
	t.Helper()
	stdout, stderr, err := captureStdoutStderr(t, func() error {
		return app.Run(args)
	})
	if err != nil {
		errText = err.Error()
	}
	return stdout, stderr, ExitCode(err), errText
}
