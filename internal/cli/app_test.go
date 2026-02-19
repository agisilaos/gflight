package cli

import (
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
