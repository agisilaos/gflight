package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/agisilaos/gflight/internal/provider"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: ExitSuccess},
		{name: "generic", err: errors.New("boom"), want: ExitGenericFailure},
		{name: "usage", err: newExitError(ExitInvalidUsage, "bad args"), want: ExitInvalidUsage},
		{name: "notify", err: wrapExitError(ExitNotifyFailure, errors.New("smtp")), want: ExitNotifyFailure},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWrapProviderErrorMapping(t *testing.T) {
	if got := ExitCode(wrapProviderError(provider.ErrAuthRequired)); got != ExitAuthRequired {
		t.Fatalf("expected auth required exit code, got %d", got)
	}
	if got := ExitCode(wrapProviderError(provider.ErrRateLimited)); got != ExitProviderFailure {
		t.Fatalf("expected provider failure exit code, got %d", got)
	}
}

func TestWrapValidationErrorMapping(t *testing.T) {
	if got := ExitCode(wrapValidationError(errors.New("bad"))); got != ExitInvalidUsage {
		t.Fatalf("expected invalid usage exit code, got %d", got)
	}
	if got := ExitCode(wrapValidationError(errProviderAuthMissing)); got != ExitAuthRequired {
		t.Fatalf("expected auth required exit code, got %d", got)
	}
}

func TestWrapNotifyErrorMapping(t *testing.T) {
	if got := ExitCode(wrapNotifyError(errors.New("smtp"))); got != ExitNotifyFailure {
		t.Fatalf("expected notify failure exit code, got %d", got)
	}
}

func TestErrorHints(t *testing.T) {
	t.Run("provider auth missing", func(t *testing.T) {
		hints := ErrorHints(errProviderAuthMissing)
		if len(hints) == 0 {
			t.Fatalf("expected auth hints")
		}
		joined := strings.Join(hints, "\n")
		if !strings.Contains(joined, "gflight auth login") {
			t.Fatalf("expected auth login hint, got %v", hints)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		err := newExitError(ExitInvalidUsage, `unknown command "watcch"`)
		hints := ErrorHints(err)
		if len(hints) != 1 || hints[0] != "gflight --help" {
			t.Fatalf("unexpected unknown command hints: %v", hints)
		}
	})

	t.Run("watch run selector", func(t *testing.T) {
		err := newExitError(ExitInvalidUsage, "watch run requires exactly one of --all or --id")
		hints := ErrorHints(err)
		if len(hints) != 1 || hints[0] != "gflight help watch run" {
			t.Fatalf("unexpected selector hints: %v", hints)
		}
	})
}
