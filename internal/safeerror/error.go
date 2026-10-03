// Package safeerror retains error identity without rendering untrusted details.
package safeerror

type Error struct {
	Message string
	Cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }
