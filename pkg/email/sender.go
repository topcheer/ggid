// Package email provides a pluggable email sending interface and SMTP implementation
// for the GGID IAM platform.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Sender is the interface for sending emails.
type Sender interface {
	// Send sends an email to the given recipient.
	Send(ctx context.Context, msg *Message) error
	// SendBatch sends multiple emails in a single connection.
	SendBatch(ctx context.Context, msgs []*Message) error
}

// Message represents an email message.
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextBody    string
	HTMLBody    string
	ReplyTo     string
	Headers     map[string]string
	Attachments []Attachment
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}

// Config holds SMTP server configuration.
type Config struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	From     string `yaml:"from" json:"from"`
	FromName string `yaml:"from_name" json:"fromName"`
	// TLSMode: "none", "starttls", "tls"
	TLSMode string `yaml:"tls_mode" json:"tlsMode"`
	// Timeout for SMTP operations.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// SMTPSender implements Sender using Go's net/smtp package.
type SMTPSender struct {
	cfg Config
}

// NewSMTPSender creates a new SMTP sender.
func NewSMTPSender(cfg Config) *SMTPSender {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.TLSMode == "" {
		cfg.TLSMode = "starttls"
	}
	return &SMTPSender{cfg: cfg}
}

// Send sends a single email.
func (s *SMTPSender) Send(ctx context.Context, msg *Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("email: recipient list is empty")
	}
	if msg.Subject == "" {
		return fmt.Errorf("email: subject is empty")
	}

	return s.send(ctx, msg)
}

// SendBatch sends multiple emails.
func (s *SMTPSender) SendBatch(ctx context.Context, msgs []*Message) error {
	for _, msg := range msgs {
		if err := s.Send(ctx, msg); err != nil {
			return fmt.Errorf("email: failed to send to %v: %w", msg.To, err)
		}
	}
	return nil
}

func (s *SMTPSender) send(ctx context.Context, msg *Message) error {
	from := msg.From
	if from == "" {
		from = s.cfg.From
	}
	if s.cfg.FromName != "" && from == s.cfg.From {
		from = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.From)
	}

	host := s.cfg.Host
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	headers := s.buildHeaders(from, msg)
	body := s.buildBody(msg)
	rawMsg := headers + "\r\n" + body

	recipients := append([]string{}, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	switch s.cfg.TLSMode {
	case "tls":
		return s.sendWithTLS(addr, host, auth, from, recipients, []byte(rawMsg))
	case "none":
		return smtp.SendMail(addr, auth, s.extractAddress(from), recipients, []byte(rawMsg))
	default: // starttls
		return smtp.SendMail(addr, auth, s.extractAddress(from), recipients, []byte(rawMsg))
	}
}

func (s *SMTPSender) sendWithTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}

	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("email: TLS dial failed: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("email: SMTP client failed: %w", err)
	}
	defer c.Close()

	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("email: SMTP auth failed: %w", err)
	}
	if err = c.Mail(s.extractAddress(from)); err != nil {
		return fmt.Errorf("email: MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err = c.Rcpt(recipient); err != nil {
			return fmt.Errorf("email: RCPT TO failed for %s: %w", recipient, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("email: DATA failed: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("email: write body failed: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("email: close data failed: %w", err)
	}

	return c.Quit()
}

func (s *SMTPSender) buildHeaders(from string, msg *Message) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.Cc) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	if msg.ReplyTo != "" {
		b.WriteString(fmt.Sprintf("Reply-To: %s\r\n", msg.ReplyTo))
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	if msg.HTMLBody != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	for k, v := range msg.Headers {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	b.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	return b.String()
}

func (s *SMTPSender) buildBody(msg *Message) string {
	if msg.HTMLBody != "" {
		return msg.HTMLBody + "\r\n"
	}
	return msg.TextBody + "\r\n"
}

func (s *SMTPSender) extractAddress(from string) string {
	// Extract email from "Name <email>" format
	start := strings.Index(from, "<")
	end := strings.Index(from, ">")
	if start >= 0 && end > start {
		return from[start+1 : end]
	}
	return from
}

// NoopSender is a no-op sender for development/testing.
type NoopSender struct{}

// NewNoopSender creates a sender that does nothing (for testing).
func NewNoopSender() *NoopSender { return &NoopSender{} }

func (n *NoopSender) Send(_ context.Context, _ *Message) error { return nil }
func (n *NoopSender) SendBatch(_ context.Context, _ []*Message) error { return nil }

// LogSender logs emails instead of sending them (for development).
type LogSender struct {
	LogFunc func(format string, args ...interface{})
}

// NewLogSender creates a sender that logs emails.
func NewLogSender(logFunc func(format string, args ...interface{})) *LogSender {
	return &LogSender{LogFunc: logFunc}
}

func (l *LogSender) Send(_ context.Context, msg *Message) error {
	if l.LogFunc != nil {
		l.LogFunc("[EMAIL] To=%v Subject=%s", msg.To, msg.Subject)
	}
	return nil
}

func (l *LogSender) SendBatch(_ context.Context, msgs []*Message) error {
	for _, msg := range msgs {
		if err := l.Send(context.Background(), msg); err != nil {
			return err
		}
	}
	return nil
}
