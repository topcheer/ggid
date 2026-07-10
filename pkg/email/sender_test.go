package email

import (
	"context"
	"testing"
)

func TestNoopSender_Send(t *testing.T) {
	s := NewNoopSender()
	err := s.Send(context.Background(), &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoopSender_SendBatch(t *testing.T) {
	s := NewNoopSender()
	msgs := []*Message{
		{To: []string{"a@example.com"}, Subject: "A"},
		{To: []string{"b@example.com"}, Subject: "B"},
	}
	err := s.SendBatch(context.Background(), msgs)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestLogSender_Send(t *testing.T) {
	var logged bool
	s := NewLogSender(func(format string, args ...interface{}) {
		logged = true
	})
	err := s.Send(context.Background(), &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !logged {
		t.Error("expected log function to be called")
	}
}

func TestSMTPSender_Send_EmptyRecipients(t *testing.T) {
	s := NewSMTPSender(Config{Host: "localhost", Port: 25})
	err := s.Send(context.Background(), &Message{
		To:      []string{},
		Subject: "Test",
	})
	if err == nil {
		t.Error("expected error for empty recipients")
	}
}

func TestSMTPSender_Send_EmptySubject(t *testing.T) {
	s := NewSMTPSender(Config{Host: "localhost", Port: 25})
	err := s.Send(context.Background(), &Message{
		To:      []string{"test@example.com"},
		Subject: "",
	})
	if err == nil {
		t.Error("expected error for empty subject")
	}
}

func TestSMTPSender_ExtractAddress(t *testing.T) {
	s := NewSMTPSender(Config{})
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user@example.com"},
		{"John <john@example.com>", "john@example.com"},
		{"Jane Doe <jane@example.com>", "jane@example.com"},
	}
	for _, tt := range tests {
		got := s.extractAddress(tt.input)
		if got != tt.expected {
			t.Errorf("extractAddress(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSMTPSender_BuildHeaders(t *testing.T) {
	s := NewSMTPSender(Config{From: "noreply@ggid.dev", FromName: "GGID"})
	msg := &Message{
		To:      []string{"user@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "Test Subject",
		ReplyTo: "support@ggid.dev",
	}
	headers := s.buildHeaders("GGID <noreply@ggid.dev>", msg)
	if !contains(headers, "To: user@example.com") {
		t.Error("expected To header")
	}
	if !contains(headers, "Cc: cc@example.com") {
		t.Error("expected Cc header")
	}
	if !contains(headers, "Subject: Test Subject") {
		t.Error("expected Subject header")
	}
	if !contains(headers, "Reply-To: support@ggid.dev") {
		t.Error("expected Reply-To header")
	}
	if !contains(headers, "MIME-Version: 1.0") {
		t.Error("expected MIME-Version header")
	}
}

func TestSMTPSender_BuildHeaders_HTML(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{
		To:       []string{"user@example.com"},
		Subject:  "HTML Test",
		HTMLBody: "<p>Hello</p>",
	}
	headers := s.buildHeaders("noreply@ggid.dev", msg)
	if !contains(headers, "Content-Type: text/html") {
		t.Error("expected text/html content type")
	}
}

func TestSMTPSender_BuildBody_HTML(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{HTMLBody: "<p>Test</p>"}
	body := s.buildBody(msg)
	if body != "<p>Test</p>\r\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestSMTPSender_BuildBody_Text(t *testing.T) {
	s := NewSMTPSender(Config{})
	msg := &Message{TextBody: "Hello"}
	body := s.buildBody(msg)
	if body != "Hello\r\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestPasswordResetHTML(t *testing.T) {
	html := PasswordResetHTML(PasswordResetData{
		UserName: "John",
		Link:     "https://example.com/reset?token=abc",
		Expiry:   "30 minutes",
	})
	if !contains(html, "John") {
		t.Error("expected user name in HTML")
	}
	if !contains(html, "https://example.com/reset?token=abc") {
		t.Error("expected link in HTML")
	}
	if !contains(html, "30 minutes") {
		t.Error("expected expiry in HTML")
	}
}

func TestEmailVerificationHTML(t *testing.T) {
	html := EmailVerificationHTML(EmailVerificationData{
		UserName: "Jane",
		Link:     "https://example.com/verify?token=xyz",
	})
	if !contains(html, "Jane") {
		t.Error("expected user name in HTML")
	}
	if !contains(html, "Verify Your Email") {
		t.Error("expected title in HTML")
	}
}

func TestWelcomeHTML(t *testing.T) {
	html := WelcomeHTML(WelcomeData{
		UserName: "Bob",
		AppName:  "MyApp",
		Link:     "https://example.com",
	})
	if !contains(html, "MyApp") {
		t.Error("expected app name in HTML")
	}
	if !contains(html, "Bob") {
		t.Error("expected user name in HTML")
	}
}

func TestMFACodeHTML(t *testing.T) {
	html := MFACodeHTML(MFACodeData{
		UserName: "Alice",
		Code:     "123456",
	})
	if !contains(html, "123456") {
		t.Error("expected code in HTML")
	}
	if !contains(html, "Alice") {
		t.Error("expected user name in HTML")
	}
}

func TestSMTPSender_DefaultConfig(t *testing.T) {
	s := NewSMTPSender(Config{})
	if s.cfg.Timeout != 30*1e9 { // 30 seconds in nanoseconds
		t.Errorf("expected default timeout 30s, got %v", s.cfg.Timeout)
	}
	if s.cfg.TLSMode != "starttls" {
		t.Errorf("expected default TLS mode starttls, got %s", s.cfg.TLSMode)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
