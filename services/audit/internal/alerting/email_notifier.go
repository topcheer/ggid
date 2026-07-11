package alerting

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// EmailConfig holds SMTP server configuration for email notifications.
type EmailConfig struct {
	Host     string // SMTP server hostname
	Port     int    // SMTP server port (587 for STARTTLS, 465 for SSL)
	Username string // SMTP auth username
	Password string // SMTP auth password
	From     string // sender email address
	UseTLS   bool   // use explicit TLS (port 465); otherwise STARTTLS
}

// EmailNotifier sends alert notifications via email.
type EmailNotifier struct {
	cfg EmailConfig
}

// NewEmailNotifier creates a new email notifier with the given SMTP config.
func NewEmailNotifier(cfg EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

// Notify sends an email alert to all email-type actions in the rule.
func (e *EmailNotifier) Notify(ctx context.Context, alert *Alert, actions []AlertAction) error {
	for _, action := range actions {
		if action.Type != "email" || action.Target == "" {
			continue
		}

		subject := action.Params["subject"]
		if subject == "" {
			subject = fmt.Sprintf("[GGID Alert] %s triggered", alert.RuleName)
		}

		body := fmt.Sprintf(
			"Alert: %s\nRule: %s\nTenant: %s\nTrigger: %s\nCount: %d\nFired At: %s\n",
			alert.RuleName,
			alert.RuleID,
			alert.TenantID,
			alert.Trigger,
			alert.Count,
			alert.FiredAt.Format("2006-01-02 15:04:05 UTC"),
		)

		if err := e.sendMail(action.Target, subject, body); err != nil {
			slog.Error("email alert failed", "rule", alert.RuleName, "recipient", action.Target, "error", err)
			return fmt.Errorf("send email to %s: %w", action.Target, err)
		}
		slog.Info("email alert sent", "rule", alert.RuleName, "recipient", action.Target)
	}
	return nil
}

// sendMail sends a plain-text email via SMTP.
func (e *EmailNotifier) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)
	auth := smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.Host)

	msg := strings.Join([]string{
		"From: " + e.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if e.cfg.UseTLS {
		return e.sendMailTLS(addr, auth, e.cfg.From, []string{to}, []byte(msg))
	}
	return smtp.SendMail(addr, auth, e.cfg.From, []string{to}, []byte(msg))
}

// sendMailTLS sends email over a direct TLS connection (for port 465).
func (e *EmailNotifier) sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: e.cfg.Host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, e.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()

	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err = c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return c.Quit()
}
