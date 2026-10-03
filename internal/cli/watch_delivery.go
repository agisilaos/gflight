package cli

import (
	"fmt"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

func newPendingAlert(w model.Watch, alert model.Alert) model.PendingAlert {
	p := model.PendingAlert{Alert: alert}
	if w.NotifyTerminal {
		p.Deliveries = append(p.Deliveries, model.AlertDelivery{Channel: "terminal", Status: model.DeliveryPending})
	}
	if w.NotifyEmail {
		p.Deliveries = append(p.Deliveries, model.AlertDelivery{Channel: "email", Destination: w.EmailTo, Status: model.DeliveryPending})
	}
	if w.NotifyWebhook {
		p.Deliveries = append(p.Deliveries, model.AlertDelivery{Channel: "webhook", Destination: w.WebhookURL, Status: model.DeliveryPending})
	}
	return p
}

func hasPendingPrice(w model.Watch, alert model.Alert) bool {
	for _, p := range w.PendingAlerts {
		if p.Alert.LowestPrice == alert.LowestPrice && p.Alert.Currency == alert.Currency {
			return true
		}
	}
	return false
}

func sendAlertDelivery(n notifyDispatcher, d model.AlertDelivery, alert model.Alert) error {
	switch d.Channel {
	case "terminal":
		return n.SendTerminal(alert)
	case "email":
		return n.SendEmail(d.Destination, alert)
	case "webhook":
		return n.SendWebhook(d.Destination, alert)
	default:
		return fmt.Errorf("%w: unknown notification channel", notify.ErrNotDispatched)
	}
}
