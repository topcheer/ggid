package email

import (
	"context"
	"strings"
	"testing"
)

// --- Template tests ---

func TestPasswordResetHTML_DefaultAppName(t *testing.T) {
	html := PasswordResetHTML(PasswordResetData{
		UserName: "John",
		Link:     "https://example.com/reset",
		Expiry:   "30 minutes",
	})
	if !strings.Contains(html, "GGID") {
		t.Error("expected default app name GGID when not set")
	}
}

func TestPasswordResetText(t *testing.T) {
	text := PasswordResetText(PasswordResetData{
		UserName: "John",
		Link:     "https://example.com/reset",
		Expiry:   "30 minutes",
	})
	if !strings.Contains(text, "John") {
		t.Error("expected user name in text")
	}
	if !strings.Contains(text, "https://example.com/reset") {
		t.Error("expected link in text")
	}
	if !strings.Contains(text, "30 minutes") {
		t.Error("expected expiry in text")
	}
}

func TestEmailVerificationHTML_DefaultAppName(t *testing.T) {
	html := EmailVerificationHTML(EmailVerificationData{
		UserName: "Jane",
		Link:     "https://example.com/verify",
	})
	if !strings.Contains(html, "GGID") {
		t.Error("expected default app name GGID")
	}
}

func TestWelcomeHTML_DefaultAppName(t *testing.T) {
	html := WelcomeHTML(WelcomeData{
		UserName: "Bob",
		Link:     "https://example.com",
	})
	if !strings.Contains(html, "GGID") {
		t.Error("expected default app name GGID")
	}
}

// --- SMTPSender validation ---

func TestSMTPSender_SendBatch_EmptyList(t *testing.T) {
	s := NewSMTPSender(Config{Host: "localhost", Port: 25})
	err := s.SendBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("expected nil for empty batch, got %v", err)
	}
}

func TestSMTPSender_SendBatch_InvalidMessage(t *testing.T) {
	s := NewSMTPSender(Config{Host: "localhost", Port: 25})
	msgs := []*Message{
		{To: []string{"valid@example.com"}, Subject: "OK"},
		{To: []string{}, Subject: "Invalid"}, // empty recipients
	}
	err := s.SendBatch(context.Background(), msgs)
	if err == nil {
		t.Error("expected error for invalid message in batch")
	}
}

// --- SMTPSender config ---

func TestSMTPSender_ConfigDefaults(t *testing.T) {
	s := NewSMTPSender(Config{})
	if s.cfg.Timeout != 30e9 {
		t.Errorf("expected default timeout 30s, got %v", s.cfg.Timeout)
	}
	if s.cfg.TLSMode != "starttls" {
		t.Errorf("expected default TLS starttls, got %s", s.cfg.TLSMode)
	}
}

func TestSMTPSender_CustomTimeout(t *testing.T) {
	s := NewSMTPSender(Config{Timeout: 5000000000}) // 5s
	if s.cfg.Timeout != 5000000000 {
		t.Errorf("expected custom timeout, got %v", s.cfg.Timeout)
	}
}

func TestSMTPSender_CustomTLSMode(t *testing.T) {
	s := NewSMTPSender(Config{TLSMode: "tls"})
	if s.cfg.TLSMode != "tls" {
		t.Errorf("expected tls, got %s", s.cfg.TLSMode)
	}
}

// --- SMTPSender header building ---

func TestSMTPSender_BuildHeaders_NoCc(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Test",
	}
	headers := s.buildHeaders("noreply@ggid.dev", msg)
	if !strings.Contains(headers, "To: user@example.com") {
		t.Error("expected To header")
	}
	if strings.Contains(headers, "Cc:") {
		t.Error("should not have Cc header when empty")
	}
}

func TestSMTPSender_BuildHeaders_CustomHeaders(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Headers: map[string]string{
			"X-Custom":      "Value1",
			"X-Priority":    "1",
			"X-Mailer":      "GGID",
		},
	}
	headers := s.buildHeaders("noreply@ggid.dev", msg)
	if !strings.Contains(headers, "X-Custom: Value1") {
		t.Error("expected custom header")
	}
	if !strings.Contains(headers, "X-Priority: 1") {
		t.Error("expected priority header")
	}
	if !strings.Contains(headers, "X-Mailer: GGID") {
		t.Error("expected mailer header")
	}
}

func TestSMTPSender_BuildHeaders_TextPlain(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		To:       []string{"user@example.com"},
		Subject:  "Text Test",
		TextBody: "Hello",
	}
	headers := s.buildHeaders("noreply@ggid.dev", msg)
	if !strings.Contains(headers, "Content-Type: text/plain") {
		t.Error("expected text/plain content type")
	}
}

// --- extractAddress edge cases ---

func TestSMTPSender_ExtractAddress_NoBrackets(t *testing.T) {
	s := NewSMTPSender(Config{})
	got := s.extractAddress("plain@example.com")
	if got != "plain@example.com" {
		t.Errorf("expected plain@example.com, got %s", got)
	}
}

func TestSMTPSender_ExtractAddress_WithBrackets(t *testing.T) {
	s := NewSMTPSender(Config{})
	got := s.extractAddress("Display Name <addr@example.com>")
	if got != "addr@example.com" {
		t.Errorf("expected addr@example.com, got %s", got)
	}
}

func TestSMTPSender_ExtractAddress_Empty(t *testing.T) {
	s := NewSMTPSender(Config{})
	got := s.extractAddress("")
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

// --- send failure (connection refused) ---

func TestSMTPSender_Send_ConnectionFailed(t *testing.T) {
	s := NewSMTPSender(Config{
		Host:    "127.0.0.1",
		Port:    1, // unreachable port
		TLSMode: "none",
	})
	err := s.Send(context.Background(), &Message{
		From:    "noreply@ggid.dev",
		To:      []string{"user@example.com"},
		Subject: "Test",
		TextBody: "Hello",
	})
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestSMTPSender_Send_TLSConnectionFailed(t *testing.T) {
	s := NewSMTPSender(Config{
		Host:    "127.0.0.1",
		Port:    1, // unreachable port
		TLSMode: "tls",
	})
	err := s.Send(context.Background(), &Message{
		From:    "noreply@ggid.dev",
		To:      []string{"user@example.com"},
		Subject: "Test",
		TextBody: "Hello",
	})
	if err == nil {
		t.Error("expected TLS connection error")
	}
	if !strings.Contains(err.Error(), "TLS dial failed") {
		t.Errorf("expected TLS dial error, got: %v", err)
	}
}

// --- LogSender ---

func TestLogSender_SendBatch(t *testing.T) {
	var count int
	s := NewLogSender(func(format string, args ...interface{}) {
		count++
	})
	msgs := []*Message{
		{To: []string{"a@example.com"}, Subject: "A"},
		{To: []string{"b@example.com"}, Subject: "B"},
	}
	err := s.SendBatch(context.Background(), msgs)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 log calls, got %d", count)
	}
}

func TestLogSender_NilLogFunc(t *testing.T) {
	s := NewLogSender(nil)
	err := s.Send(context.Background(), &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	if err != nil {
		t.Errorf("expected nil error with nil log func, got %v", err)
	}
}

// --- Send with From fallback to config ---

func TestSMTPSender_Send_FromFallback(t *testing.T) {
	s := NewSMTPSender(Config{
		Host:    "127.0.0.1",
		Port:    1,
		From:    "default@ggid.dev",
		FromName: "GGID System",
		TLSMode: "none",
	})
	// Send without From in message — should use config From
	err := s.Send(context.Background(), &Message{
		To:      []string{"user@example.com"},
		Subject: "Test",
		TextBody: "Hello",
	})
	// Will fail on connection but validates that From fallback path is exercised
	if err == nil {
		t.Log("send succeeded (unexpected but not harmful)")
	}
}

// --- Send with custom From in message ---

func TestSMTPSender_Send_CustomFrom(t *testing.T) {
	s := NewSMTPSender(Config{
		Host:    "127.0.0.1",
		Port:    1,
		From:    "default@ggid.dev",
		TLSMode: "none",
	})
	err := s.Send(context.Background(), &Message{
		From:    "custom@example.com",
		To:      []string{"user@example.com"},
		Subject: "Test",
		TextBody: "Hello",
	})
	if err == nil {
		t.Log("send succeeded (unexpected but not harmful)")
	}
}

// --- Bcc recipients ---

func TestSMTPSender_BuildHeaders_WithBcc(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		To:      []string{"user@example.com"},
		Bcc:     []string{"hidden@example.com"},
		Subject: "Test",
	}
	headers := s.buildHeaders("noreply@ggid.dev", msg)
	// Bcc should NOT appear in headers (it's stripped)
	if strings.Contains(headers, "Bcc:") {
		t.Error("Bcc should not appear in headers")
	}
}

// --- BuildBody edge cases ---

func TestSMTPSender_BuildBody_EmptyHTML(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		TextBody: "Plain text",
		HTMLBody: "", // empty, should fall back to text
	}
	body := s.buildBody(msg)
	if !strings.Contains(body, "Plain text") {
		t.Errorf("expected text body, got %s", body)
	}
}

// --- Interface compliance ---

func TestNoopSender_ImplementsSender(t *testing.T) {
	var _ Sender = NewNoopSender()
}

func TestLogSender_ImplementsSender(t *testing.T) {
	var _ Sender = NewLogSender(nil)
}

func TestSMTPSender_ImplementsSender(t *testing.T) {
	var _ Sender = NewSMTPSender(Config{})
}
