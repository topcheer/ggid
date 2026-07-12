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

type AlertWebhook struct {
	URL        string   `json:"url"`
	Secret     string   `json:"secret"`
	EventTypes []string `json:"event_types"`
	Enabled    bool     `json:"enabled"`
	MaxRetries int      `json:"max_retries"`
}

type WebhookStats struct {
	Delivered    int `json:"delivered"`
	Failed       int `json:"failed"`
	Retried      int `json:"retried"`
	DeadLettered int `json:"dead_lettered"`
}

type DeadLetterEntry struct {
	Alert      []byte    `json:"alert"`
	Error      string    `json:"error"`
	Attempts   int       `json:"attempts"`
	LastTried  time.Time `json:"last_tried"`
}

type AlertWebhookService struct {
	mu          sync.RWMutex
	webhooks    map[string]*AlertWebhook
	stats       map[string]*WebhookStats
	deadLetters map[string][]DeadLetterEntry
	dedupCache  map[string]time.Time
	client      *http.Client
}

func (s *AlertWebhookService) RegisterWebhook(name string, wh AlertWebhook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhooks[name] = &wh
	s.stats[name] = &WebhookStats{}
}

func (s *AlertWebhookService) DeliverAlert(name string, alert []byte) error {
	s.mu.Lock()
	wh, ok := s.webhooks[name]
	if !ok || !wh.Enabled {
		s.mu.Unlock()
		return fmt.Errorf("webhook not found or disabled")
	}
	// Dedup check
	dedupKey := name + ":" + string(alert)
	if lastSent, exists := s.dedupCache[dedupKey]; exists && time.Since(lastSent) < 5*time.Minute {
		s.mu.Unlock()
		return nil // skip duplicate
	}
	s.dedupCache[dedupKey] = time.Now()
	stats := s.stats[name]
	s.mu.Unlock()

	maxRetries := wh.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
			s.mu.Lock()
			stats.Retried++
			s.mu.Unlock()
		}
		err := s.doDelivery(wh, alert)
		if err == nil {
			s.mu.Lock()
			stats.Delivered++
			s.mu.Unlock()
			return nil
		}
		lastErr = err
	}

	s.mu.Lock()
	stats.Failed++
	stats.DeadLettered++
	s.deadLetters[name] = append(s.deadLetters[name], DeadLetterEntry{
		Alert:     alert,
		Error:     lastErr.Error(),
		Attempts:  maxRetries + 1,
		LastTried: time.Now(),
	})
	s.mu.Unlock()
	return lastErr
}

func (s *AlertWebhookService) doDelivery(wh *AlertWebhook, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(payload)
		req.Header.Set("X-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (s *AlertWebhookService) HealthCheck(name string) bool {
	s.mu.RLock()
	wh, ok := s.webhooks[name]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	resp, err := s.client.Get(wh.URL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 400
}

func (s *AlertWebhookService) GetStats(name string) *WebhookStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats[name]
}

func (s *AlertWebhookService) GetDeadLetters(name string) []DeadLetterEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deadLetters[name]
}

// --- AlertWebhookSender for existing test compatibility ---

type AlertWebhookSender struct {
	url    string
	secret string
	sent   int
	mu     sync.Mutex
}

func NewAlertWebhook(url, secret string) *AlertWebhookSender {
	return &AlertWebhookSender{url: url, secret: secret}
}

func (w *AlertWebhookSender) Post(ctx context.Context, payload []byte) error {
	if w.url == "" {
		return fmt.Errorf("empty endpoint")
	}
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.secret != "" {
		mac := hmac.New(sha256.New, []byte(w.secret))
		mac.Write(payload)
		req.Header.Set("X-GGID-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	w.mu.Lock()
	w.sent++
	w.mu.Unlock()
	return nil
}

func (w *AlertWebhookSender) GetSent() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sent
}

func (w *AlertWebhookSender) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}