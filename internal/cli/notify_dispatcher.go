package cli

import (
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

type notifyDispatcher interface {
	SendTerminal(alert model.Alert)
	SendEmail(to string, alert model.Alert) error
	SendWebhook(url string, alert model.Alert) error
}

type defaultNotifyDispatcher struct {
	n notify.Notifier
}

func newDefaultNotifyDispatcher(n notify.Notifier) notifyDispatcher {
	return defaultNotifyDispatcher{n: n}
}

func (d defaultNotifyDispatcher) SendTerminal(alert model.Alert) {
	d.n.SendTerminal(alert)
}

func (d defaultNotifyDispatcher) SendEmail(to string, alert model.Alert) error {
	return d.n.SendEmail(to, alert)
}

func (d defaultNotifyDispatcher) SendWebhook(url string, alert model.Alert) error {
	return d.n.SendWebhook(url, alert)
}
