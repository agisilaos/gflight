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
