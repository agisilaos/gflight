package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

type watchSearchFunc func(model.SearchQuery) (model.SearchResult, error)
type watchNotifyFunc func(model.Watch, model.Alert) error

func runWatchPass(
	watches []model.Watch,
	watchID string,
	runAll bool,
	search watchSearchFunc,
	notify watchNotifyFunc,
	now time.Time,
	verbose bool,
	errw io.Writer,
) ([]model.Alert, []string) {
	alerts := make([]model.Alert, 0)
	notifyErrs := make([]string, 0)

	for i := range watches {
		w := &watches[i]
		if !shouldRunWatch(*w, watchID, runAll) {
			continue
		}
		res, err := search(w.Query)
		if err != nil {
			if verbose && errw != nil {
				fmt.Fprintf(errw, "watch %s failed: %v\n", w.ID, err)
			}
			continue
		}
		alert, triggered := evaluateWatchResult(w, res, now)
		if !triggered {
			continue
		}
		alerts = append(alerts, alert)
		if err := notify(*w, alert); err != nil {
			notifyErrs = append(notifyErrs, err.Error())
		}
	}
	return alerts, notifyErrs
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
