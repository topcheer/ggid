package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmailConfig holds email provider settings.
type EmailConfig struct {
	Provider  string `json:"provider"`  // smtp, sendgrid, ses, mailgun
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	From     string `json:"from_email"`
	UseTLS   bool   `json:"use_tls"`
	APIKey   string `json:"-"` // never expose
}

// EmailMessage represents an outbound email.
type EmailMessage struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data"`
}

// emailRepo manages email config + log in PG.
type emailRepo struct {
	pool   *pgxpool.Pool
	config EmailConfig
}

func newEmailRepo(pool *pgxpool.Pool) *emailRepo {
	return &emailRepo{
		pool: pool,
		config: EmailConfig{Provider: "smtp", Host: "localhost", Port: 587, UseTLS: true, From: "noreply@ggid.local"},
	}
}

func (r *emailRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS email_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			to_addr TEXT NOT NULL, subject TEXT NOT NULL,
			template TEXT NOT NULL DEFAULT '', status TEXT DEFAULT 'pending',
			sent_at TIMESTAMPTZ, error TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_email_status ON email_log(status);
	`)
	return err
}

func (r *emailRepo) LogEmail(ctx context.Context, to, subject, template, status, errMsg string) {
	if r.pool == nil { return }
	r.pool.Exec(ctx, `INSERT INTO email_log (to_addr,subject,template,status,error,sent_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		to, subject, template, status, errMsg, time.Now().UTC())
}

// SendEmail sends via SMTP (with retry). Returns error if all retries fail.
func (r *emailRepo) SendEmail(ctx context.Context, msg *EmailMessage) error {
	body := r.renderTemplate(msg)
	fullMsg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		r.config.From, msg.To, msg.Subject, body)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		err := r.sendSMTP(msg.To, fullMsg)
		if err == nil {
			r.LogEmail(ctx, msg.To, msg.Subject, msg.Template, "sent", "")
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second) // backoff: 2s, 4s, 6s
	}
	r.LogEmail(ctx, msg.To, msg.Subject, msg.Template, "failed", lastErr.Error())
	return lastErr
}

func (r *emailRepo) sendSMTP(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)
	var auth smtp.Auth
	if r.config.Username != "" {
		auth = smtp.PlainAuth("", r.config.Username, r.config.APIKey, r.config.Host)
	}
	if r.config.UseTLS {
		return sendEmailTLS(addr, auth, r.config.From, []string{to}, []byte(msg))
	}
	return smtp.SendMail(addr, auth, r.config.From, []string{to}, []byte(msg))
}

func sendEmailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil { return err }
	defer client.Close()
	if err := client.StartTLS(&tls.Config{ServerName: strings.Split(addr, ":")[0]}); err != nil { return err }
	if auth != nil { if err := client.Auth(auth); err != nil { return err } }
	if err := client.Mail(from); err != nil { return err }
	for _, r := range to { if err := client.Rcpt(r); err != nil { return err } }
	w, err := client.Data()
	if err != nil { return err }
	defer w.Close()
	_, err = w.Write(msg)
	return err
}

// renderTemplate renders email body from template name + data.
func (r *emailRepo) renderTemplate(msg *EmailMessage) string {
	switch msg.Template {
	case "email_verification":
		return fmt.Sprintf(`<html><body><h2>Verify Your Email</h2><p>Click <a href="%s/verify-email?token=%s">here</a> to verify your email.</p><p>This link expires in 24 hours.</p></body></html>`,
			msg.Data["base_url"], msg.Data["token"])
	case "password_reset":
		return fmt.Sprintf(`<html><body><h2>Reset Your Password</h2><p>Click <a href="%s/reset-password?token=%s">here</a> to reset your password.</p><p>This link expires in 30 minutes.</p></body></html>`,
			msg.Data["base_url"], msg.Data["token"])
	case "breach_notification":
		return fmt.Sprintf(`<html><body><h2>Security Alert</h2><p>%s</p><p>If this wasn't you, please contact your administrator immediately.</p></body></html>`,
			msg.Data["message"])
	case "test":
		return `<html><body><h2>Test Email</h2><p>This is a test email from GGID.</p></body></html>`
	default:
		return fmt.Sprintf(`<html><body><p>%s</p></body></html>`, msg.Data["body"])
	}
}

// --- HTTP Handlers ---

func (h *Handler) handleEmailConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := EmailConfig{Provider: "smtp", Host: "localhost", Port: 587, UseTLS: true, From: "noreply@ggid.local"}
		if h.emailRepo != nil { cfg = h.emailRepo.config }
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut, http.MethodPost:
		var cfg EmailConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		validProviders := map[string]bool{"smtp": true, "sendgrid": true, "ses": true, "mailgun": true}
		if !validProviders[cfg.Provider] {
			writeError(w, http.StatusBadRequest, "provider must be smtp, sendgrid, ses, or mailgun")
			return
		}
		if h.emailRepo != nil { h.emailRepo.config = cfg }
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "provider": cfg.Provider})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleEmailTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct{ To string `json:"to"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
			return
	}
	if req.To == "" {
		writeError(w, http.StatusBadRequest, "to address required")
		return
	}
	if h.emailRepo != nil {
		go h.emailRepo.SendEmail(nil, &EmailMessage{
			To: req.To, Subject: "GGID Test Email", Template: "test",
			Data: map[string]string{},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "queued", "to": req.To, "message": "test email queued for delivery"})
}

func (h *Handler) SetEmailRepo(repo *emailRepo) {
	h.emailRepo = repo
}

var _ = uuid.New
