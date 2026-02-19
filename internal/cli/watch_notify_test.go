package cli

import (
	"testing"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
)

func TestSendWatchNotificationsReturnsEmailError(t *testing.T) {
	app := NewApp("test")
	n := notify.Notifier{Config: config.Config{}}
	w := model.Watch{
		ID:          "w_1",
		NotifyEmail: true,
		EmailTo:     "alerts@example.com",
	}
	alert := model.Alert{WatchID: "w_1", WatchName: "test", TriggeredAt: time.Now().UTC()}

	err := app.sendWatchNotifications(n, w, alert)
	if err == nil {
		t.Fatalf("expected email notification error")
	}
}
