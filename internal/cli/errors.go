package cli

import (
	"errors"
	"fmt"

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
