package cli

import (
	"errors"
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
