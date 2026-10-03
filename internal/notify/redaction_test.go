package notify

import (
	"errors"
	"fmt"
	"github.com/agisilaos/gflight/internal/model"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type redactionTransport func(*http.Request) (*http.Response, error)

func (f redactionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWebhookDiagnosticsOmitSecrets(t *testing.T) {
	const secret = "SENTINEL-webhook-secret"
	for _, cause := range []error{errors.New(secret), &net.DNSError{Err: secret, Name: secret}} {
		client := &http.Client{Transport: redactionTransport(func(*http.Request) (*http.Response, error) { return nil, cause })}
		err := (Notifier{}).sendWebhookWithClient("https://example.invalid/"+secret+"?token="+secret, model.Alert{}, client)
		if err == nil || strings.Contains(fmt.Sprintf("%v %+v", err, err), secret) {
			t.Fatalf("unsafe diagnostic: %v", err)
		}
		if !errors.Is(err, cause) {
			t.Fatal("cause lost")
		}
	}
	for _, code := range []int{400, 401, 429, 500} {
		client := &http.Client{Transport: redactionTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Status: secret, Body: io.NopCloser(strings.NewReader(secret))}, nil
		})}
		err := (Notifier{}).sendWebhookWithClient("https://example.invalid/"+secret, model.Alert{}, client)
		if err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), fmt.Sprint(code)) {
			t.Fatalf("unsafe HTTP diagnostic: %v", err)
		}
	}
	err := (Notifier{}).sendWebhookWithClient("https://example.invalid/"+secret+"/%invalid", model.Alert{}, &http.Client{})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe URL diagnostic: %v", err)
	}
}
