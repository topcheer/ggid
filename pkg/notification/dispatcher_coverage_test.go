package notification

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/email"
)

// mockSender implements email.Sender for testing.
type mockSender2 struct {
	err     error
	lastMsg *email.Message
}

func (m *mockSender2) Send(ctx context.Context, msg *email.Message) error {
	m.lastMsg = msg
	return m.err
}

func TestNewDispatcher_WithCustomTimeout(t *testing.T) {
	cfg := &WebhookConfig{URL: "http://example.com", Timeout: 30 * time.Second}
	d := NewDispatcher(nil, cfg)
	if d.webhook != cfg {
		t.Error("expected webhook config to be set")
	}
	if d.client.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", d.client.Timeout)
	}
}

func TestNewDispatcher_WithNilWebhook(t *testing.T) {
	d := NewDispatcher(&mockSender2{}, nil)
	if d.webhook != nil {
		t.Error("expected nil webhook")
	}
	if d.client.Timeout != 10*time.Second {
		t.Errorf("expected default 10s timeout, got %v", d.client.Timeout)
	}
}

func TestNewDispatcher_WithZeroTimeout(t *testing.T) {
	cfg := &WebhookConfig{URL: "http://example.com", Timeout: 0}
	d := NewDispatcher(nil, cfg)
	if d.client.Timeout != 10*time.Second {
		t.Errorf("expected default 10s for zero timeout, got %v", d.client.Timeout)
	}
}

func TestDispatch_NoChannels(t *testing.T) {
	d := NewDispatcher(nil, nil)
	results := d.Dispatch(context.Background(), &Notification{
		Type:    "test",
		Subject: "Test",
	})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDispatch_EmailOnly(t *testing.T) {
	sender := &mockSender2{}
	d := NewDispatcher(sender, nil)
	results := d.Dispatch(context.Background(), &Notification{
		Type:    "test",
		Email:   "user@example.com",
		Subject: "Subject",
		Message: "Body",
	})
	if len(results) != 1 || results[0].Channel != "email" || !results[0].Success {
		t.Errorf("expected 1 successful email result, got %v", results)
	}
	if sender.lastMsg == nil || sender.lastMsg.To[0] != "user@example.com" {
		t.Error("email not sent correctly")
	}
}

func TestDispatch_EmailError(t *testing.T) {
	sender := &mockSender2{err: fmt.Errorf("smtp error")}
	d := NewDispatcher(sender, nil)
	results := d.Dispatch(context.Background(), &Notification{
		Email:   "user@example.com",
		Subject: "S",
		Message: "M",
	})
	if len(results) != 1 || results[0].Success {
		t.Errorf("expected failed result, got %v", results)
	}
	if results[0].Error == nil {
		t.Error("expected error in result")
	}
}

func TestDispatch_BothChannels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := &mockSender2{}
	d := NewDispatcher(sender, &WebhookConfig{URL: srv.URL})
	results := d.Dispatch(context.Background(), &Notification{
		Email:   "user@example.com",
		Subject: "S",
		Message: "M",
	})
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSendWebhook_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(nil, &WebhookConfig{
		URL:     srv.URL,
		Headers: map[string]string{"X-Signature": "abc123", "X-Tenant": "t1"},
	})
	err := d.DispatchWebhook(context.Background(), &Notification{
		Type: "test", Subject: "S", Message: "M",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedHeaders.Get("X-Signature") != "abc123" {
		t.Error("custom header not sent")
	}
	if receivedHeaders.Get("X-Tenant") != "t1" {
		t.Error("custom header not sent")
	}
}

func TestSendWebhook_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewDispatcher(nil, &WebhookConfig{URL: srv.URL})
	err := d.DispatchWebhook(context.Background(), &Notification{
		Type: "test", Subject: "S", Message: "M",
	})
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	d := NewDispatcher(nil, &WebhookConfig{URL: "http://127.0.0.1:1"})
	err := d.DispatchWebhook(context.Background(), &Notification{
		Type: "test", Subject: "S", Message: "M",
	})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestDispatchWebhook_NotConfigured(t *testing.T) {
	d := NewDispatcher(nil, nil)
	err := d.DispatchWebhook(context.Background(), &Notification{})
	if err == nil {
		t.Error("expected error when webhook not configured")
	}
}

func TestDispatchEmail_NotConfigured(t *testing.T) {
	d := NewDispatcher(nil, nil)
	err := d.DispatchEmail(context.Background(), "a@b.c", "s", "b", "")
	if err == nil {
		t.Error("expected error when email not configured")
	}
}

func TestDispatchEmail_Success(t *testing.T) {
	sender := &mockSender2{}
	d := NewDispatcher(sender, nil)
	err := d.DispatchEmail(context.Background(), "a@b.c", "subj", "body", "<p>body</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sender.lastMsg == nil || sender.lastMsg.Subject != "subj" {
		t.Error("email not sent correctly")
	}
	if sender.lastMsg.HTMLBody != "<p>body</p>" {
		t.Error("HTML body not set")
	}
}
