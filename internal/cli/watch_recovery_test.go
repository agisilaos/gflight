package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
	"github.com/agisilaos/gflight/internal/watcher"
)

func recoveryWatch() model.WatchStore {
	return model.WatchStore{Watches: []model.Watch{{ID: "w1", Enabled: true, LastLowestPrice: 900, NotifyTerminal: true, NotifyEmail: true, EmailTo: "synthetic@example.invalid"}}}
}
func priceResult(price int) watchSearchFunc {
	return func(model.SearchQuery) (model.SearchResult, error) {
		return model.SearchResult{Flights: []model.Flight{{Price: price, Currency: "USD"}}}, nil
	}
}

func TestPendingDeliverySurvivesSaveReload(t *testing.T) {
	disk := watcher.Store{Path: filepath.Join(t.TempDir(), "watches.json")}
	ws := recoveryWatch()
	terminal, email := 0, 0
	send := func(d model.AlertDelivery, a model.Alert) error {
		if d.Channel == "terminal" {
			terminal++
			return nil
		}
		email++
		return fmt.Errorf("%w: missing SMTP configuration", notify.ErrNotDispatched)
	}
	run := func(search watchSearchFunc, retry bool) (watchRunReport, []string, error) {
		return runWatchPass(ws.Watches, "", true, search, send, func() error { return disk.Save(ws) }, retry, time.Now(), false, nil)
	}
	first, errs, err := run(priceResult(800), false)
	if err != nil || len(errs) != 1 || first.Pending != 1 || first.Uncertain != 0 || terminal != 1 || email != 1 {
		t.Fatalf("first %+v %v %v terminal=%d email=%d", first, errs, err, terminal, email)
	}
	ws, err = disk.Load()
	if err != nil {
		t.Fatal(err)
	}
	if ws.Watches[0].LastLowestPrice != 800 || ws.Watches[0].PendingAlerts[0].Alert.LowestPrice != 800 {
		t.Fatal("observation and pending payload not persisted")
	}
	send = func(d model.AlertDelivery, a model.Alert) error {
		if d.Channel == "terminal" {
			terminal++
		} else {
			email++
		}
		if a.LowestPrice != 800 {
			t.Fatalf("changed payload %+v", a)
		}
		return nil
	}
	second, errs, err := run(priceResult(800), false)
	if err != nil || len(errs) != 0 || second.Triggered != 0 || second.Recovered != 1 || second.Pending != 0 || terminal != 1 || email != 2 {
		t.Fatalf("recovery %+v %v %v terminal=%d email=%d", second, errs, err, terminal, email)
	}
	ws, err = disk.Load()
	if err != nil || len(ws.Watches[0].PendingAlerts) != 0 {
		t.Fatalf("completed state %+v %v", ws, err)
	}
}

func TestUncertainDeliveryRequiresExplicitRetry(t *testing.T) {
	disk := watcher.Store{Path: filepath.Join(t.TempDir(), "watches.json")}
	ws := recoveryWatch()
	ws.Watches[0].TargetPrice = 850
	calls := map[string]int{}
	send := func(d model.AlertDelivery, a model.Alert) error {
		calls[d.Channel]++
		if d.Channel == "email" {
			return errors.New("lost response")
		}
		return nil
	}
	run := func(search watchSearchFunc, retry bool) (watchRunReport, []string, error) {
		return runWatchPass(ws.Watches, "", true, search, send, func() error { return disk.Save(ws) }, retry, time.Now(), false, nil)
	}
	first, _, err := run(priceResult(800), false)
	if err != nil || first.Uncertain != 1 {
		t.Fatalf("first %+v %v", first, err)
	}
	ws, err = disk.Load()
	if err != nil {
		t.Fatal(err)
	}
	held, _, err := run(priceResult(800), false)
	if err != nil || held.Triggered != 0 || held.Uncertain != 1 || calls["email"] != 1 || calls["terminal"] != 1 {
		t.Fatalf("automatic replay %+v calls=%v err=%v", held, calls, err)
	}
	send = func(d model.AlertDelivery, a model.Alert) error { calls[d.Channel]++; return nil }
	recovered, _, err := run(func(model.SearchQuery) (model.SearchResult, error) {
		return model.SearchResult{}, errors.New("provider unavailable")
	}, true)
	if err != nil || recovered.ProviderFailures != 1 || recovered.Recovered != 1 || recovered.Pending != 0 || calls["email"] != 2 || calls["terminal"] != 1 {
		t.Fatalf("explicit recovery %+v calls=%v err=%v", recovered, calls, err)
	}
}

func TestNewPriceDoesNotReplacePendingAlert(t *testing.T) {
	ws := recoveryWatch()
	ws.Watches[0].NotifyTerminal = false
	send := func(model.AlertDelivery, model.Alert) error { return notify.ErrNotDispatched }
	for _, price := range []int{800, 750} {
		if _, _, err := runWatchPass(ws.Watches, "", true, priceResult(price), send, func() error { return nil }, false, time.Now(), false, nil); err != nil {
			t.Fatal(err)
		}
	}
	pending := ws.Watches[0].PendingAlerts
	if len(pending) != 2 || pending[0].Alert.LowestPrice != 800 || pending[1].Alert.LowestPrice != 750 || ws.Watches[0].LastLowestPrice != 750 {
		t.Fatalf("lost original alert %+v", ws)
	}
}

func TestDeliveryCheckpointFailure(t *testing.T) {
	// Save 1 records observation; 2 reserves terminal send; 3 records its result.
	for _, failAt := range []int{1, 2, 3, 5} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			disk := watcher.Store{Path: filepath.Join(t.TempDir(), "watches.json")}
			ws := recoveryWatch()
			if err := disk.Save(ws); err != nil {
				t.Fatal(err)
			}
			saves, calls := 0, 0
			_, _, err := runWatchPass(ws.Watches, "", true, priceResult(800), func(model.AlertDelivery, model.Alert) error { calls++; return nil }, func() error {
				saves++
				if saves == failAt {
					return errors.New("disk failure")
				}
				return disk.Save(ws)
			}, false, time.Now(), false, nil)
			expected := 0
			if failAt == 3 {
				expected = 1
			}
			if failAt == 5 {
				expected = 2
			}
			if err == nil || calls != expected {
				t.Fatalf("err=%v calls=%d expected=%d", err, calls, expected)
			}
			ws, err = disk.Load()
			if err != nil {
				t.Fatal(err)
			}
			if failAt >= 3 {
				replayed := []string{}
				report, _, err := runWatchPass(ws.Watches, "", true, priceResult(800), func(d model.AlertDelivery, a model.Alert) error { replayed = append(replayed, d.Channel); return nil }, func() error { return disk.Save(ws) }, false, time.Now(), false, nil)
				if err != nil || report.Uncertain != 1 {
					t.Fatalf("lost send not held %+v %v", report, err)
				}
				if failAt == 3 && (len(replayed) != 1 || replayed[0] != "email") {
					t.Fatalf("terminal repeated %v", replayed)
				}
				if failAt == 5 && len(replayed) != 0 {
					t.Fatalf("completed channels repeated %v", replayed)
				}
			}
		})
	}
}

func TestUnknownDeliveryStatusFailsClosed(t *testing.T) {
	ws := recoveryWatch()
	ws.Watches[0].PendingAlerts = []model.PendingAlert{{Alert: model.Alert{LowestPrice: 800}, Deliveries: []model.AlertDelivery{{Channel: "email", Status: "future"}}}}
	calls := 0
	r, _, err := runWatchPass(ws.Watches, "", true, priceResult(900), func(model.AlertDelivery, model.Alert) error { calls++; return nil }, func() error { return nil }, true, time.Now(), false, nil)
	if err != nil || calls != 0 || r.Uncertain != 1 {
		t.Fatalf("unknown status %+v calls=%d err=%v", r, calls, err)
	}
}

func TestCLIUncertainWebhookRecovery(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	calls := 0
	status := http.StatusServiceUnavailable
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(status) }))
	defer server.Close()
	app := NewApp("test")
	_, _, code, _ := runCLIWithCapture(t, app, []string{"--json", "auth", "login", "--provider", "google-url"})
	if code != ExitSuccess {
		t.Fatal(code)
	}
	stateDir := t.TempDir()
	disk := watcher.Store{Path: filepath.Join(stateDir, "watches.json")}
	ws := recoveryWatch()
	w := &ws.Watches[0]
	w.NotifyTerminal = false
	w.NotifyEmail = false
	w.NotifyWebhook = true
	w.WebhookURL = server.URL
	w.Query = model.SearchQuery{From: "SFO", To: "ATH", Depart: "2026-12-01"}
	w.PendingAlerts = []model.PendingAlert{newPendingAlert(*w, model.Alert{WatchID: w.ID, LowestPrice: 800, Currency: "USD"})}
	if err := disk.Save(ws); err != nil {
		t.Fatal(err)
	}
	args := []string{"--state-dir", stateDir, "--json", "watch", "run", "--id", "w1"}
	for pass := 0; pass < 3; pass++ {
		if pass == 2 {
			status = http.StatusOK
			args = append(args, "--retry-uncertain")
		}
		stdout, stderr, code, errText := runCLIWithCapture(t, app, args)
		var report watchRunReport
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("invalid report %s: %v", stdout, err)
		}
		if pass < 2 {
			if code != ExitNotifyFailure || calls != 1 || report.Uncertain != 1 || report.Pending != 1 {
				t.Fatalf("held pass=%d code=%d calls=%d report=%+v stderr=%s err=%s", pass, code, calls, report, stderr, errText)
			}
		} else if code != ExitSuccess || calls != 2 || report.Recovered != 1 || report.Pending != 0 {
			t.Fatalf("recovery code=%d calls=%d report=%+v err=%s", code, calls, report, errText)
		}
	}
}

func TestNotifyTerminalReportsWriteFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	err = NewApp("test").Run([]string{"notify", "test", "--channel", "terminal"})
	if err == nil {
		t.Fatal("failed terminal write reported success")
	}
}
