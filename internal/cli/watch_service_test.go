package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

func TestShouldRunWatch(t *testing.T) {
	w := model.Watch{ID: "w1", Enabled: true}
	if !shouldRunWatch(w, "", true) {
		t.Fatalf("expected runAll watch to run")
	}
	if !shouldRunWatch(w, "w1", false) {
		t.Fatalf("expected selected watch to run")
	}
	if shouldRunWatch(w, "w2", false) {
		t.Fatalf("expected non-selected watch not to run")
	}
	w.Enabled = false
	if shouldRunWatch(w, "w1", true) {
		t.Fatalf("expected disabled watch not to run")
	}
}

func TestEvaluateWatchResultTargetPrice(t *testing.T) {
	now := time.Date(2026, 2, 19, 22, 0, 0, 0, time.UTC)
	w := model.Watch{ID: "w1", Name: "athens", TargetPrice: 700}
	res := model.SearchResult{Flights: []model.Flight{{Price: 650, Currency: "USD"}}, URL: "https://x"}

	alert, ok := evaluateWatchResult(&w, res, now)
	if !ok {
		t.Fatalf("expected alert")
	}
	if !strings.Contains(alert.Reason, "target") {
		t.Fatalf("unexpected reason: %s", alert.Reason)
	}
	if w.LastLowestPrice != 650 {
		t.Fatalf("expected updated lowest price")
	}
}

func TestEvaluateWatchResultPriceDrop(t *testing.T) {
	now := time.Date(2026, 2, 19, 22, 0, 0, 0, time.UTC)
	w := model.Watch{ID: "w1", Name: "athens", LastLowestPrice: 900}
	res := model.SearchResult{Flights: []model.Flight{{Price: 800, Currency: "USD"}}, URL: "https://x"}

	alert, ok := evaluateWatchResult(&w, res, now)
	if !ok {
		t.Fatalf("expected alert")
	}
	if !strings.Contains(alert.Reason, "dropped") {
		t.Fatalf("unexpected reason: %s", alert.Reason)
	}
}

func TestRunWatchPassCollectsNotifyErrors(t *testing.T) {
	now := time.Date(2026, 2, 19, 22, 0, 0, 0, time.UTC)
	watches := []model.Watch{{ID: "w1", Name: "athens", Enabled: true, TargetPrice: 700, Query: model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10"}}}

	search := func(model.SearchQuery) (model.SearchResult, error) {
		return model.SearchResult{Flights: []model.Flight{{Price: 650, Currency: "USD"}}, URL: "https://x"}, nil
	}
	notify := func(model.Watch, model.Alert) error {
		return errors.New("smtp down")
	}

	alerts, notifyErrs := runWatchPass(watches, "", true, search, notify, now, false, nil)
	if alerts.Triggered != 1 {
		t.Fatalf("expected 1 triggered alert, got %d", alerts.Triggered)
	}
	if len(alerts.Alerts) != 1 {
		t.Fatalf("expected 1 alert payload, got %d", len(alerts.Alerts))
	}
	if len(notifyErrs) != 1 {
		t.Fatalf("expected 1 notify error, got %d", len(notifyErrs))
	}
	if alerts.NotifyFailures != 1 {
		t.Fatalf("expected 1 notify failure count, got %d", alerts.NotifyFailures)
	}
	if alerts.Evaluated != 1 {
		t.Fatalf("expected 1 evaluated watch, got %d", alerts.Evaluated)
	}
}

func TestRunWatchPassVerboseProviderErrors(t *testing.T) {
	now := time.Date(2026, 2, 19, 22, 0, 0, 0, time.UTC)
	watches := []model.Watch{{ID: "w1", Name: "athens", Enabled: true, Query: model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10"}}}
	search := func(model.SearchQuery) (model.SearchResult, error) {
		return model.SearchResult{}, errors.New("provider timeout")
	}
	notify := func(model.Watch, model.Alert) error { return nil }
	var buf bytes.Buffer

	alerts, notifyErrs := runWatchPass(watches, "", true, search, notify, now, true, &buf)
	if len(alerts.Alerts) != 0 {
		t.Fatalf("expected no alerts")
	}
	if len(notifyErrs) != 0 {
		t.Fatalf("expected no notify errs")
	}
	if alerts.ProviderFailures != 1 {
		t.Fatalf("expected 1 provider failure, got %d", alerts.ProviderFailures)
	}
	if alerts.Evaluated != 1 {
		t.Fatalf("expected 1 evaluated watch, got %d", alerts.Evaluated)
	}
	if !strings.Contains(buf.String(), "provider timeout") {
		t.Fatalf("expected verbose error output, got: %s", buf.String())
	}
}
