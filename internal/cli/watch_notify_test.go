package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
	"github.com/agisilaos/gflight/internal/watcher"
)

func TestSendWatchNotificationsReturnsEmailError(t *testing.T) {
	app := NewApp("test")
	n := notify.Notifier{Config: config.Config{}}
	w := model.Watch{
		ID:          "w_1",
		NotifyEmail: true,
		EmailTo:     "alerts@example.com",
	}
	alert := model.Alert{WatchID: "w_1", WatchName: "test", TriggeredAt: time.Now().UTC()}

	err := app.sendWatchNotifications(n, w, alert)
	if err == nil {
		t.Fatalf("expected email notification error")
	}
}

func TestSendWatchNotificationsReturnsWebhookError(t *testing.T) {
	app := NewApp("test")
	n := notify.Notifier{Config: config.Config{}}
	w := model.Watch{
		ID:            "w_2",
		NotifyWebhook: true,
	}
	alert := model.Alert{WatchID: "w_2", WatchName: "test", TriggeredAt: time.Now().UTC()}

	err := app.sendWatchNotifications(n, w, alert)
	if err == nil {
		t.Fatalf("expected webhook notification error")
	}
}

func TestWatchTestJSONKeepsStdoutClean(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	app := NewApp("test")

	if err := app.Run([]string{"--state-dir", stateDir, "--json", "watch", "create", "--name", "athens", "--from", "SFO", "--to", "ATH", "--depart", "2026-06-10", "--notify-terminal"}); err != nil {
		t.Fatalf("create watch: %v", err)
	}

	store := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws, err := store.Load()
	if err != nil {
		t.Fatalf("load watch store: %v", err)
	}
	if len(ws.Watches) != 1 {
		t.Fatalf("expected 1 watch, got %d", len(ws.Watches))
	}

	stdout, stderr, runErr := captureStdoutStderr(t, func() error {
		return app.Run([]string{"--state-dir", stateDir, "--json", "watch", "test", "--id", ws.Watches[0].ID})
	})
	if runErr != nil {
		t.Fatalf("watch test failed: %v", runErr)
	}
	if strings.Contains(stdout, "ALERT ") {
		t.Fatalf("stdout should not include terminal alert text: %q", stdout)
	}
	if !strings.Contains(strings.TrimSpace(stdout), "\"watch_id\"") {
		t.Fatalf("stdout should contain json payload, got: %q", stdout)
	}
	if !strings.Contains(stderr, "ALERT ") {
		t.Fatalf("stderr should include terminal alert text, got: %q", stderr)
	}
}

func captureStdoutStderr(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	oldOut := os.Stdout
	oldErr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	runErr := fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	bOut, _ := io.ReadAll(rOut)
	bErr, _ := io.ReadAll(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return string(bOut), string(bErr), runErr
}
