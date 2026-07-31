// Package notification provides a unified notification dispatcher that can
// deliver messages via multiple channels (email, webhook, SMS).
package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/email"
)

// Dispatcher routes notifications to one or more channels.
type Dispatcher struct {
	email   email.Sender
	webhook *WebhookConfig
	client  *http.Client
}

// WebhookConfig holds webhook delivery settings.
type WebhookConfig struct {
	URL     string
	Headers map[string]string
	Timeout time.Duration
	Secret  string // HMAC-SHA256 signing secret for payload verification
}

// NewWebhookConfigFromEnv creates a WebhookConfig from environment variables.
// Reads NOTIFICATION_WEBHOOK_URL, NOTIFICATION_WEBHOOK_SECRET, and
// NOTIFICATION_WEBHOOK_TIMEOUT (seconds). Returns nil if no URL is set.
// SECURITY: logs a warning if URL is set but Secret is empty in production.
func NewWebhookConfigFromEnv() *WebhookConfig {
	url := os.Getenv("NOTIFICATION_WEBHOOK_URL")
	if url == "" {
		return nil
	}
	cfg := &WebhookConfig{
		URL:    url,
		Secret: os.Getenv("NOTIFICATION_WEBHOOK_SECRET"),
	}
	if secs := os.Getenv("NOTIFICATION_WEBHOOK_TIMEOUT"); secs != "" {
		if d, err := time.ParseDuration(secs + "s"); err == nil {
			cfg.Timeout = d
		}
	}
	env := os.Getenv("GGID_ENV")
	if cfg.Secret == "" && env != "test" && env != "dev" {
		slog.Warn("NOTIFICATION_WEBHOOK_SECRET not set — webhook payloads will be unsigned (unauthenticated)")
	}
	return cfg
}

// Notification represents a notification to be dispatched.
type Notification struct {
	Type     string                 `json:"type"` // e.g., "password_reset", "user_registered"
	TenantID string                 `json:"tenant_id"`
	UserID   string                 `json:"user_id,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Subject  string                 `json:"subject"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// Result holds the outcome of a notification dispatch.
type Result struct {
	Channel string // "email", "webhook"
	Success bool
	Error   error
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(emailSender email.Sender, webhookCfg *WebhookConfig) *Dispatcher {
	timeout := 10 * time.Second
	if webhookCfg != nil && webhookCfg.Timeout > 0 {
		timeout = webhookCfg.Timeout
	}
	d := &Dispatcher{
		email:   emailSender,
		webhook: webhookCfg,
		client: &http.Client{
			Timeout: timeout,
			// SECURITY: Prevent SSRF via redirect to internal IPs.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				DialContext: ssrfSafeDialContext,
			},
		},
	}
	// SECURITY: enforce HTTPS for webhook URLs to prevent PII leakage over plaintext.
	// Allow http:// only for localhost (testing/development).
	if webhookCfg != nil && webhookCfg.URL != "" {
		if !strings.HasPrefix(webhookCfg.URL, "https://") {
			isLocal := strings.Contains(webhookCfg.URL, "127.0.0.1") || strings.Contains(webhookCfg.URL, "localhost")
			if !isLocal {
				slog.Warn("webhook URL is not HTTPS; notifications will not be sent", "url", webhookCfg.URL)
				d.webhook = nil // disable insecure webhook
			}
		}
	}
	return d
}

// Dispatch sends a notification via all configured channels concurrently.
func (d *Dispatcher) Dispatch(ctx context.Context, n *Notification) []Result {
	var wg sync.WaitGroup
	results := make([]Result, 0)
	var mu sync.Mutex

	// Email channel
	if d.email != nil && n.Email != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := d.email.Send(ctx, &email.Message{
				To:       []string{n.Email},
				Subject:  n.Subject,
				TextBody: n.Message,
			})
			mu.Lock()
			results = append(results, Result{Channel: "email", Success: err == nil, Error: err})
			mu.Unlock()
		}()
	}

	// Webhook channel
	if d.webhook != nil && d.webhook.URL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := d.sendWebhook(ctx, n)
			mu.Lock()
			results = append(results, Result{Channel: "webhook", Success: err == nil, Error: err})
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

// DispatchEmail sends only via email channel.
func (d *Dispatcher) DispatchEmail(ctx context.Context, to, subject, textBody, htmlBody string) error {
	if d.email == nil {
		return fmt.Errorf("notification: email sender not configured")
	}
	return d.email.Send(ctx, &email.Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	})
}

// DispatchWebhook sends only via webhook channel.
func (d *Dispatcher) DispatchWebhook(ctx context.Context, n *Notification) error {
	if d.webhook == nil || d.webhook.URL == "" {
		return fmt.Errorf("notification: webhook not configured")
	}
	return d.sendWebhook(ctx, n)
}

func (d *Dispatcher) sendWebhook(ctx context.Context, n *Notification) error {
	payload, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("notification: marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("notification: create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// HMAC-SHA256 signature for payload authenticity verification.
	if d.webhook.Secret != "" {
		mac := hmac.New(sha256.New, []byte(d.webhook.Secret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-GGID-Signature", "sha256="+sig)
	}

	for k, v := range d.webhook.Headers {
		req.Header.Set(k, v)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("notification: webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification: webhook returned %d", resp.StatusCode)
	}
	return nil
}

// ssrfSafeDialContext blocks connections to private/link-local/loopback IPs
// to prevent SSRF attacks via webhook URLs.
func ssrfSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Allow localhost in test mode for unit tests using httptest.Server.
	if os.Getenv("GGID_ENV") == "test" {
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		return dialer.DialContext(ctx, network, addr)
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS resolution failed: %w", err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no DNS records for %s", host)
		}
		ip = ips[0].IP
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return nil, fmt.Errorf("connection to %s blocked by SSRF protection", ip)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
}
