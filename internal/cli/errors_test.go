package cli

import (
	"errors"
	"testing"
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
