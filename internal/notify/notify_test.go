package notify

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

func TestSendWebhookSuccess(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected json content type")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := Notifier{}
	err := n.SendWebhook(srv.URL, model.Alert{WatchID: "w1", WatchName: "a", TriggeredAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("expected webhook success, got %v", err)
	}
	if !hit {
		t.Fatalf("expected webhook endpoint hit")
	}
}

func TestSendWebhookHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	}))
	defer srv.Close()

	n := Notifier{}
	err := n.SendWebhook(srv.URL, model.Alert{WatchID: "w1", WatchName: "a", TriggeredAt: time.Now().UTC()})
	if err == nil {
		t.Fatalf("expected webhook failure")
	}
}

func TestSendWebhookClassifiesRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("slow down"))
	}))
	defer srv.Close()

	n := Notifier{}
	err := n.SendWebhook(srv.URL, model.Alert{WatchID: "w1", WatchName: "a", TriggeredAt: time.Now().UTC()})
	if err == nil {
		t.Fatalf("expected webhook failure")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limited classification, got: %v", err)
	}
}

func TestClassifyWebhookRequestErrorDNS(t *testing.T) {
	err := &neturl.Error{Err: &net.DNSError{Err: "no such host", Name: "example.invalid"}}
	got := classifyWebhookRequestError(err)
	if !strings.Contains(got, "dns lookup failed") {
		t.Fatalf("expected dns classification, got: %s", got)
	}
}

func TestClassifyWebhookRequestErrorTimeout(t *testing.T) {
	err := &neturl.Error{Err: timeoutErr{}, Op: "Post", URL: "https://example.invalid"}
	got := classifyWebhookRequestError(err)
	if !strings.Contains(got, "timeout") {
		t.Fatalf("expected timeout classification, got: %s", got)
	}
}

func TestClassifyWebhookRequestErrorGeneric(t *testing.T) {
	got := classifyWebhookRequestError(errors.New("boom"))
	if !strings.Contains(got, "transport error") {
		t.Fatalf("expected generic transport classification, got: %s", got)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
