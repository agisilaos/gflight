package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

type watchSearchFunc func(model.SearchQuery) (model.SearchResult, error)
type watchNotifyFunc func(model.Watch, model.Alert) error

type watchRunReport struct {
	Evaluated        int           `json:"evaluated"`
	Triggered        int           `json:"triggered"`
	ProviderFailures int           `json:"provider_failures"`
	NotifyFailures   int           `json:"notify_failures"`
	Alerts           []model.Alert `json:"alerts"`
}

func runWatchPass(
	watches []model.Watch,
	watchID string,
	runAll bool,
	search watchSearchFunc,
	notify watchNotifyFunc,
	now time.Time,
	verbose bool,
	errw io.Writer,
) (watchRunReport, []string) {
	report := watchRunReport{
		Alerts: make([]model.Alert, 0),
	}
	notifyErrs := make([]string, 0)

	for i := range watches {
		w := &watches[i]
		if !shouldRunWatch(*w, watchID, runAll) {
			continue
		}
		report.Evaluated++
		res, err := search(w.Query)
		if err != nil {
			report.ProviderFailures++
			if verbose && errw != nil {
				fmt.Fprintf(errw, "watch %s failed: %v\n", w.ID, err)
			}
			continue
		}
		alert, triggered := evaluateWatchResult(w, res, now)
		if !triggered {
			continue
		}
		report.Triggered++
		report.Alerts = append(report.Alerts, alert)
		if err := notify(*w, alert); err != nil {
			notifyErrs = append(notifyErrs, err.Error())
			report.NotifyFailures++
		}
	}
	return report, notifyErrs
}

func shouldRunWatch(w model.Watch, watchID string, runAll bool) bool {
	if !w.Enabled {
		return false
	}
	if watchID != "" {
		return w.ID == watchID
	}
	return runAll
}

func evaluateWatchResult(w *model.Watch, res model.SearchResult, now time.Time) (model.Alert, bool) {
	lowest := 0
	currency := "USD"
	if len(res.Flights) > 0 {
		lowest = res.Flights[0].Price
		currency = res.Flights[0].Currency
	}

	reason := ""
	if w.TargetPrice > 0 && lowest > 0 && lowest <= w.TargetPrice {
		reason = fmt.Sprintf("price reached target <= %d", w.TargetPrice)
	} else if w.LastLowestPrice > 0 && lowest > 0 && lowest < w.LastLowestPrice {
		reason = fmt.Sprintf("price dropped from %d to %d", w.LastLowestPrice, lowest)
	}

	w.LastRunAt = now.UTC()
	if lowest > 0 {
		w.LastLowestPrice = lowest
	}
	w.UpdatedAt = now.UTC()

	if reason == "" {
		return model.Alert{}, false
	}

	return model.Alert{
		WatchID:     w.ID,
		WatchName:   w.Name,
		TriggeredAt: now.UTC(),
		Reason:      reason,
		LowestPrice: lowest,
		Currency:    currency,
		URL:         res.URL,
	}, true
}

func shouldReturnProviderFailure(report watchRunReport, strict bool) bool {
	if report.Evaluated == 0 {
		return false
	}
	if strict && report.ProviderFailures > 0 {
		return true
	}
	return report.ProviderFailures == report.Evaluated
}
