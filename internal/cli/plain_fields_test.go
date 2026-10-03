package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agisilaos/gflight/internal/model"
)

func TestPlainFieldEscapes(t *testing.T) {
	input := "α\\x\t\r\n\x1b\x00\x7f"
	if got := escapePlainField(input); got != `α\\x\t\r\n\x1b\x00\x7f` {
		t.Fatalf("%q", got)
	}
	out, err := captureStdoutForRun(t, func() error { writePlainKV("name", input); return nil })
	if err != nil || out != "name="+escapePlainField(input)+"\n" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestPlainWatchListEmptyAndEscapedRows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := t.TempDir()
	a := NewApp("test")
	for _, count := range []int{0, 1, 2} {
		store, err := a.watcherStore(stateDir)
		if err != nil {
			t.Fatal(err)
		}
		ws := model.WatchStore{}
		for i := 0; i < count; i++ {
			ws.Watches = append(ws.Watches, model.Watch{ID: "id", Name: "trip\t\n\x1b\\", Query: model.SearchQuery{From: "SFO", To: "ATH"}})
		}
		if err := store.Save(ws); err != nil {
			t.Fatal(err)
		}
		out, err := captureStdoutForRun(t, func() error { return a.Run([]string{"--plain", "--state-dir", stateDir, "watch", "list"}) })
		if err != nil {
			t.Fatal(err)
		}
		rows := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(rows) != count+1 || rows[0] != "id\tname\tenabled\ttarget_price\tfrom\tto\tdepart" {
			t.Fatalf("%q", out)
		}
		for _, row := range rows[1:] {
			cols := strings.Split(row, "\t")
			if len(cols) != 7 || cols[1] != `trip\t\n\x1b\\` {
				t.Fatalf("%q", row)
			}
		}
		jsonOut, err := captureStdoutForRun(t, func() error { return a.Run([]string{"--json", "--state-dir", stateDir, "watch", "list"}) })
		var got []model.Watch
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(jsonOut), &got); err != nil {
			t.Fatal(err)
		}
		if count > 0 && got[0].Name != ws.Watches[0].Name {
			t.Fatalf("JSON value changed: %q", got[0].Name)
		}
	}
}

func TestPlainWatchPreviewDoesNotSave(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "not-created")
	a := NewApp("test")
	args := []string{"--plain", "--state-dir", stateDir, "watch", "create", "--dry-run", "--from", "SFO", "--to", "ATH", "--depart", "2026-12-01", "--name", "line\nname"}
	out, err := captureStdoutForRun(t, func() error { return a.Run(args) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, `dry_run=true`+"\t"+`name=line\nname`+"\t") || strings.Count(out, "\n") != 1 || len(strings.Split(strings.TrimSuffix(out, "\n"), "\t")) != 20 {
		t.Fatalf("%q", out)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run created state: %v", err)
	}
	out, err = captureStdoutForRun(t, func() error { return a.Run(append([]string{"--json"}, args...)) })
	var watch model.Watch
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &watch); err != nil {
		t.Fatal(err)
	}
	if watch.Name != "line\nname" {
		t.Fatalf("%q", watch.Name)
	}
}
