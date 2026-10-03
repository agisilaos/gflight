package provider

import (
	"encoding/json"
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

func TestSerpItineraryDuration(t *testing.T) {
	for _, tc := range []struct {
		name, raw, want string
		invalid         bool
	}{
		{"direct", `{"total_duration":120,"flights":[{"duration":120}]}`, "120m", false},
		{"connection", `{"total_duration":240,"flights":[{"duration":60},{"duration":120}],"layovers":[{"duration":60}]}`, "240m", false},
		{"overnight zones", `{"total_duration":780,"flights":[{"duration":120,"departure_airport":{"time":"2026-11-01 23:00"}},{"duration":600,"arrival_airport":{"time":"2026-11-03 06:00"}}]}`, "780m", false},
		{"missing", `{"flights":[{"duration":60},{"duration":120}]}`, "", false},
		{"null", `{"total_duration":null,"flights":[{"duration":60}]}`, "", false},
		{"zero", `{"total_duration":0,"flights":[{"duration":60}]}`, "", false},
		{"negative", `{"total_duration":-1,"flights":[{"duration":60}]}`, "", false},
		{"malformed", `{"total_duration":"240","flights":[{"duration":60}]}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw serpFlight
			err := json.Unmarshal([]byte(tc.raw), &raw)
			if tc.invalid {
				if err == nil {
					t.Fatal("accepted malformed duration")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := mapSerpFlight(model.SearchQuery{}, raw)
			if got.Duration != tc.want {
				t.Fatalf("duration=%q want=%q", got.Duration, tc.want)
			}
		})
	}
}
