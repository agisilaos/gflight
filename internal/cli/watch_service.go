package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

type watchSearchFunc func(model.SearchQuery) (model.SearchResult, error)
type watchNotifyFunc func(model.AlertDelivery, model.Alert) error

type watchRunReport struct {
	Pending          int           `json:"pending"`
	Uncertain        int           `json:"uncertain"`
	Recovered        int           `json:"recovered"`
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
	send watchNotifyFunc,
	checkpoint func() error,
	retryUncertain bool,
	now time.Time,
	verbose bool,
	errw io.Writer,
) (watchRunReport, []string, error) {
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
		oldPending := len(w.PendingAlerts)
		res, err := search(w.Query)
		if err != nil {
			report.ProviderFailures++
			if verbose && errw != nil {
				fmt.Fprintf(errw, "watch %s failed: %v\n", w.ID, err)
			}
		} else if alert, triggered := evaluateWatchResult(w, res, now); triggered && !hasPendingPrice(*w, alert) {
			report.Triggered++
			report.Alerts = append(report.Alerts, alert)
			w.PendingAlerts = append(w.PendingAlerts, newPendingAlert(*w, alert))
		}
		// An interrupted send is ambiguous, even if the process never saw its result.
		for j := range w.PendingAlerts {
			for k := range w.PendingAlerts[j].Deliveries {
				d := &w.PendingAlerts[j].Deliveries[k]
				if d.Status == "in_flight" {
					d.Status = "uncertain"
				}
			}
		}
		if err := checkpoint(); err != nil {
			return report, notifyErrs, err
		}
		remaining := make([]model.PendingAlert, 0, len(w.PendingAlerts))
		for j := range w.PendingAlerts {
			pending := &w.PendingAlerts[j]
			failed := false
			for k := range pending.Deliveries {
				d := &pending.Deliveries[k]
				if d.Status == "delivered" {
					continue
				}
				if d.Status != "pending" && !(d.Status == "uncertain" && retryUncertain) {
					failed = true
					report.Uncertain++
					notifyErrs = append(notifyErrs, fmt.Sprintf("watch %s %s outcome uncertain; inspect delivery before --retry-uncertain (may duplicate)", w.ID, d.Channel))
					continue
				}
				d.Status = "in_flight"
				if err := checkpoint(); err != nil {
					return report, notifyErrs, err
				}
				err := send(*d, pending.Alert)
				d.Status = "delivered"
				if err != nil {
					failed = true
					d.Status = "uncertain"
					if errors.Is(err, notify.ErrNotDispatched) {
						d.Status = "pending"
					} else {
						report.Uncertain++
					}
					notifyErrs = append(notifyErrs, fmt.Sprintf("watch %s %s delivery %s: %v", w.ID, d.Channel, d.Status, err))
				}
				if err := checkpoint(); err != nil {
					return report, notifyErrs, err
				}
			}
			if failed {
				report.NotifyFailures++
				remaining = append(remaining, *pending)
			} else if j < oldPending {
				report.Recovered++
			}
		}
		w.PendingAlerts = remaining
		report.Pending += len(remaining)
		if err := checkpoint(); err != nil {
			return report, notifyErrs, err
		}
	}
	return report, notifyErrs, nil
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
