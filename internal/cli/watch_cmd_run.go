package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

func (a App) cmdWatchRun(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	watchID := fs.String("id", "", "Watch ID")
	runAll := fs.Bool("all", false, "Run all watches")
	failOnProviderErrors := fs.Bool("fail-on-provider-errors", false, "Exit non-zero when any provider failure occurs")
	once := fs.Bool("once", true, "Single pass")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	_ = once
	if (*watchID == "" && !*runAll) || (*watchID != "" && *runAll) {
		return newExitError(ExitInvalidUsage, "watch run requires exactly one of --all or --id")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if err := validateProviderRuntime(cfg); err != nil {
		return wrapValidationError(err)
	}
	p, err := a.resolveProvider(cfg, g)
	if err != nil {
		return err
	}
	n := newDefaultNotifyDispatcher(notify.Notifier{Config: cfg})
	report, notifyErrs := runWatchPass(
		ws.Watches,
		*watchID,
		*runAll,
		p.Search,
		func(w model.Watch, alert model.Alert) error { return a.sendWatchNotifications(n, w, alert) },
		time.Now().UTC(),
		g.Verbose,
		os.Stderr,
	)
	if err := store.Save(ws); err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if g.JSON {
		if err := writeJSON(report); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
	}
	if !g.JSON {
		fmt.Printf(
			"Watch run summary: evaluated=%d triggered=%d provider_failures=%d notify_failures=%d\n",
			report.Evaluated,
			report.Triggered,
			report.ProviderFailures,
			report.NotifyFailures,
		)
	}
	if len(notifyErrs) > 0 {
		return newExitError(ExitNotifyFailure, "%s", strings.Join(notifyErrs, "; "))
	}
	if shouldReturnProviderFailure(report, *failOnProviderErrors) {
		if *failOnProviderErrors && report.ProviderFailures > 0 {
			return newExitError(ExitProviderFailure, "provider failures occurred (%d/%d)", report.ProviderFailures, report.Evaluated)
		}
		return newExitError(ExitProviderFailure, "all provider requests failed (%d/%d)", report.ProviderFailures, report.Evaluated)
	}
	if report.Triggered == 0 {
		if !g.JSON {
			fmt.Println("No alerts triggered")
		}
		return nil
	}
	if !g.JSON {
		fmt.Printf("Triggered %d alert(s)\n", report.Triggered)
	}
	return nil
}

func (a App) sendWatchNotifications(n notifyDispatcher, w model.Watch, alert model.Alert) error {
	notifyErrs := make([]string, 0)
	if w.NotifyTerminal {
		n.SendTerminal(alert)
	}
	if w.NotifyEmail {
		if err := n.SendEmail(w.EmailTo, alert); err != nil {
			notifyErrs = append(notifyErrs, fmt.Sprintf("watch %s email failed: %v", w.ID, err))
		}
	}
	if w.NotifyWebhook {
		if err := n.SendWebhook(w.WebhookURL, alert); err != nil {
			notifyErrs = append(notifyErrs, fmt.Sprintf("watch %s webhook failed: %v", w.ID, err))
		}
	}
	if len(notifyErrs) > 0 {
		return newExitError(ExitNotifyFailure, "%s", strings.Join(notifyErrs, "; "))
	}
	return nil
}

func (a App) cmdWatchTest(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if *id == "" {
		return newExitError(ExitInvalidUsage, "--id is required")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	for _, w := range ws.Watches {
		if w.ID != *id {
			continue
		}
		alert := model.Alert{
			WatchID:     w.ID,
			WatchName:   w.Name,
			TriggeredAt: time.Now().UTC(),
			Reason:      "manual test",
			LowestPrice: w.TargetPrice,
			Currency:    firstOr(w.Query.Currency, "USD"),
			URL:         "https://www.google.com/travel/flights",
		}
		cfg, _ := config.Load()
		n := newDefaultNotifyDispatcher(notify.Notifier{Config: cfg})
		if err := a.sendWatchNotifications(n, w, alert); err != nil {
			return newExitError(ExitNotifyFailure, "%v", err)
		}
		return writeMaybeJSON(g, alert)
	}
	return newExitError(ExitGenericFailure, "watch not found: %s", *id)
}
