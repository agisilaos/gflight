package cli

import (
	"errors"
	"fmt"
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
