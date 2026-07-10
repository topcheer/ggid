package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/pkg/email"
)

func TestNewDispatcher(t *testing.T) {
	d := NewDispatcher(email.NewNoopSender(), &WebhookConfig{URL: "http://localhost/hook"})
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if d.email == nil {
		t.Error("expected email sender to be set")
	}
	if d.webhook == nil {
		t.Error("expected webhook config to be set")
	}
}

func TestNewDispatcher_NilChannels(t *testing.T) {
	d := NewDispatcher(nil, nil)
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestDispatcher_Dispatch_EmailOnly(t *testing.T) {
	var sent bool
	sender := &mockSender{onSend: func() { sent = true }}
	d := NewDispatcher(sender, nil)

	results := d.Dispatch(context.Background(), &Notification{
		Email:   "user@example.com",
		Subject: "Test",
		Message: "Hello",
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Channel != "email" || !results[0].Success {
		t.Errorf("expected email success, got %+v", results[0])
	}
	if !sent {
		t.Error("expected email to be sent")
	}
}

func TestDispatcher_Dispatch_WebhookOnly(t *testing.T) {
	var received bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		var n Notification
		_ = json.NewDecoder(r.Body).Decode(&n)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(nil, &WebhookConfig{URL: srv.URL, Headers: map[string]string{"X-Test": "val"}})

	results := d.Dispatch(context.Background(), &Notification{
		Type:    "test",
		Subject: "Test",
		Message: "Hello",
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Channel != "webhook" || !results[0].Success {
		t.Errorf("expected webhook success, got %+v", results[0])
	}
	if !received {
		t.Error("expected webhook to be received")
	}
}

func TestDispatcher_Dispatch_BothChannels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(email.NewNoopSender(), &WebhookConfig{URL: srv.URL})

	results := d.Dispatch(context.Background(), &Notification{
		Email:   "user@example.com",
		Subject: "Both",
		Message: "Test",
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("channel %s failed: %v", r.Channel, r.Error)
		}
	}
}

func TestDispatcher_Dispatch_NoChannels(t *testing.T) {
	d := NewDispatcher(nil, nil)
	results := d.Dispatch(context.Background(), &Notification{
		Subject: "Test",
	})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDispatcher_DispatchEmail(t *testing.T) {
	d := NewDispatcher(email.NewNoopSender(), nil)
	err := d.DispatchEmail(context.Background(), "user@example.com", "Test", "Body", "")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestDispatcher_DispatchEmail_NoSender(t *testing.T) {
	d := NewDispatcher(nil, nil)
	err := d.DispatchEmail(context.Background(), "user@example.com", "Test", "Body", "")
	if err == nil {
		t.Error("expected error when email sender not configured")
	}
}

func TestDispatcher_DispatchWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(nil, &WebhookConfig{URL: srv.URL})
	err := d.DispatchWebhook(context.Background(), &Notification{
		Type:    "test",
		Subject: "Test",
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestDispatcher_DispatchWebhook_NotConfigured(t *testing.T) {
	d := NewDispatcher(nil, nil)
	err := d.DispatchWebhook(context.Background(), &Notification{})
	if err == nil {
		t.Error("expected error when webhook not configured")
	}
}

func TestDispatcher_WebhookErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewDispatcher(nil, &WebhookConfig{URL: srv.URL})
	results := d.Dispatch(context.Background(), &Notification{
		Subject: "Test",
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected webhook failure")
	}
	if results[0].Error == nil {
		t.Error("expected non-nil error")
	}
}

// mockSender is a test email.Sender
type mockSender struct {
	onSend func()
}

func (m *mockSender) Send(_ context.Context, _ *email.Message) error {
	if m.onSend != nil {
		m.onSend()
	}
	return nil
}

func (m *mockSender) SendBatch(_ context.Context, msgs []*email.Message) error {
	for range msgs {
		if m.onSend != nil {
			m.onSend()
		}
	}
	return nil
}
