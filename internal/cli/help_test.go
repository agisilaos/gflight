package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFocusedHelpWithoutConfiguration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	if err := os.Mkdir(filepath.Join(dir, "gflight"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gflight", "config.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{"search", "watch create", "watch delete"} {
		words := strings.Fields(topic)
		var first string
		for _, args := range [][]string{append([]string{"help"}, words...), append(append([]string{}, words...), "--help"), append(append([]string{}, words...), "-h")} {
			out, err := captureStdoutForRun(t, func() error { return NewApp("test").Run(args) })
			if err != nil || !strings.HasPrefix(out, "gflight "+topic+" -") || !strings.Contains(out, "EXAMPLE:") {
				t.Fatalf("%v: %v %s", args, err, out)
			}
			if first != "" && out != first {
				t.Fatalf("help spellings differ: %v", args)
			}
			first = out
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "gflight", "watches.json")); !os.IsNotExist(err) {
		t.Fatalf("help touched state: %v", err)
	}
}

func TestMissingInputPointsToLeafHelp(t *testing.T) {
	for _, topic := range []string{"search", "watch create", "watch delete"} {
		err := NewApp("test").Run(strings.Fields(topic))
		if ExitCode(err) != ExitInvalidUsage || !strings.Contains(err.Error(), "gflight help "+topic) {
			t.Fatalf("%s: %v", topic, err)
		}
	}
}
