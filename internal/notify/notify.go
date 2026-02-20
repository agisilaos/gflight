package notify

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
)

type Notifier struct {
	Config config.Config
}

func (n Notifier) SendTerminal(alert model.Alert) {
	fmt.Fprintf(os.Stderr, "ALERT %s (%s): %s. Lowest price: %d %s\n%s\n",
		alert.WatchName,
		alert.WatchID,
		alert.Reason,
		alert.LowestPrice,
		alert.Currency,
		alert.URL,
	)
}

func (n Notifier) SendEmail(to string, alert model.Alert) error {
	if n.Config.SMTPHost == "" || n.Config.SMTPUsername == "" || n.Config.SMTPPassword == "" || n.Config.SMTPSender == "" {
		return fmt.Errorf("email not configured: set smtp_host/smtp_username/smtp_password/smtp_sender")
	}
	if to == "" {
		return fmt.Errorf("missing email recipient")
	}
	addr := fmt.Sprintf("%s:%d", n.Config.SMTPHost, n.Config.SMTPPort)
	auth := smtp.PlainAuth("", n.Config.SMTPUsername, n.Config.SMTPPassword, n.Config.SMTPHost)
	subject := fmt.Sprintf("gflight alert: %s", alert.WatchName)
	body := fmt.Sprintf("Reason: %s\nLowest price: %d %s\nGoogle Flights: %s\nTriggered at: %s\n",
		alert.Reason,
		alert.LowestPrice,
		alert.Currency,
		alert.URL,
		alert.TriggeredAt.Format("2006-01-02 15:04:05 MST"),
	)
	msg := strings.Join([]string{
		"From: " + n.Config.SMTPSender,
		"To: " + to,
		"Subject: " + subject,
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, n.Config.SMTPSender, []string{to}, []byte(msg))
}
