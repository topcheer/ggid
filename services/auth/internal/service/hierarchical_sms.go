package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ggid/ggid/pkg/hierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HierarchicalSMSSender resolves SMS provider config from the DB
// (app → tenant → instance → env fallback) on each SendSMS call.
// This ensures runtime config changes (via Console/API) take effect
// without pod restart.
type HierarchicalSMSSender struct {
	pool     *pgxpool.Pool
	tenantID *uuid.UUID
	clientID *string
	// fallbackSender is used when no DB config exists (env vars).
	fallback SMSSender
}

// NewHierarchicalSMSSender creates an SMS sender that reads from the
// hierarchical config system. The fallback sender (from env vars)
// is used when no DB config is found.
func NewHierarchicalSMSSender(pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string, fallback SMSSender) *HierarchicalSMSSender {
	return &HierarchicalSMSSender{
		pool:           pool,
		tenantID:       tenantID,
		clientID:       clientID,
		fallback:       fallback,
	}
}

// SMSProviderConfig holds the DB-stored SMS provider settings.
type SMSProviderConfig struct {
	Provider   string `json:"provider"`    // "twilio", "sns", "log", "http_webhook"
	AccountSID string `json:"account_sid"` // Twilio
	AuthToken  string `json:"auth_token"`  // Twilio
	FromNumber string `json:"from_number"` // Twilio
	Region     string `json:"region"`      // SNS
	AccessKey  string `json:"access_key"`  // SNS
	SecretKey  string `json:"secret_key"`  // SNS
	// HTTPWebhook: custom HTTP endpoint for SMS sending
	HTTPWebhook *HTTPProviderConfig `json:"http_webhook,omitempty"`
}

func (s *HierarchicalSMSSender) SendSMS(to, message string) error {
	ctx := context.Background()

	// Try hierarchical config lookup
	cfg, err := hierarchy.GetConfig(ctx, s.pool, hierarchy.KeySMSProvider, s.tenantID, s.clientID, nil)
	if err != nil || cfg == nil {
		// Fallback to env-based sender
		if s.fallback != nil {
			return s.fallback.SendSMS(to, message)
		}
		return fmt.Errorf("SMS provider not configured at any scope")
	}

	var smsCfg SMSProviderConfig
	if err := json.Unmarshal(cfg.Config, &smsCfg); err != nil {
		return fmt.Errorf("invalid SMS provider config: %w", err)
	}

	switch smsCfg.Provider {
	case "twilio":
		sender := &TwilioSMSSender{
			accountSID: smsCfg.AccountSID,
			authToken:  smsCfg.AuthToken,
			fromNumber: smsCfg.FromNumber,
		}
		return sender.SendSMS(to, message)
	case "sns":
		sender := &AWSSNSSMSSender{
			accessKey: smsCfg.AccessKey,
			secretKey: smsCfg.SecretKey,
			region:    smsCfg.Region,
		}
		return sender.SendSMS(to, message)
	case "http_webhook":
		if smsCfg.HTTPWebhook == nil {
			return fmt.Errorf("http_webhook config is empty")
		}
		return executeHTTPProvider(*smsCfg.HTTPWebhook, map[string]string{
			"phone":   to,
			"message": message,
		})
	case "log":
		return (&LogSMSSender{}).SendSMS(to, message)
	default:
		log.Printf("SMS provider config source=%s type=%s — using fallback", cfg.Source, smsCfg.Provider)
		if s.fallback != nil {
			return s.fallback.SendSMS(to, message)
		}
		return fmt.Errorf("unknown SMS provider type: %s", smsCfg.Provider)
	}
}

// IsSMSConfigured checks whether SMS provider is available at any scope.
func IsSMSConfigured(pool *pgxpool.Pool, tenantID *uuid.UUID, clientID *string) bool {
	if pool == nil {
		return false
	}
	return hierarchy.IsAvailable(context.Background(), pool, hierarchy.KeySMSProvider, tenantID, clientID)
}
