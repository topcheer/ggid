package service

import (
	"context"
	"testing"
)

// --- Mock senders for testing ---

type mockEmailSender struct {
	lastTo   []string
	lastSubj string
	count    int
}

func (m *mockEmailSender) Send(to []string, subject string, body []byte) error {
	m.count++
	m.lastTo = to
	m.lastSubj = subject
	return nil
}

type mockSMSSender struct {
	lastTo  string
	lastMsg string
	count   int
}

func (m *mockSMSSender) SendSMS(to, message string) error {
	m.count++
	m.lastTo = to
	m.lastMsg = message
	return nil
}

type mockPushSender struct {
	lastToken string
	lastTitle string
	count     int
}

func (m *mockPushSender) SendPush(token, title, body string) error {
	m.count++
	m.lastToken = token
	m.lastTitle = title
	return nil
}

func TestNotification_Email(t *testing.T) {
	emailer := &mockEmailSender{}
	p := NewNotificationProvider(emailer, nil, nil)

	err := p.Send(context.Background(), &Notification{
		Channel: ChannelEmail, To: "user@test.com", Subject: "Verify", Body: "Click here",
	})
	if err != nil {
		t.Fatalf("send email: %v", err)
	}
	if emailer.count != 1 {
		t.Error("should have sent 1 email")
	}
	if emailer.lastTo[0] != "user@test.com" {
		t.Error("wrong recipient")
	}
}

func TestNotification_SMS(t *testing.T) {
	sms := &mockSMSSender{}
	p := NewNotificationProvider(nil, sms, nil)

	err := p.Send(context.Background(), &Notification{
		Channel: ChannelSMS, To: "+1234567890", Body: "Code: 1234",
	})
	if err != nil {
		t.Fatalf("send SMS: %v", err)
	}
	if sms.count != 1 {
		t.Error("should have sent 1 SMS")
	}
}

func TestNotification_Push(t *testing.T) {
	push := &mockPushSender{}
	p := NewNotificationProvider(nil, nil, push)

	err := p.Send(context.Background(), &Notification{
		Channel: ChannelPush, To: "device-token-abc", Subject: "Alert", Body: "New login",
	})
	if err != nil {
		t.Fatalf("send push: %v", err)
	}
	if push.count != 1 {
		t.Error("should have sent 1 push")
	}
}

func TestNotification_UnconfiguredChannel(t *testing.T) {
	p := NewNotificationProvider(nil, nil, nil)
	err := p.Send(context.Background(), &Notification{Channel: ChannelEmail})
	if err == nil {
		t.Error("should error when channel not configured")
	}
}

func TestNotification_UnknownChannel(t *testing.T) {
	p := NewNotificationProvider(nil, nil, nil)
	err := p.Send(context.Background(), &Notification{Channel: "fax"})
	if err == nil {
		t.Error("should error for unknown channel")
	}
}

func TestNotification_FromTemplate(t *testing.T) {
	emailer := &mockEmailSender{}
	p := NewNotificationProvider(emailer, nil, nil)

	err := p.SendFromTemplate(context.Background(), ChannelEmail, "user@test.com", "welcome", map[string]any{"body": "Welcome!"})
	if err != nil {
		t.Fatalf("template send: %v", err)
	}
	if emailer.count != 1 {
		t.Error("should send 1 email")
	}
}

func TestNotification_GetSent(t *testing.T) {
	emailer := &mockEmailSender{}
	p := NewNotificationProvider(emailer, nil, nil)

	p.Send(context.Background(), &Notification{Channel: ChannelEmail, To: "a@test.com"})
	p.Send(context.Background(), &Notification{Channel: ChannelEmail, To: "b@test.com"})

	sent := p.GetSent()
	if len(sent) != 2 {
		t.Errorf("expected 2 sent, got %d", len(sent))
	}
}

func TestNotification_Nil(t *testing.T) {
	p := NewNotificationProvider(nil, nil, nil)
	err := p.Send(context.Background(), nil)
	if err == nil {
		t.Error("should error on nil notification")
	}
}
