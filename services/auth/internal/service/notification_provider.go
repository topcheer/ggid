package service

import (
	"context"
	"fmt"
	"sync"
)

// NotificationChannel represents a delivery method.
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
	ChannelPush  NotificationChannel = "push"
)

// Notification represents a message to send.
type Notification struct {
	Channel  NotificationChannel
	To       string
	Subject  string
	Body     string
	Template string
	Data     map[string]any
}

// NotificationProvider sends notifications across multiple channels.
type NotificationProvider struct {
	mu      sync.RWMutex
	emailer EmailSender
	sms     SMSSender
	push    PushSender
	sent    []Notification // log for testing
}

// EmailSender sends email notifications.
type EmailSender interface {
	Send(to []string, subject string, body []byte) error
}

// SMSSender sends SMS notifications.
type SMSSender interface {
	SendSMS(to, message string) error
}

// PushSender sends push notifications.
type PushSender interface {
	SendPush(deviceToken, title, body string) error
}

// NewNotificationProvider creates a multi-channel notification provider.
func NewNotificationProvider(emailer EmailSender, sms SMSSender, push PushSender) *NotificationProvider {
	return &NotificationProvider{emailer: emailer, sms: sms, push: push}
}

// Send dispatches a notification via the appropriate channel.
func (p *NotificationProvider) Send(ctx context.Context, n *Notification) error {
	if n == nil {
		return fmt.Errorf("nil notification")
	}
	switch n.Channel {
	case ChannelEmail:
		if p.emailer == nil {
			return fmt.Errorf("email channel not configured")
		}
		err := p.emailer.Send([]string{n.To}, n.Subject, []byte(n.Body))
		p.log(*n)
		return err
	case ChannelSMS:
		if p.sms == nil {
			return fmt.Errorf("SMS channel not configured")
		}
		err := p.sms.SendSMS(n.To, n.Body)
		p.log(*n)
		return err
	case ChannelPush:
		if p.push == nil {
			return fmt.Errorf("push channel not configured")
		}
		err := p.push.SendPush(n.To, n.Subject, n.Body)
		p.log(*n)
		return err
	default:
		return fmt.Errorf("unknown channel: %s", n.Channel)
	}
}

// SendFromTemplate renders a template and sends via the specified channel.
func (p *NotificationProvider) SendFromTemplate(ctx context.Context, channel NotificationChannel, to, template string, data map[string]any) error {
	subject := fmt.Sprintf("[GGID] %s", template)
	body := renderTemplate(template, data)
	return p.Send(ctx, &Notification{
		Channel: channel, To: to, Subject: subject, Body: body, Template: template, Data: data,
	})
}

func renderTemplate(name string, data map[string]any) string {
	if v, ok := data["body"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("Notification: %s", name)
}

func (p *NotificationProvider) log(n Notification) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, n)
}

// GetSent returns all sent notifications (for testing).
func (p *NotificationProvider) GetSent() []Notification {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]Notification, len(p.sent))
	copy(out, p.sent)
	return out
}
