package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// --- CIBA Backchannel Authentication Flow Tests ---

func TestCIBA_BackchannelAuth_MissingHints(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	tenantID := uuid.New()
	clientID := "ciba-nohint-client"

	repo.clients[clientID] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ClientID:      clientID,
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code", "urn:openid:params:grant-type:ciba"},
		RedirectURIs:  []string{"https://example.com/cb"},
		Enabled:       true,
	}

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:     tenantID,
		ClientID:     clientID,
		ClientSecret: "secret-1",
	})
	if err == nil {
		t.Fatal("expected error when no login hint provided")
	}
}

func TestCIBA_BackchannelAuth_InvalidClient(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	_, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  tenantID,
		ClientID:  "nonexistent-client",
		LoginHint: "user@example.com",
	})
	if err == nil {
		t.Fatal("expected error for invalid client")
	}
}

func TestCIBA_BackchannelAuth_Success(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	tenantID := uuid.New()
	clientID := "ciba-success-client"

	repo.clients[clientID] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ClientID:      clientID,
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code", "urn:openid:params:grant-type:ciba"},
		RedirectURIs:  []string{"https://example.com/cb"},
		Enabled:       true,
	}

	resp, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:       tenantID,
		ClientID:       clientID,
		LoginHint:      "user@example.com",
		Scope:          "openid profile",
		BindingMessage: "Approve login from ACME Corp",
	})
	if err != nil {
		t.Fatalf("BackchannelAuthentication failed: %v", err)
	}
	if resp.AuthReqID == "" {
		t.Error("expected non-empty auth_req_id")
	}
	if resp.ExpiresIn <= 0 {
		t.Error("expected positive expires_in")
	}
	if resp.Interval <= 0 {
		t.Error("expected positive interval")
	}
}

func TestCIBA_PollToken_Pending(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	tenantID := uuid.New()
	clientID := "ciba-poll-client"

	repo.clients[clientID] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ClientID:      clientID,
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code", "urn:openid:params:grant-type:ciba"},
		RedirectURIs:  []string{"https://example.com/cb"},
		Enabled:       true,
	}

	resp, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:  tenantID,
		ClientID:  clientID,
		LoginHint: "user@example.com",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Poll immediately — should get pending error.
	_, err = svc.PollCIBAToken(context.Background(), tenantID, resp.AuthReqID, clientID, "")
	if err == nil {
		t.Fatal("expected pending error")
	}
}

func TestCIBA_PollToken_AuthReqIDNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.PollCIBAToken(context.Background(), uuid.New(), "nonexistent-req-id", "test-client", "secret-1")
	if err == nil {
		t.Fatal("expected error for unknown auth_req_id")
	}
}

func TestCIBA_StatusTransitions(t *testing.T) {
	if CIBAStatusPending != "pending" {
		t.Errorf("expected pending, got %s", CIBAStatusPending)
	}
	if CIBAStatusApproved != "approved" {
		t.Errorf("expected approved, got %s", CIBAStatusApproved)
	}
	if CIBAStatusDenied != "denied" {
		t.Errorf("expected denied, got %s", CIBAStatusDenied)
	}
	if CIBAStatusExpired != "expired" {
		t.Errorf("expected expired, got %s", CIBAStatusExpired)
	}
}

func TestCIBA_DefaultConstants(t *testing.T) {
	if cibaDefaultExpiry != 300 {
		t.Errorf("expected 300, got %d", cibaDefaultExpiry)
	}
	if cibaDefaultInterval != 5 {
		t.Errorf("expected 5, got %d", cibaDefaultInterval)
	}
}

func TestCIBA_RequestedExpiry(t *testing.T) {
	svc, repo, _, _ := newTestOAuthService()
	tenantID := uuid.New()
	clientID := "ciba-expiry-client"

	repo.clients[clientID] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ClientID:      clientID,
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code", "urn:openid:params:grant-type:ciba"},
		RedirectURIs:  []string{"https://example.com/cb"},
		Enabled:       true,
	}

	resp, err := svc.BackchannelAuthentication(context.Background(), &BackchannelAuthRequest{
		TenantID:        tenantID,
		ClientID:        clientID,
		LoginHint:       "user@example.com",
		RequestedExpiry: 120,
	})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if resp.ExpiresIn > 120 {
		t.Errorf("expected expires_in <= 120, got %d", resp.ExpiresIn)
	}
}

var _ = time.Now
