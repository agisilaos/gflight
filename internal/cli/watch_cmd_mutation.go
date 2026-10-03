package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
)

type watchCreateOptions struct {
	name                                       string
	target                                     int
	notifyTerminal, notifyEmail, notifyWebhook bool
	emailTo, webhookURL                        string
	dryRun                                     bool
}

func newWatchCreateFlagSet() (*flag.FlagSet, *model.SearchQuery, *watchCreateOptions) {
	fs, q := newSearchFlagSet("watch create")
	opts := &watchCreateOptions{}
	fs.StringVar(&opts.name, "name", "", "Watch name (default: FROM-TO-DEPART)")
	fs.IntVar(&opts.target, "target-price", 0, "Alert when price <= target; 0 alerts on price drops")
	fs.BoolVar(&opts.notifyTerminal, "notify-terminal", true, "Send terminal notifications")
	fs.BoolVar(&opts.notifyEmail, "notify-email", false, "Send email notifications")
	fs.BoolVar(&opts.notifyWebhook, "notify-webhook", false, "Send webhook notifications")
	fs.StringVar(&opts.emailTo, "email-to", "", "Email recipient (default: configured notify_email)")
	fs.StringVar(&opts.webhookURL, "webhook-url", "", "Webhook URL override (default: configured webhook_url)")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "Preview watch without saving")
	return fs, q, opts
}

func (a App) cmdWatchCreate(g globalFlags, args []string) error {
	fs, q, opts := newWatchCreateFlagSet()
	if err := parseNamedFlags(fs, args); err != nil {
		return err
	}
	if err := validateQuery(*q); err != nil {
		return newExitError(ExitInvalidUsage, "%v\nSee: gflight help watch create", err)
	}
	if opts.name == "" {
		opts.name = fmt.Sprintf("%s-%s-%s", q.From, q.To, q.Depart)
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if opts.emailTo == "" {
		opts.emailTo = cfg.DefaultNotifyEmail
	}
	if opts.webhookURL == "" {
		opts.webhookURL = cfg.WebhookURL
	}
	w := model.Watch{
		ID:             fmt.Sprintf("w_%d", time.Now().UnixNano()),
		Name:           opts.name,
		Query:          *q,
		Enabled:        true,
		TargetPrice:    opts.target,
		NotifyTerminal: opts.notifyTerminal,
		NotifyEmail:    opts.notifyEmail,
		NotifyWebhook:  opts.notifyWebhook,
		EmailTo:        opts.emailTo,
		WebhookURL:     opts.webhookURL,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if opts.dryRun {
		return a.writeMaybeJSON(g, w)
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
		a.writePlainKV("watch_id", w.ID)
		return nil
	}
	return a.writeMaybeJSON(g, w)
}

func (a App) cmdWatchList(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := parseNamedFlags(fs, args); err != nil {
		return err
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	ws, err := store.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	sort.Slice(ws.Watches, func(i, j int) bool {
		if ws.Watches[i].CreatedAt.Equal(ws.Watches[j].CreatedAt) {
			return ws.Watches[i].ID < ws.Watches[j].ID
		}
		return ws.Watches[i].CreatedAt.After(ws.Watches[j].CreatedAt)
	})
	if g.JSON {
		return a.writeJSON(ws.Watches)
	}
	if len(ws.Watches) == 0 {
		fmt.Fprintln(a.output(), "No watches configured")
		return nil
	}
	if g.Plain {
		a.writePlainTableHeader("id", "name", "enabled", "target_price", "from", "to", "depart")
	}
	for _, w := range ws.Watches {
		if g.Plain {
			a.writePlainTableRow(
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
		fmt.Fprintf(a.output(), "%s\t%s\t%s->%s\t%s\ttarget=%d\tenabled=%t\n", w.ID, w.Name, w.Query.From, w.Query.To, w.Query.Depart, w.TargetPrice, w.Enabled)
	}
	return nil
}

func (a App) cmdWatchSetEnabled(g globalFlags, args []string, enabled bool) error {
	fs := flag.NewFlagSet("watch set-enabled", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	if err := parseNamedFlags(fs, args); err != nil {
		return err
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
			a.writePlainKV("watch_id", ws.Watches[i].ID, "enabled", strconv.FormatBool(ws.Watches[i].Enabled))
			return nil
		}
		return a.writeMaybeJSON(g, ws.Watches[i])
	}
	return newExitError(ExitGenericFailure, "watch not found: %s", *id)
}

type watchDeleteOptions struct {
	id, confirm string
	force       bool
}

func newWatchDeleteFlagSet() (*flag.FlagSet, *watchDeleteOptions) {
	fs := flag.NewFlagSet("watch delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := &watchDeleteOptions{}
	fs.StringVar(&opts.id, "id", "", "Watch ID")
	fs.BoolVar(&opts.force, "force", false, "Delete without confirmation")
	fs.StringVar(&opts.confirm, "confirm", "", "Confirmation token (watch ID)")
	return fs, opts
}

func (a App) cmdWatchDelete(g globalFlags, args []string) error {
	fs, opts := newWatchDeleteFlagSet()
	if err := parseNamedFlags(fs, args); err != nil {
		return err
	}
	if opts.id == "" {
		return newExitError(ExitInvalidUsage, "--id is required\nSee: gflight help watch delete")
	}
	if !opts.force && opts.confirm != opts.id {
		return newExitError(ExitInvalidUsage, "destructive action: pass --force or --confirm with the watch ID")
	}
	if g.NoInput && !opts.force {
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
		if w.ID == opts.id {
			found = true
			continue
		}
		filtered = append(filtered, w)
	}
	if !found {
		return newExitError(ExitGenericFailure, "watch not found: %s", opts.id)
	}
	ws.Watches = filtered
	if err := store.Save(ws); err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if g.Plain && !g.JSON {
		a.writePlainKV("deleted_id", opts.id)
		return nil
	}
	return a.writeMaybeJSON(g, map[string]any{"deleted": opts.id})
}
