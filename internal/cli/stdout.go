package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// commandOutput records delivery failure without interrupting a completed
// operation's status reporting. It never retries a write or a command.
type commandOutput struct {
	writer io.Writer
	err    error
}

func (w *commandOutput) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

func (a App) output() io.Writer {
	if a.stdout != nil {
		return a.stdout
	}
	return os.Stdout
}

func (a App) Run(args []string) error {
	out := &commandOutput{writer: a.output()}
	a.stdout = out
	err := a.run(args)
	if out.err != nil {
		return errors.Join(err, fmt.Errorf("stdout write failed: %w; any completed changes remain applied; inspect state before retrying", out.err))
	}
	return err
}
