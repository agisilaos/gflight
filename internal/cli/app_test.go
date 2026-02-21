package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/watcher"
)

func TestWatchEnableDisableAndDelete(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()

	app := NewApp("test")
	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}

	id := onlyWatchID(t, stateDir)
	if err := app.Run([]string{"--state-dir", stateDir, "watch", "disable", "--id", id}); err != nil {
		t.Fatalf("disable watch: %v", err)
	}
	if w := onlyWatch(t, stateDir); w.Enabled {
		t.Fatalf("watch should be disabled")
	}

	if err := app.Run([]string{"--state-dir", stateDir, "watch", "enable", "--id", id}); err != nil {
		t.Fatalf("enable watch: %v", err)
	}
	if w := onlyWatch(t, stateDir); !w.Enabled {
		t.Fatalf("watch should be enabled")
	}

	if err := app.Run([]string{"--state-dir", stateDir, "watch", "delete", "--id", id, "--force"}); err != nil {
		t.Fatalf("delete watch: %v", err)
	}

	store := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws, err := store.Load()
	if err != nil {
		t.Fatalf("load store after delete: %v", err)
	}
	if len(ws.Watches) != 0 {
		t.Fatalf("expected 0 watches after delete, got %d", len(ws.Watches))
	}
}

func TestWatchDeleteRequiresConfirmationOrForce(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()

	app := NewApp("test")
	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}
	id := onlyWatchID(t, stateDir)

	err := app.Run([]string{"--state-dir", stateDir, "watch", "delete", "--id", id})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force/confirm error, got %v", err)
	}

	err = app.Run([]string{"--state-dir", stateDir, "watch", "delete", "--id", id, "--confirm", "wrong"})
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected confirm error, got %v", err)
	}

	if err := app.Run([]string{"--state-dir", stateDir, "watch", "delete", "--id", id, "--confirm", id}); err != nil {
		t.Fatalf("delete with confirm should pass: %v", err)
	}
}

func TestValidateQuery(t *testing.T) {
	err := validateQuery(model.SearchQuery{})
	if err == nil {
		t.Fatalf("expected validation error for missing fields")
	}
	if got := ExitCode(err); got != ExitInvalidUsage {
		t.Fatalf("expected usage exit code, got %d", got)
	}
}

func TestParseGlobalFlagsAnywhere(t *testing.T) {
	g, rest, err := parseGlobal([]string{"auth", "status", "--json", "--timeout", "5s"})
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}
	if !g.JSON {
		t.Fatalf("expected json global flag to be parsed")
	}
	if g.Timeout != "5s" {
		t.Fatalf("expected timeout=5s, got %q", g.Timeout)
	}
	if len(rest) != 2 || rest[0] != "auth" || rest[1] != "status" {
		t.Fatalf("unexpected rest args: %#v", rest)
	}
}

func TestParseGlobalUnknownFlagFallsThroughToSubcommand(t *testing.T) {
	_, rest, err := parseGlobal([]string{"search", "--from", "SFO"})
	if err != nil {
		t.Fatalf("parseGlobal should not fail on subcommand flags: %v", err)
	}
	if len(rest) != 3 {
		t.Fatalf("expected subcommand args preserved, got %#v", rest)
	}
}

func TestRunGlobalFlagAfterSubcommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	app := NewApp("test")
	err := app.Run([]string{"auth", "status", "--json"})
	if err != nil {
		t.Fatalf("expected no error for trailing global flag, got: %v", err)
	}
}

func TestWatchRunRequiresExactlyOneSelector(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	err := app.Run([]string{"--state-dir", stateDir, "watch", "run"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage for missing selector, got err=%v code=%d", err, ExitCode(err))
	}

	err = app.Run([]string{"--state-dir", stateDir, "watch", "run", "--all", "--id", "w_1"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage for dual selectors, got err=%v code=%d", err, ExitCode(err))
	}
}

func TestWatchRunReturnsAuthRequiredWhenProviderCredentialsMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}

	err := app.Run([]string{"--state-dir", stateDir, "watch", "run", "--all", "--once"})
	if ExitCode(err) != ExitAuthRequired {
		t.Fatalf("expected auth required exit code, got err=%v code=%d", err, ExitCode(err))
	}
}

func TestWatchCreatePlainPrintsStableID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"})
	})
	if err != nil {
		t.Fatalf("watch create failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "watch_id=w_") {
		t.Fatalf("expected stable plain watch_id output, got: %q", out)
	}
}

func TestWatchDeletePlainPrintsDeletedID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}
	id := onlyWatchID(t, stateDir)

	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "--state-dir", stateDir, "watch", "delete", "--id", id, "--force"})
	})
	if err != nil {
		t.Fatalf("watch delete failed: %v", err)
	}
	if strings.TrimSpace(out) != "deleted_id="+id {
		t.Fatalf("expected plain deleted_id output, got: %q", out)
	}
}

func TestWatchListPlainUsesStableSchema(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")
	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "--state-dir", stateDir, "watch", "list"})
	})
	if err != nil {
		t.Fatalf("watch list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header+row output, got: %q", out)
	}
	if lines[0] != "id\tname\tenabled\ttarget_price\tfrom\tto\tdepart" {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	cols := strings.Split(lines[1], "\t")
	if len(cols) != 7 {
		t.Fatalf("expected 7 columns in row, got %d (%q)", len(cols), lines[1])
	}
}

func TestHelpWatchRunShowsCommandSpecificHelp(t *testing.T) {
	app := NewApp("test")
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"help", "watch", "run"})
	})
	if err != nil {
		t.Fatalf("help watch run failed: %v", err)
	}
	if !strings.Contains(out, "gflight watch run - Execute saved watch checks") {
		t.Fatalf("expected command-specific watch run help, got: %q", out)
	}
	if !strings.Contains(out, "--fail-on-provider-errors") {
		t.Fatalf("expected watch run flags in help, got: %q", out)
	}
}

func TestGlobalHelpRoutesToCommandSpecificHelp(t *testing.T) {
	app := NewApp("test")
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--help", "doctor"})
	})
	if err != nil {
		t.Fatalf("--help doctor failed: %v", err)
	}
	if !strings.Contains(out, "gflight doctor - Run preflight checks for automation readiness") {
		t.Fatalf("expected command-specific doctor help, got: %q", out)
	}
	if !strings.Contains(out, "gflight doctor [--strict]") {
		t.Fatalf("expected doctor usage in help, got: %q", out)
	}
}

func TestCompletionScripts(t *testing.T) {
	app := NewApp("test")
	cases := []struct {
		shell       string
		wantSnippet string
	}{
		{shell: "bash", wantSnippet: "complete -F _gflight_completions gflight"},
		{shell: "zsh", wantSnippet: "#compdef gflight"},
		{shell: "fish", wantSnippet: "complete -c gflight -f"},
	}
	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			out, err := captureStdoutForRun(t, func() error {
				return app.Run([]string{"completion", tc.shell})
			})
			if err != nil {
				t.Fatalf("completion failed: %v", err)
			}
			if !strings.Contains(out, tc.wantSnippet) {
				t.Fatalf("expected snippet %q, got: %q", tc.wantSnippet, out)
			}
		})
	}
}

func TestCompletionUsageErrors(t *testing.T) {
	app := NewApp("test")
	err := app.Run([]string{"completion"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage for missing shell, got err=%v code=%d", err, ExitCode(err))
	}
	err = app.Run([]string{"completion", "pwsh"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage for unsupported shell, got err=%v code=%d", err, ExitCode(err))
	}
	err = app.Run([]string{"completion", "path"})
	if ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected invalid usage for missing path shell, got err=%v code=%d", err, ExitCode(err))
	}
}

func TestCompletionPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	app := NewApp("test")
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"completion", "path", "zsh"})
	})
	if err != nil {
		t.Fatalf("completion path zsh failed: %v", err)
	}
	if strings.TrimSpace(out) != filepath.Join(home, ".zsh", "completions", "_gflight") {
		t.Fatalf("unexpected zsh path output: %q", out)
	}
}

func TestTypoSuggestions(t *testing.T) {
	app := NewApp("test")
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "root", args: []string{"serch"}, wantErr: `did you mean "search"`},
		{name: "watch", args: []string{"watch", "rn"}, wantErr: `did you mean "run"`},
		{name: "auth", args: []string{"auth", "statsu"}, wantErr: `did you mean "status"`},
		{name: "config", args: []string{"config", "gt"}, wantErr: `did you mean "get"`},
		{name: "notify", args: []string{"notify", "tset"}, wantErr: `did you mean "test"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := app.Run(tc.args)
			if err == nil {
				t.Fatalf("expected error for args: %v", tc.args)
			}
			if ExitCode(err) != ExitInvalidUsage {
				t.Fatalf("expected invalid usage, got %d", ExitCode(err))
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q in error, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestAuthStatusPlainUsesStableKeyValueOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := NewApp("test")
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "auth", "status"})
	})
	if err != nil {
		t.Fatalf("auth status plain failed: %v", err)
	}
	if !strings.Contains(out, "provider=") || !strings.Contains(out, "serpapi_key=") {
		t.Fatalf("expected stable key=value output, got: %q", out)
	}
}

func TestNotifyTerminalPlainUsesStableKeyValueOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := NewApp("test")
	out, _, err := captureStdoutStderr(t, func() error {
		return app.Run([]string{"--plain", "notify", "test", "--channel", "terminal"})
	})
	if err != nil {
		t.Fatalf("notify test plain failed: %v", err)
	}
	if strings.TrimSpace(out) != "ok=true\tchannel=terminal" {
		t.Fatalf("unexpected notify plain output: %q", out)
	}
}

func TestSearchPlainPrintsHeaderAndURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := NewApp("test")
	if err := app.Run([]string{"auth", "login", "--provider", "google-url"}); err != nil {
		t.Fatalf("auth login: %v", err)
	}
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "search", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"})
	})
	if err != nil {
		t.Fatalf("search plain failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header and url lines, got: %q", out)
	}
	if lines[0] != "price\tcurrency\tairline\tdepart_time\tarrive_time\tstops" {
		t.Fatalf("unexpected search plain header: %q", lines[0])
	}
	if !strings.HasPrefix(lines[len(lines)-1], "url=") {
		t.Fatalf("expected url key=value line, got: %q", lines[len(lines)-1])
	}
}

func TestWatchRunPlainPrintsSummaryKeyValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")
	if err := app.Run([]string{"auth", "login", "--provider", "google-url"}); err != nil {
		t.Fatalf("auth login: %v", err)
	}
	if err := app.Run([]string{"--state-dir", stateDir, "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}
	out, err := captureStdoutForRun(t, func() error {
		return app.Run([]string{"--plain", "--state-dir", stateDir, "watch", "run", "--all", "--once"})
	})
	if err != nil {
		t.Fatalf("watch run plain failed: %v", err)
	}
	if !strings.Contains(out, "evaluated=") || !strings.Contains(out, "triggered=") || !strings.Contains(out, "provider_failures=") {
		t.Fatalf("expected summary key=value output, got: %q", out)
	}
}

func captureStdoutForRun(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = oldOut
	b, _ := io.ReadAll(r)
	_ = r.Close()
	return string(b), runErr
}

func onlyWatchID(t *testing.T, stateDir string) string {
	t.Helper()
	w := onlyWatch(t, stateDir)
	if w.ID == "" {
		t.Fatalf("watch id is empty")
	}
	return w.ID
}

func onlyWatch(t *testing.T, stateDir string) model.Watch {
	t.Helper()
	store := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws, err := store.Load()
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if len(ws.Watches) != 1 {
		t.Fatalf("expected 1 watch, got %d", len(ws.Watches))
	}
	return ws.Watches[0]
}
