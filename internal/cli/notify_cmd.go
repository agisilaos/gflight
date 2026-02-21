package cli

import (
	"flag"
	"os"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

func (a App) cmdNotify(g globalFlags, args []string) error {
	if len(args) == 0 || args[0] != "test" {
		return newExitError(ExitInvalidUsage, "notify supports only: notify test")
	}
	fs := flag.NewFlagSet("notify test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	channel := fs.String("channel", "terminal", "terminal|email|webhook")
	to := fs.String("to", "", "Email recipient")
	url := fs.String("url", "", "Webhook URL override")
	if err := fs.Parse(args[1:]); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	n := newDefaultNotifyDispatcher(notify.Notifier{Config: cfg})
	alert := model.Alert{
		WatchID:     "test",
		WatchName:   "test-notification",
		TriggeredAt: time.Now().UTC(),
		Reason:      "notification test",
		LowestPrice: 499,
		Currency:    "USD",
		URL:         "https://www.google.com/travel/flights",
	}
	switch *channel {
	case "terminal":
		n.SendTerminal(alert)
		return writeMaybeJSON(g, map[string]any{"ok": true, "channel": "terminal"})
	case "email":
		recipient := *to
		if err := validateNotifyEmailRuntime(cfg, recipient); err != nil {
			return wrapExitError(ExitNotifyFailure, err)
		}
		if recipient == "" {
			recipient = cfg.DefaultNotifyEmail
		}
		if err := n.SendEmail(recipient, alert); err != nil {
			return wrapExitError(ExitNotifyFailure, err)
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "channel": "email", "to": recipient})
	case "webhook":
		webhookURL := *url
		if err := validateNotifyWebhookRuntime(cfg, webhookURL); err != nil {
			return wrapExitError(ExitNotifyFailure, err)
		}
		if webhookURL == "" {
			webhookURL = cfg.WebhookURL
		}
		if err := n.SendWebhook(webhookURL, alert); err != nil {
			return wrapExitError(ExitNotifyFailure, err)
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "channel": "webhook", "url": webhookURL})
	default:
		return newExitError(ExitInvalidUsage, "--channel must be terminal, email, or webhook")
	}
}
