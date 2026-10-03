package notify

import (
	"errors"
	"github.com/agisilaos/gflight/internal/model"
	"os"
	"testing"
)

func TestPreflightErrorsAreNotDispatched(t *testing.T) {
	n := Notifier{}
	for _, err := range []error{n.SendEmail("", model.Alert{}), n.SendWebhook("", model.Alert{}), n.SendWebhook("not-a-url", model.Alert{})} {
		if !errors.Is(err, ErrNotDispatched) {
			t.Fatalf("not classified: %v", err)
		}
	}
}
func TestTerminalWriteFailureIsReturned(t *testing.T) {
	previous := os.Stderr
	t.Cleanup(func() { os.Stderr = previous })
	f, err := os.CreateTemp(t.TempDir(), "closed")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = f
	if err := (Notifier{}).SendTerminal(model.Alert{}); err == nil {
		t.Fatal("write failure hidden")
	}
}
