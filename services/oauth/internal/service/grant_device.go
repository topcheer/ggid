package service

// Device flow (RFC 8628) methods for OAuthService.
// Extracted from oauth_service.go for maintainability.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *OAuthService) CreateDeviceAuthorization(req *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
	// SECURITY: Rate limit device code creation per client to prevent memory exhaustion DoS.
	// Use a simple in-memory counter with cleanup via the device code expiry.
	deviceCodeMu.Lock()
	now := time.Now()
	cutoff := now.Add(-time.Hour)
	active := 0
	for _, info := range deviceCodeStore {
		if info.CreatedAt.After(cutoff) && info.ClientID == req.ClientID {
			active++
		}
	}
	if active >= 10 {
		deviceCodeMu.Unlock()
		return nil, fmt.Errorf("too many pending device authorization requests, please try again later")
	}
	deviceCodeMu.Unlock()

	// SECURITY (R24 P1): Validate client_id exists and tenant is valid.
	if req.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}

	deviceCode := generateDeviceCode(40)
	userCode := generateUserCode()

	info := &DeviceCodeInfo{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   req.ClientID,
		TenantID:   req.TenantID,
		Scope:      req.Scope,
		Status:     "pending",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	deviceCodeMu.Lock()
	deviceCodeStore[deviceCode] = info
	userCodeIndex[userCode] = deviceCode
	deviceCodeMu.Unlock()

	verificationURI := req.Issuer + "/device"
	if req.Issuer == "" {
		verificationURI = "/device"
	}

	return &DeviceAuthorizationResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: verificationURI,
		ExpiresIn:       900, // 15 minutes
		Interval:        5,   // 5 seconds between polls
	}, nil
}

func (s *OAuthService) PollDeviceToken(ctx context.Context, deviceCode, clientID string) (*TokenResponse, error) {
	deviceCodeMu.RLock()
	info, ok := deviceCodeStore[deviceCode]
	status := ""
	var userID *uuid.UUID
	var tenantID uuid.UUID
	var clientIDStored string
	if ok {
		status = info.Status
		userID = info.UserID
		tenantID = info.TenantID
		clientIDStored = info.ClientID
	}
	deviceCodeMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid_device_code")
	}

	if clientID != "" && clientIDStored != "" && clientID != clientIDStored {
		return nil, fmt.Errorf("invalid_client")
	}

	if time.Now().After(info.ExpiresAt) {
		deviceCodeMu.Lock()
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, info.UserCode)
		deviceCodeMu.Unlock()
		return nil, fmt.Errorf("expired_token")
	}

	if status == "pending" {
		deviceCodeMu.Lock()
		if info.LastPoll != nil && time.Since(*info.LastPoll) < 5*time.Second {
			deviceCodeMu.Unlock()
			return nil, fmt.Errorf("slow_down")
		}
		now := time.Now()
		info.LastPoll = &now
		deviceCodeMu.Unlock()
		return nil, fmt.Errorf("authorization_pending")
	}

	if status == "denied" {
		return nil, fmt.Errorf("access_denied")
	}

	if status == "approved" && userID != nil {
		deviceCodeMu.Lock()
		info2, ok2 := deviceCodeStore[deviceCode]
		if !ok2 || info2.Status != "approved" {
			deviceCodeMu.Unlock()
			return nil, fmt.Errorf("authorization_pending")
		}
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, info2.UserCode)
		deviceCodeMu.Unlock()

		accessToken, expiresIn, err := s.issueDeviceAccessToken(tenantID, *userID)
		if err != nil {
			return nil, err
		}

		scopeStr := ""
		if len(info.Scope) > 0 {
			scopeStr = strings.Join(info.Scope, " ")
		}

		return &TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			Scope:       scopeStr,
		}, nil
	}

	return nil, fmt.Errorf("authorization_pending")
}

func (s *OAuthService) ApproveDeviceCode(userCode string, userID uuid.UUID, tenantID uuid.UUID) error {
	if userID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}
	deviceCodeMu.Lock()
	defer deviceCodeMu.Unlock()

	deviceCode, ok := userCodeIndex[userCode]
	if !ok {
		return fmt.Errorf("invalid user_code")
	}

	info, ok := deviceCodeStore[deviceCode]
	if !ok {
		return fmt.Errorf("device code not found")
	}

	if time.Now().After(info.ExpiresAt) {
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, userCode)
		return fmt.Errorf("expired user_code")
	}

	if info.TenantID != uuid.Nil && tenantID != uuid.Nil && info.TenantID != tenantID {
		return fmt.Errorf("tenant mismatch")
	}

	info.Status = "approved"
	info.UserID = &userID
	return nil
}

func (s *OAuthService) DenyDeviceCode(userCode string) error {
	deviceCodeMu.Lock()
	defer deviceCodeMu.Unlock()

	deviceCode, ok := userCodeIndex[userCode]
	if !ok {
		return fmt.Errorf("invalid user_code")
	}

	info, ok := deviceCodeStore[deviceCode]
	if !ok {
		return fmt.Errorf("device code not found")
	}

	if time.Now().After(info.ExpiresAt) {
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, userCode)
		return fmt.Errorf("expired user_code")
	}

	info.Status = "denied"
	return nil
}

func (s *OAuthService) issueDeviceAccessToken(tenantID, userID uuid.UUID) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	claims := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       userID.String(),
		"aud":       "ggid",
		"tenant_id": tenantID.String(),
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"jti":       uuid.New().String(),
	}

	token := jwt.NewWithClaims(s.signingMethod(), claims)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return "", 0, fmt.Errorf("sign device token: %w", err)
	}

	return signed, int(time.Until(expiresAt).Seconds()), nil
}

func generateDeviceCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[cryptoRandInt(len(charset))]
	}
	return string(b)
}

func generateUserCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	part1 := make([]byte, 4)
	part2 := make([]byte, 4)
	for i := range part1 {
		part1[i] = charset[cryptoRandInt(len(charset))]
	}
	for i := range part2 {
		part2[i] = charset[cryptoRandInt(len(charset))]
	}
	return string(part1) + "-" + string(part2)
}

// Ensure domain import is used
var _ = domain.OAuthClient{}
