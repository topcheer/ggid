package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AlertWebhook posts alert notifications to external URLs with HMAC signing.
type AlertWebhook struct {
	mu       sync.RWMutex
	endpoint string
	secret   string
	client   *http.Client
	sent     int
}

func NewAlertWebhook(endpoint, secret string) *AlertWebhook {
	return &AlertWebhook{
		endpoint: endpoint,
		secret:   secret,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// Post sends an alert payload with HMAC-SHA256 signature header.
func (w *AlertWebhook) Post(_ context.Context, payload []byte) error {
	if w.endpoint == "" {
		return fmt.Errorf("webhook endpoint not configured")
	}
	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, w.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GGID-Signature", "sha256="+sig)

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	w.mu.Lock()
	w.sent++
	w.mu.Unlock()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// Sign computes the HMAC-SHA256 signature for a payload.
func (w *AlertWebhook) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (w *AlertWebhook) GetSent() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.sent
}
