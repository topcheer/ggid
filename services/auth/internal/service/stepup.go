package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

const stepUpTokenTTL = 5 * time.Minute

// StepUpChallenge represents the result of initiating a step-up authentication challenge.
type StepUpChallenge struct {
	Challenge string `json:"challenge"`
	Method    string `json:"method"` // "password" or "mfa"
}

// StepUpResult represents the result of completing a step-up challenge.
type StepUpResult struct {
	StepUpToken string `json:"step_up_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// InitStepUp creates a step-up authentication challenge for a user.
// The user must complete the challenge (re-enter password or MFA code)
// to obtain a short-lived step-up token valid for 5 minutes.
// method must be "password" or "mfa".
func (s *AuthService) InitStepUp(ctx context.Context, userID uuid.UUID, method string) (*StepUpChallenge, error) {
	switch method {
	case "password", "mfa":
		// valid
	default:
		return nil, fmt.Errorf("unsupported step-up method: %s", method)
	}

	challenge, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate step-up challenge: %w", err)
	}

	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant context required: %w", err)
	}

	key := fmt.Sprintf("ggid:stepup:%s", challenge)
	val := fmt.Sprintf("%s:%s:%s", tc.TenantID, userID, method)
	if err := s.rateLimiter.rdb.Set(ctx, key, val, stepUpTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store step-up challenge: %w", err)
	}

	return &StepUpChallenge{
		Challenge: challenge,
		Method:    method,
	}, nil
}

// VerifyStepUp completes a step-up authentication challenge.
// For "password" method, the user's password is re-verified.
// For "mfa" method, the user's TOTP code is verified.
// On success, a short-lived step-up token is issued.
func (s *AuthService) VerifyStepUp(ctx context.Context, challenge, code, password string) (*StepUpResult, error) {
	key := fmt.Sprintf("ggid:stepup:%s", challenge)

	val, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	parts := splitColon(val, 3)
	if len(parts) != 3 {
		s.rateLimiter.rdb.Del(ctx, key)
		return nil, ErrInvalidCredentials
	}

	tenantID, err := uuid.Parse(parts[0])
	if err != nil {
		s.rateLimiter.rdb.Del(ctx, key)
		return nil, ErrInvalidCredentials
	}
	userID, err := uuid.Parse(parts[1])
	if err != nil {
		s.rateLimiter.rdb.Del(ctx, key)
		return nil, ErrInvalidCredentials
	}
	method := parts[2]

	switch method {
	case "password":
		cred, err := s.credentialRepo.FindByUserID(ctx, tenantID, userID)
		if err != nil || cred == nil {
			s.rateLimiter.rdb.Del(ctx, key)
			return nil, ErrInvalidCredentials
		}
		match, err := crypto.VerifyPassword(password, cred.Secret)
		if err != nil || !match {
			return nil, ErrInvalidCredentials
		}

	case "mfa":
		if s.mfaService == nil {
			s.rateLimiter.rdb.Del(ctx, key)
			return nil, fmt.Errorf("MFA service not configured")
		}
		if err := s.mfaService.VerifyUserCode(ctx, tenantID, userID, code); err != nil {
			return nil, err
		}

	default:
		s.rateLimiter.rdb.Del(ctx, key)
		return nil, fmt.Errorf("unsupported step-up method: %s", method)
	}

	// Challenge verified — delete the challenge.
	s.rateLimiter.rdb.Del(ctx, key)

	// Issue step-up token (separate from JWT, checked by gateway middleware).
	stepUpToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate step-up token: %w", err)
	}

	tokenKey := fmt.Sprintf("ggid:stepup-token:%s", stepUpToken)
	tokenVal := fmt.Sprintf("%s:%s", tenantID, userID)
	if err := s.rateLimiter.rdb.Set(ctx, tokenKey, tokenVal, stepUpTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store step-up token: %w", err)
	}

	return &StepUpResult{
		StepUpToken: stepUpToken,
		ExpiresIn:   int(stepUpTokenTTL.Seconds()),
	}, nil
}

// ValidateStepUpToken checks whether a step-up token is valid for the given user.
// Returns nil if valid, ErrInvalidCredentials if invalid or expired.
func (s *AuthService) ValidateStepUpToken(ctx context.Context, token string, userID uuid.UUID) error {
	key := fmt.Sprintf("ggid:stepup-token:%s", token)
	val, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	if err != nil {
		return ErrInvalidCredentials
	}

	parts := splitColon(val, 2)
	if len(parts) != 2 {
		return ErrInvalidCredentials
	}

	uid, err := uuid.Parse(parts[1])
	if err != nil {
		return ErrInvalidCredentials
	}

	if uid != userID {
		return ErrInvalidCredentials
	}

	return nil
}

// ACRStepUpCheck evaluates whether the current session meets the requested ACR level.
// If not, returns a non-nil StepUpChallenge with the required acr_values.
// Supported ACR levels: urn:mace:incommon:iap:silver (1) < urn:mace:incommon:iap:gold (2).
func (s *AuthService) ACRStepUpCheck(ctx context.Context, userID uuid.UUID, currentACR, requestedACR string) (bool, *StepUpChallenge, error) {
	current := acrLevel(currentACR)
	required := acrLevel(requestedACR)

	if current >= required {
		return true, nil, nil
	}

	// Need step-up: determine the method.
	method := "password"
	if required >= 2 {
		method = "mfa"
	}

	challenge, err := s.InitStepUp(ctx, userID, method)
	if err != nil {
		return false, nil, err
	}

	challenge.Method = method
	return false, challenge, nil
}

// acrLevel maps an ACR string to a numeric level.
func acrLevel(acr string) int {
	switch acr {
	case "urn:mace:incommon:iap:gold":
		return 2
	case "urn:mace:incommon:iap:silver":
		return 1
	case "1":
		return 1
	case "2":
		return 2
	default:
		return 0
	}
}

// splitColon splits s by ':' into at most n parts.
func splitColon(s string, n int) []string {
	result := make([]string, 0, n)
	start := 0
	for i := 0; i < len(s) && len(result) < n-1; i++ {
		if s[i] == ':' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
