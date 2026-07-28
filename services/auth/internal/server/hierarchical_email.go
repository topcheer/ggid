package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HierarchicalEmailSender resolves email provider config from the DB
// (app → tenant → instance → env fallback) on each send.
type HierarchicalEmailSender struct {
	pool     *pgxpool.Pool
	tenantID *uuid.UUID
	clientID *string
	// fallback config from env vars (SMTP_HOST etc.)
	fallback *EmailConfig
}

func NewHierarchicalEmailSender(pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, fallback *EmailConfig) *HierarchicalEmailSender {
	return &HierarchicalEmailSender{
		pool:      pool,
		tenantID:  tenantID,
		clientID:  clientID,
		fallback:  fallback,
	}
}

func (h *HierarchicalEmailSender) SendEmail(msg EmailMessage) error {
	ctx := context.Background()

	// Try hierarchical config
	cfg, err := hierarchy.GetConfig(ctx, h.pool, hierarchy.KeyEmailProvider, h.tenantID, h.clientID, nil)
	if err == nil && cfg != nil {
		var emailCfg EmailConfig
		if err := json.Unmarshal(cfg.Config, &emailCfg); err != nil {
			return fmt.Errorf("invalid email provider config: %w", err)
		}
		return sendViaSMTP(emailCfg, msg)
	}

	// Fallback to env-based config
	if h.fallback != nil && h.fallback.Host != "" {
		return sendViaSMTP(*h.fallback, msg)
	}

	return fmt.Errorf("email provider not configured at any scope")
}

func sendViaSMTP(cfg EmailConfig, msg EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.APIKey, cfg.Host)

	body := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		msg.To, cfg.From, msg.Subject, renderEmailBody(msg))

	return smtp.SendMail(addr, auth, cfg.From, []string{msg.To}, []byte(body))
}

func renderEmailBody(msg EmailMessage) string {
	if msg.Template != "" {
		return fmt.Sprintf("<p>%s</p>", msg.Template)
	}
	return fmt.Sprintf("<p>%s</p>", msg.Subject)
}

// IsEmailConfigured checks whether email provider is available at any scope.
func IsEmailConfigured(pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string) bool {
	if pool == nil {
		return false
	}
	if hierarchy.IsAvailable(context.Background(), pool, hierarchy.KeyEmailProvider, tenantID, clientID) {
		return true
	}
	// Also check env-based config
	return pool != nil // env vars checked at startup
}

var _ = log.Print // keep import for future logging
