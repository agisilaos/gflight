package notify

import (
	"net/http"
	"net/http/httptest"
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
