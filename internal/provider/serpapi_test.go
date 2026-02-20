package provider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

func TestSerpAPIRetriesTransientAndSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"temporary"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"best_flights":[{"price":650,"flights":[{"airline":"Aegean","flight_number":"A3","departure_airport":{"airport":"SFO","time":"10:00"},"arrival_airport":{"airport":"ATH","time":"09:00"},"duration":780}],"layovers":[]}],"other_flights":[],"search_metadata":{"google_flights_url":"https://google.example/flights"}}`))
	}))
	defer srv.Close()

	p := SerpAPIProvider{
		APIKey:  "k",
		BaseURL: srv.URL,
		Retries: 2,
		Backoff: time.Millisecond,
		Timeout: 2 * time.Second,
		Client:  &http.Client{Timeout: 2 * time.Second},
	}
	res, err := p.Search(model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10", Currency: "USD"})
	if err != nil {
		t.Fatalf("search should succeed after retry: %v", err)
	}
	if len(res.Flights) != 1 {
		t.Fatalf("expected 1 flight, got %d", len(res.Flights))
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func TestSerpAPIClassifiesAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	p := SerpAPIProvider{APIKey: "bad", BaseURL: srv.URL, Retries: 2, Backoff: time.Millisecond}
	_, err := p.Search(model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10"})
	if err == nil {
		t.Fatalf("expected auth error")
	}
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("expected ErrAuthRequired, got %v", err)
	}
}

func TestSerpAPIClassifiesRateLimitAndRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	p := SerpAPIProvider{APIKey: "k", BaseURL: srv.URL, Retries: 1, Backoff: time.Millisecond}
	_, err := p.Search(model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10"})
	if err == nil {
		t.Fatalf("expected rate limit error")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func TestSerpAPITimeoutIsTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"best_flights":[],"other_flights":[],"search_metadata":{"google_flights_url":"https://x"}}`))
	}))
	defer srv.Close()

	p := SerpAPIProvider{APIKey: "k", BaseURL: srv.URL, Retries: 0, Timeout: 20 * time.Millisecond, Client: &http.Client{Timeout: 20 * time.Millisecond}}
	_, err := p.Search(model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10"})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, ErrTransient) {
		t.Fatalf("expected ErrTransient, got %v", err)
	}
}

func TestBuildSerpURLUsesBasePath(t *testing.T) {
	got := buildSerpURL("https://example.com", model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10", Adults: 1}, "key")
	wantPrefix := "https://example.com/search.json?"
	if got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, got)
	}
	if !strings.Contains(got, "api_key=key") {
		t.Fatalf("expected api key in url: %s", got)
	}
}
