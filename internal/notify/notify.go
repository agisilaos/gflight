package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

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

func (n Notifier) SendWebhook(url string, alert model.Alert) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("missing webhook url")
	}
	payload, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
