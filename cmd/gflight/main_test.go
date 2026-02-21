package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsHintsForUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"watcch"}, &stderr)
	if code != 2 {
		t.Fatalf("expected invalid usage code 2, got %d", code)
	}
	out := stderr.String()
	if !strings.Contains(out, `unknown command "watcch"`) {
		t.Fatalf("expected unknown command error, got: %q", out)
	}
	if !strings.Contains(out, "next: gflight --help") {
		t.Fatalf("expected help hint, got: %q", out)
	}
}

func TestRunPrintsHintsForWatchRunSelectorError(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"watch", "run"}, &stderr)
	if code != 2 {
		t.Fatalf("expected invalid usage code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "next: gflight help watch run") {
		t.Fatalf("expected watch run hint, got: %q", stderr.String())
	}
}

