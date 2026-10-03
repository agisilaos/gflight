package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashCompletionDepth(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "completion.bash")
	if err := os.WriteFile(file, []byte(bashCompletionScript()), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"watch", ""}, "create list enable disable delete run test"},
		{[]string{"watch", "run", ""}, ""},
		{[]string{"--json", "watch", ""}, "create list enable disable delete run test"},
		{[]string{"--state-dir", "watch", "watch", ""}, "create list enable disable delete run test"},
		{[]string{"--state-dir", ""}, ""},
		{[]string{"watch", "--", ""}, ""},
		{[]string{"completion", "path", ""}, "bash zsh fish"},
		{[]string{"watch", "cr"}, "create"},
		{[]string{"--bogus", ""}, ""},
	}
	for _, tc := range cases {
		script := `source "$1"; shift; COMP_WORDS=("$@"); COMP_CWORD=$((${#COMP_WORDS[@]}-1)); _gflight_completions; printf '%s\n' "${COMPREPLY[@]}"`
		args := append([]string{"--noprofile", "--norc", "-c", script, "test", file, "gflight"}, tc.args...)
		out, err := exec.Command(bash, args...).CombinedOutput()
		if err != nil || strings.Join(strings.Fields(string(out)), " ") != tc.want {
			t.Fatalf("%v: %v %q want %q", tc.args, err, out, tc.want)
		}
	}
}
