package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSmokeRealProviderScriptSkipsWhenDisabled(t *testing.T) {
	repoRoot := testRepoRoot(t)
	script := filepath.Join(repoRoot, "scripts", "smoke-real-provider.sh")
	cmd := exec.Command(script)
	cmd.Dir = repoRoot
	cmd.Env = minimalEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected skip success, got err=%v out=%s", err, string(out))
	}
	if !strings.Contains(string(out), "skipped") {
		t.Fatalf("expected skip message, got: %s", string(out))
	}
}

func TestSmokeRealProviderScriptFailsWithoutRequiredEnv(t *testing.T) {
	repoRoot := testRepoRoot(t)
	script := filepath.Join(repoRoot, "scripts", "smoke-real-provider.sh")
	cmd := exec.Command(script)
	cmd.Dir = repoRoot
	cmd.Env = append(minimalEnv(), "GFLIGHT_SMOKE_REAL=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure when required env missing, got success: %s", string(out))
	}
	if !strings.Contains(string(out), "missing required env var") {
		t.Fatalf("expected missing env error, got: %s", string(out))
	}
}

func minimalEnv() []string {
	path := os.Getenv("PATH")
	home := os.Getenv("HOME")
	env := []string{}
	if path != "" {
		env = append(env, "PATH="+path)
	}
	if home != "" {
		env = append(env, "HOME="+home)
	}
	return env
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
