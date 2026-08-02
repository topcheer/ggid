package service

// JWT Bearer grant (RFC 7523) and AMR/ACR helpers for OAuthService.
// Extracted from oauth_service.go for maintainability.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *OAuthService) JWTBearerGrant(ctx context.Context, req *JWTBearerRequest) (*TokenResponse, error) {
	if req.Assertion == "" {
		return nil, fmt.Errorf("assertion is required")
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	unverifiedToken, _, err := parser.ParseUnverified(req.Assertion, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("malformed assertion JWT: %w", err)
	}

	claims, ok := unverifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid assertion claims")
	}

	iss, _ := claims["iss"].(string)
	if iss == "" {
		return nil, fmt.Errorf("assertion missing 'iss' claim")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("assertion missing 'sub' claim")
	}

	asPubKey := s.keyProvider.Public()
	token, err := jwt.Parse(req.Assertion, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return asPubKey, nil
	}, jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}))
	if err != nil {
		if req.ClientID != "" {
			externalKey, fetchErr := s.fetchExternalIssuerKey(ctx, iss, req.ClientID, unverifiedToken.Header)
			if fetchErr != nil {
				return nil, fmt.Errorf("assertion signature verification failed (AS key) and no external key available for issuer %q: %w", iss, fetchErr)
			}
			token, err = jwt.Parse(req.Assertion, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return externalKey, nil
			})
			if err != nil {
				return nil, fmt.Errorf("assertion signature verification failed with external key: %w", err)
			}
		} else {
			return nil, fmt.Errorf("assertion signature verification failed: %w", err)
		}
	}

	verifiedClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid assertion claims after verification")
	}

	if s.issuer != "" {
		audMatched := false
		switch aud := verifiedClaims["aud"].(type) {
		case string:
			audMatched = aud == s.issuer
		case []any:
			for _, a := range aud {
				if as, ok := a.(string); ok && as == s.issuer {
					audMatched = true
					break
				}
			}
		}
		if !audMatched {
			return nil, fmt.Errorf("assertion audience does not include issuer %q", s.issuer)
		}
	}

	sub, ok = verifiedClaims["sub"].(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("assertion missing 'sub' claim")
	}

	exp, ok := verifiedClaims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("assertion missing 'exp' claim")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("assertion has expired")
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("assertion sub must be a valid user ID")
	}

	if s.pool != nil {
		var status string
		err := s.pool.QueryRow(ctx, `SELECT status FROM users WHERE id = $1 AND tenant_id = $2`, userID, req.TenantID).Scan(&status)
		if err != nil {
			return nil, fmt.Errorf("assertion sub does not match any known user")
		}
		if status != "active" {
			return nil, fmt.Errorf("user account is not active")
		}
	}

	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	gidClaims := jwt.MapClaims{
		"iss":           s.issuer,
		"sub":           userID.String(),
		"aud":           "ggid",
		"tenant_id":     req.TenantID.String(),
		"iat":           now.Unix(),
		"exp":           expiresAt.Unix(),
		"jti":           uuid.New().String(),
		"assertion_iss": iss,
	}

	gidToken := jwt.NewWithClaims(s.signingMethod(), gidClaims)
	gidToken.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := gidToken.SignedString(s.keyProvider.Signer())
	if err != nil {
		return nil, fmt.Errorf("sign jwt-bearer token: %w", err)
	}

	scopeStr := ""
	if len(req.Scope) > 0 {
		scopeStr = strings.Join(req.Scope, " ")
	}

	return &TokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
		Scope:       scopeStr,
	}, nil
}

func computeAMR(authMethods []string) []string {
	amr := []string{}
	hasMFA := false
	for _, m := range authMethods {
		switch m {
		case "password":
			amr = append(amr, "pwd")
		case "totp", "hotp":
			amr = append(amr, "otp")
			hasMFA = true
		case "webauthn":
			amr = append(amr, "fpt")
			hasMFA = true
		case "sms_otp":
			amr = append(amr, "sms")
			hasMFA = true
		}
	}
	if hasMFA {
		amr = append(amr, "mfa")
	}
	return amr
}

func computeACR(authMethods []string) string {
	hasPwd, hasMFA, hasHardware := false, false, false
	for _, m := range authMethods {
		switch m {
		case "password":
			hasPwd = true
		case "webauthn":
			hasHardware = true
			hasMFA = true
		case "totp", "hotp", "sms_otp":
			hasMFA = true
		}
	}
	if hasHardware {
		return "AAL3"
	}
	if hasMFA {
		return "AAL2"
	}
	if hasPwd {
		return "AAL1"
	}
	return ""
}
