package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/safeerror"
)

// ErrNotDispatched means validation stopped delivery before any external send.
var ErrNotDispatched = errors.New("notification not dispatched")

type Notifier struct {
	Config config.Config
}

func (n Notifier) SendTerminal(alert model.Alert) error {
	_, err := fmt.Fprintf(os.Stderr, "ALERT %s (%s): %s. Lowest price: %d %s\n%s\n",
		alert.WatchName,
		alert.WatchID,
		alert.Reason,
		alert.LowestPrice,
		alert.Currency,
		alert.URL,
	)
	return err
}

func (n Notifier) SendEmail(to string, alert model.Alert) error {
	if n.Config.SMTPHost == "" || n.Config.SMTPUsername == "" || n.Config.SMTPPassword == "" || n.Config.SMTPSender == "" {
		return fmt.Errorf("%w: email not configured: set smtp_host/smtp_username/smtp_password/smtp_sender", ErrNotDispatched)
	}
	if to == "" {
		return fmt.Errorf("%w: missing email recipient", ErrNotDispatched)
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
	return n.sendWebhookWithClient(url, alert, &http.Client{Timeout: 10 * time.Second})
}

func (n Notifier) sendWebhookWithClient(url string, alert model.Alert, client *http.Client) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("%w: missing webhook url", ErrNotDispatched)
	}
	parsed, parseErr := neturl.Parse(url)
	if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%w: invalid webhook request URL", ErrNotDispatched)
	}
	payload, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return &safeerror.Error{Message: "invalid webhook request URL", Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &safeerror.Error{Message: classifyWebhookRequestError(err), Cause: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return fmt.Errorf("webhook authorization failed: HTTP %d", resp.StatusCode)
		case resp.StatusCode == http.StatusTooManyRequests:
			return fmt.Errorf("webhook endpoint rate limited: HTTP %d", resp.StatusCode)
		case resp.StatusCode >= 500:
			return fmt.Errorf("webhook endpoint server error: HTTP %d", resp.StatusCode)
		default:
			return fmt.Errorf("webhook request failed: HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

func classifyWebhookRequestError(err error) string {
	var urlErr *neturl.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return "webhook timeout (check endpoint latency or network)"
		}
		err = urlErr.Err
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "webhook dns lookup failed (verify webhook_url host)"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return "webhook network error (check connectivity to endpoint)"
	}

	return "webhook request transport error"
}
