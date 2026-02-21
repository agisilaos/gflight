package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
	"github.com/agisilaos/gflight/internal/watcher"
)

func (a App) watcherStore(stateOverride string) (watcher.Store, error) {
	dir, err := config.StateDir(stateOverride)
	if err != nil {
		return watcher.Store{}, err
	}
	return watcher.Store{Path: filepath.Join(dir, "watches.json")}, nil
}

func (a App) cmdWatch(g globalFlags, args []string) error {
	if len(args) == 0 {
		return newExitError(ExitInvalidUsage, "watch requires subcommand: create|list|enable|disable|delete|run|test")
	}
	sub := args[0]
	argv := args[1:]
	switch sub {
	case "create":
		return a.cmdWatchCreate(g, argv)
	case "list":
		return a.cmdWatchList(g, argv)
	case "enable":
		return a.cmdWatchSetEnabled(g, argv, true)
	case "disable":
		return a.cmdWatchSetEnabled(g, argv, false)
	case "delete":
		return a.cmdWatchDelete(g, argv)
	case "run":
		return a.cmdWatchRun(g, argv)
	case "test":
		return a.cmdWatchTest(g, argv)
	default:
		return newExitError(ExitInvalidUsage, "unknown watch subcommand %q", sub)
	}
}

func (a App) cmdWatchCreate(g globalFlags, args []string) error {
	fs, q := newSearchFlagSet("watch create")
	name := fs.String("name", "", "Watch name")
	target := fs.Int("target-price", 0, "Alert when price <= target")
	notifyTerminal := fs.Bool("notify-terminal", true, "Send terminal notifications")
	notifyEmail := fs.Bool("notify-email", false, "Send email notifications")
	notifyWebhook := fs.Bool("notify-webhook", false, "Send webhook notifications")
	emailTo := fs.String("email-to", "", "Email recipient")
	webhookURL := fs.String("webhook-url", "", "Webhook URL override")
	dryRun := fs.Bool("dry-run", false, "Preview watch without saving")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if err := validateQuery(*q); err != nil {
		return err
	}
	if *name == "" {
		*name = fmt.Sprintf("%s-%s-%s", q.From, q.To, q.Depart)
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if *emailTo == "" {
		*emailTo = cfg.DefaultNotifyEmail
	}
	if *webhookURL == "" {
		*webhookURL = cfg.WebhookURL
	}
	w := model.Watch{
		ID:             fmt.Sprintf("w_%d", time.Now().UnixNano()),
		Name:           *name,
		Query:          *q,
		Enabled:        true,
		TargetPrice:    *target,
		NotifyTerminal: *notifyTerminal,
		NotifyEmail:    *notifyEmail,
		NotifyWebhook:  *notifyWebhook,
		EmailTo:        *emailTo,
		WebhookURL:     *webhookURL,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if *dryRun {
		return writeMaybeJSON(g, w)
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws.Watches = append(ws.Watches, w)
	if err := store.Save(ws); err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if g.Plain && !g.JSON {
		writePlainKV("watch_id", w.ID)
		return nil
	}
	return writeMaybeJSON(g, w)
}

func (a App) cmdWatchList(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	sort.Slice(ws.Watches, func(i, j int) bool { return ws.Watches[i].CreatedAt.After(ws.Watches[j].CreatedAt) })
	if g.JSON {
		return writeJSON(ws.Watches)
	}
	if len(ws.Watches) == 0 {
		fmt.Println("No watches configured")
		return nil
	}
	if g.Plain {
		writePlainTableHeader("id", "name", "enabled", "target_price", "from", "to", "depart")
	}
	for _, w := range ws.Watches {
		if g.Plain {
			writePlainTableRow(
				w.ID,
				w.Name,
				strconv.FormatBool(w.Enabled),
				strconv.Itoa(w.TargetPrice),
				w.Query.From,
				w.Query.To,
				w.Query.Depart,
			)
			continue
		}
		fmt.Printf("%s\t%s\t%s->%s\t%s\ttarget=%d\tenabled=%t\n", w.ID, w.Name, w.Query.From, w.Query.To, w.Query.Depart, w.TargetPrice, w.Enabled)
	}
	return nil
}

func (a App) cmdWatchSetEnabled(g globalFlags, args []string, enabled bool) error {
	fs := flag.NewFlagSet("watch set-enabled", flag.ContinueOnError)
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
	for i := range ws.Watches {
		if ws.Watches[i].ID != *id {
			continue
		}
		ws.Watches[i].Enabled = enabled
		ws.Watches[i].UpdatedAt = time.Now().UTC()
		if err := store.Save(ws); err != nil {
			return wrapExitError(ExitGenericFailure, err)
		}
		if g.Plain && !g.JSON {
			writePlainKV("watch_id", ws.Watches[i].ID, "enabled", strconv.FormatBool(ws.Watches[i].Enabled))
			return nil
		}
		return writeMaybeJSON(g, ws.Watches[i])
	}
	return newExitError(ExitGenericFailure, "watch not found: %s", *id)
}

func (a App) cmdWatchDelete(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	force := fs.Bool("force", false, "Delete without confirmation")
	confirm := fs.String("confirm", "", "Confirmation token (watch ID)")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if *id == "" {
		return newExitError(ExitInvalidUsage, "--id is required")
	}
	if !*force && *confirm != *id {
		return newExitError(ExitInvalidUsage, "destructive action: pass --force or --confirm with the watch ID")
	}
	if g.NoInput && !*force {
		return newExitError(ExitInvalidUsage, "--no-input requires --force for watch delete")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	filtered := make([]model.Watch, 0, len(ws.Watches))
	found := false
	for _, w := range ws.Watches {
		if w.ID == *id {
			found = true
			continue
		}
		filtered = append(filtered, w)
	}
	if !found {
		return newExitError(ExitGenericFailure, "watch not found: %s", *id)
	}
	ws.Watches = filtered
	if err := store.Save(ws); err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if g.Plain && !g.JSON {
		writePlainKV("deleted_id", *id)
		return nil
	}
	return writeMaybeJSON(g, map[string]any{"deleted": *id})
}

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
