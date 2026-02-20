package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/agisilaos/gflight/internal/watcher"
)

func TestJSONContractCommandsProduceParseableStdout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	mustJSONRun(t, func() error { return app.Run([]string{"--json", "auth", "status"}) })
	mustJSONRun(t, func() error { return app.Run([]string{"--json", "auth", "login", "--provider", "google-url"}) })
	mustJSONRun(t, func() error { return app.Run([]string{"--json", "doctor"}) })
	mustJSONRun(t, func() error {
		return app.Run([]string{"--json", "search", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10"})
	})

	mustJSONRun(t, func() error {
		return app.Run([]string{"--json", "--state-dir", stateDir, "watch", "create", "--name", "a", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10", "--notify-terminal"})
	})

	store := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws, err := store.Load()
	if err != nil {
		t.Fatalf("load watch store: %v", err)
	}
	if len(ws.Watches) != 1 {
		t.Fatalf("expected one watch, got %d", len(ws.Watches))
	}
	id := ws.Watches[0].ID

	mustJSONRun(t, func() error { return app.Run([]string{"--json", "--state-dir", stateDir, "watch", "list"}) })
	mustJSONRun(t, func() error { return app.Run([]string{"--json", "--state-dir", stateDir, "watch", "test", "--id", id}) })
	mustJSONRun(t, func() error {
		return app.Run([]string{"--json", "--state-dir", stateDir, "watch", "run", "--id", id, "--once"})
	})
	mustJSONRun(t, func() error { return app.Run([]string{"--json", "notify", "test", "--channel", "terminal"}) })
	mustJSONRun(t, func() error { return app.Run([]string{"--json", "config", "get", "provider"}) })
	mustJSONRun(t, func() error {
		return app.Run([]string{"--json", "--state-dir", stateDir, "watch", "delete", "--id", id, "--force"})
	})
}

func mustJSONRun(t *testing.T, fn func() error) {
	t.Helper()
	stdout, stderr, err := captureStdoutStderr(t, fn)
	if err != nil {
		t.Fatalf("command failed: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	var v any
	if uerr := json.Unmarshal([]byte(stdout), &v); uerr != nil {
		t.Fatalf("stdout is not valid json: %v\nstdout=%s\nstderr=%s", uerr, stdout, stderr)
	}
}
