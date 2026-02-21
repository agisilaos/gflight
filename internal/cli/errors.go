package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agisilaos/gflight/internal/provider"
)

const (
	ExitSuccess         = 0
	ExitGenericFailure  = 1
	ExitInvalidUsage    = 2
	ExitAuthRequired    = 3
	ExitProviderFailure = 4
	ExitNoMatches       = 5
	ExitNotifyFailure   = 6
)

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

func newExitError(code int, format string, args ...any) error {
	return ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

func wrapExitError(code int, err error) error {
	if err == nil {
		return nil
	}
	var ex ExitError
	if errors.As(err, &ex) {
		return err
	}
	return ExitError{Code: code, Err: err}
}

func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var ex ExitError
	if errors.As(err, &ex) {
		if ex.Code <= 0 {
			return ExitGenericFailure
		}
		return ex.Code
	}
	return ExitGenericFailure
}

func wrapProviderError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, provider.ErrAuthRequired) || errors.Is(err, errProviderAuthMissing) {
		return wrapExitError(ExitAuthRequired, err)
	}
	if errors.Is(err, provider.ErrRateLimited) || errors.Is(err, provider.ErrTransient) {
		return wrapExitError(ExitProviderFailure, err)
	}
	return wrapExitError(ExitProviderFailure, err)
}

func wrapValidationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errProviderAuthMissing) {
		return wrapExitError(ExitAuthRequired, err)
	}
	return newExitError(ExitInvalidUsage, "%v", err)
}

func wrapNotifyError(err error) error {
	if err == nil {
		return nil
	}
	return wrapExitError(ExitNotifyFailure, err)
}

func ErrorHints(err error) []string {
	if err == nil {
		return nil
	}
	hints := make([]string, 0, 2)
	switch {
	case errors.Is(err, errProviderAuthMissing):
		hints = append(hints,
			"gflight auth login --provider google-url",
			"gflight config set serp_api_key <your_key>",
		)
	case errors.Is(err, errWebhookMissing):
		hints = append(hints, "gflight config set webhook_url https://example.com/hook")
	case errors.Is(err, errSMTPIncomplete):
		hints = append(hints,
			"gflight config set smtp_host smtp.gmail.com",
			"gflight config set smtp_user you@example.com",
			"gflight config set smtp_pass <app_password>",
			"gflight config set smtp_sender you@example.com",
		)
	}

	msg := err.Error()
	if strings.Contains(msg, "watch run requires exactly one of --all or --id") {
		hints = append(hints, "gflight help watch run")
	}
	if strings.Contains(msg, "unknown command") || strings.Contains(msg, "unknown watch subcommand") {
		hints = append(hints, "gflight --help")
	}

	return dedupeHints(hints)
}

func dedupeHints(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, h := range in {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}
