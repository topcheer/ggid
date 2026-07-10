// Package notification provides a unified notification dispatcher that can
// deliver messages via multiple channels (email, webhook, SMS).
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"bytes"

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
}

// Notification represents a notification to be dispatched.
type Notification struct {
	Type     string                 `json:"type"`     // e.g., "password_reset", "user_registered"
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
	return &Dispatcher{
		email:   emailSender,
		webhook: webhookCfg,
		client:  &http.Client{Timeout: timeout},
	}
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
