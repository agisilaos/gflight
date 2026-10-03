package cli

import (
	"errors"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/watcher"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agisilaos/gflight/internal/config"
)

type failingOutput struct {
	partial bool
	calls   int
}

func (w *failingOutput) Write(p []byte) (int, error) {
	w.calls++
	if w.partial {
		return len(p) / 2, nil
	}
	return 0, errors.New("sink closed")
}

func TestOutputFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	for _, partial := range []bool{false, true} {
		for _, args := range [][]string{{"--version"}, {"help", "watch", "create"}, {"completion", "bash"}, {"--json", "watch", "list"}, {"--plain", "watch", "list"}, {"auth", "status"}} {
			out := &failingOutput{partial: partial}
			a := NewApp("test")
			a.stdout = out
			err := a.Run(args)
			if ExitCode(err) != 1 || err == nil || !strings.Contains(err.Error(), "stdout write failed") {
				t.Fatalf("args=%v err=%v", args, err)
			}
			if out.calls != 1 {
				t.Fatalf("write retried: %d", out.calls)
			}
		}
	}
}

func TestOutputReceiptFailureKeepsSavedConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	a := NewApp("test")
	a.stdout = &failingOutput{partial: true}
	if err := a.Run([]string{"--json", "config", "set", "provider", "google-url"}); ExitCode(err) != 1 {
		t.Fatalf("err=%v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "google-url" {
		t.Fatalf("provider=%q", cfg.Provider)
	}
}

func TestCommandOutputLatchesShortWrite(t *testing.T) {
	sink := &failingOutput{partial: true}
	out := &commandOutput{writer: sink}
	if _, err := out.Write([]byte("hello")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("again")); !errors.Is(err, io.ErrShortWrite) || sink.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, sink.calls)
	}
}

func TestOutputFailurePreservesCommandStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	setup := NewApp("test")
	setup.stdout = io.Discard
	if err := setup.Run([]string{"auth", "login", "--provider", "google-url"}); err != nil {
		t.Fatal(err)
	}
	disk := watcher.Store{Path: filepath.Join(dir, "watches.json")}
	if err := disk.Save(model.WatchStore{Watches: []model.Watch{{ID: "invalid", Enabled: true, Query: model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-02-30"}}}}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"--json", "--plain", "--no-color"} {
		a := NewApp("test")
		a.stdout = &failingOutput{partial: true}
		err := a.Run([]string{mode, "--state-dir", dir, "watch", "run", "--id", "invalid"})
		if ExitCode(err) != ExitProviderFailure || !strings.Contains(err.Error(), "stdout write failed") {
			t.Fatalf("mode=%s err=%v code=%d", mode, err, ExitCode(err))
		}
	}
}
