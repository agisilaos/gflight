package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchAndWatchRejectIgnoredArgumentsBeforeEffects(t *testing.T) {
	commands := [][]string{
		{"search", "--from", "SFO", "--to", "ATH", "--depart", "2026-12-01"},
		{"watch", "create", "--from", "SFO", "--to", "ATH", "--depart", "2026-12-01", "--name", "Audit"},
		{"watch", "list"}, {"watch", "enable", "--id", "synthetic"},
		{"watch", "disable", "--id", "synthetic"}, {"watch", "delete", "--id", "synthetic", "--force"},
		{"watch", "run", "--all"}, {"watch", "test", "--id", "synthetic"},
	}
	for _, command := range commands {
		for _, suffix := range [][]string{{"stray", "--dry-run"}, {"stray", "--not-a-flag"}, {"--", "stray"}} {
			t.Run(command[0]+"/"+command[1]+"/"+suffix[0], func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("HOME", dir)
				t.Setenv("XDG_CONFIG_HOME", dir)
				state := filepath.Join(dir, "state")
				if err := os.MkdirAll(state, 0700); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(state, "watches.json")
				const before = `{"watches":[]}`
				if err := os.WriteFile(path, []byte(before), 0600); err != nil {
					t.Fatal(err)
				}
				args := append([]string{"--state-dir", state}, command...)
				args = append(args, suffix...)
				if err := NewApp("test").Run(args); ExitCode(err) != ExitInvalidUsage {
					t.Fatalf("expected usage failure, got %v", err)
				}
				after, err := os.ReadFile(path)
				if err != nil || string(after) != before {
					t.Fatalf("state changed: %s (%v)", after, err)
				}
			})
		}
	}
}

func TestWatchCreateRejectsStrayWordAfterDryRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	args := []string{"--state-dir", dir, "watch", "create", "--from", "SFO", "--to", "ATH", "--depart", "2026-12-01", "--dry-run", "stray"}
	if err := NewApp("test").Run(args); ExitCode(err) != ExitInvalidUsage {
		t.Fatalf("expected usage error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "watches.json")); !os.IsNotExist(err) {
		t.Fatalf("watch state created: %v", err)
	}
}
