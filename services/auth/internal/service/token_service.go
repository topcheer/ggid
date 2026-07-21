package service

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/domain"
)

// TokenService handles JWT signing and refresh-token lifecycle.
type TokenService struct {
	provider    ggidcrypto.KeyProvider
	keyID       string
	algorithm   ggidcrypto.KeyAlgorithm
	jwtIssuer   string
	jwtAudience string
	jwtTTL      time.Duration
	refreshRepo RefreshTokenRepo
	rdb         *redis.Client
}

// NewTokenService creates a token service backed by the given KeyProvider.
// The provider supplies the signing key and public key for JWT operations.
func NewTokenService(provider ggidcrypto.KeyProvider, issuer, audience string, ttl time.Duration, refreshRepo RefreshTokenRepo, rdb *redis.Client) (*TokenService, error) {
	if provider == nil {
		return nil, fmt.Errorf("key provider is required")
	}
	meta := provider.Metadata()
	if meta.KeyID == "" {
		return nil, fmt.Errorf("key provider metadata missing key ID")
	}
	if meta.Algorithm == "" {
		return nil, fmt.Errorf("key provider metadata missing algorithm")
	}
	return &TokenService{
		provider:    provider,
		keyID:       meta.KeyID,
		algorithm:   meta.Algorithm,
		jwtIssuer:   issuer,
		jwtAudience: audience,
		jwtTTL:      ttl,
		refreshRepo: refreshRepo,
		rdb:         rdb,
	}, nil
}

// AccessTokenClaims contains the JWT custom claims.
type AccessTokenClaims struct {
	TenantID    string   `json:"tenant_id"`
	Scopes      []string `json:"scopes,omitempty"`                // OAuth scopes only (openid, profile, email)
	Permissions []string `json:"permissions,omitempty"`          // Fine-grained: ["inventory:read", "orders:write"]
	Roles       []string `json:"roles,omitempty"`               // Role names: ["ERP Manager", "Viewer"]
	AMR         []string `json:"amr,omitempty"` // Authentication Method References (RFC 8707)
	ACR         string   `json:"acr,omitempty"` // Authentication Context Class Reference (AAL1/AAL2/AAL3)
	jwt.RegisteredClaims
}

// AAL (Authenticator Assurance Level) values per NIST 800-63B.
const (
	AAL1 = "AAL1" // single-factor (password)
	AAL2 = "AAL2" // multi-factor (password + OTP/WebAuthn)
	AAL3 = "AAL3" // hardware-based MFA (WebAuthn with attestation)
)

// AMR method references.
const (
	AMRPwd      = "pwd"      // password
	AMROTP      = "otp"      // TOTP/HOTP
	AMRFIDO     = "fpt"      // FIDO/WebAuthn
	AMRMFA      = "mfa"      // multi-factor authentication
	AMRKerberos = "kerb"     // Kerberos
	AMRSMS      = "sms"      // SMS OTP
	AMREmail    = "email"    // Email OTP
)

// ComputeAMR builds the amr claim from auth methods used.
func ComputeAMR(authMethods []string) []string {
	amr := make([]string, 0, len(authMethods))
	hasMFA := false
	for _, m := range authMethods {
		switch m {
		case "password":
			amr = append(amr, AMRPwd)
		case "totp", "hotp":
			amr = append(amr, AMROTP)
			hasMFA = true
		case "webauthn":
			amr = append(amr, AMRFIDO)
			hasMFA = true
		case "sms_otp":
			amr = append(amr, AMRSMS)
			hasMFA = true
		case "email_otp":
			amr = append(amr, AMREmail)
			hasMFA = true
		}
	}
	if hasMFA {
		amr = append(amr, AMRMFA)
	}
	return amr
}

// ComputeACR determines the ACR (AAL level) from auth methods.
func ComputeACR(authMethods []string) string {
	hasPassword := false
	hasMFA := false
	hasHardware := false
	for _, m := range authMethods {
		switch m {
		case "password":
			hasPassword = true
		case "webauthn":
			hasHardware = true
			hasMFA = true
		case "totp", "hotp", "sms_otp", "email_otp":
			hasMFA = true
		}
	}
	if hasHardware {
		return AAL3
	}
	if hasMFA {
		return AAL2
	}
	if hasPassword {
		return AAL1
	}
	return ""
}

// IssueAccessToken signs a new JWT for the given user.
func (ts *TokenService) IssueAccessToken(tenantID, userID uuid.UUID, scopes []string) (string, int, error) {
	token, _, expiresIn, err := ts.IssueAccessTokenWithJTI(tenantID, userID, scopes, nil, nil)
	return token, expiresIn, err
}

// IssueAccessTokenWithAMR issues a JWT with AMR/ACR claims from auth methods.
func (ts *TokenService) IssueAccessTokenWithAMR(tenantID, userID uuid.UUID, scopes []string, permissions []string, roles []string, authMethods []string) (token, jti string, expiresIn int, err error) {
	now := time.Now()
	expiresAt := now.Add(ts.jwtTTL)
	jti = uuid.New().String()

	claims := AccessTokenClaims{
		TenantID:    tenantID.String(),
		Scopes:      scopes,
		Permissions: permissions,
		Roles:       roles,
		AMR:         ComputeAMR(authMethods),
		ACR:         ComputeACR(authMethods),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.jwtIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{ts.jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	method := jwtSigningMethod(ts.algorithm)
	jwtToken := jwt.NewWithClaims(method, claims)
	jwtToken.Header["kid"] = ts.keyID

	signed, err := jwtToken.SignedString(ts.provider.Signer())
	if err != nil {
		return "", "", 0, err
	}
	return signed, jti, int(expiresAt.Sub(now).Seconds()), nil
}

// IssueAccessTokenWithJTI signs a new JWT and returns the token + jti + expiresIn.
// The jti is needed to write back to the session record for CAE revocation (Phase 2).
func (ts *TokenService) IssueAccessTokenWithJTI(tenantID, userID uuid.UUID, scopes []string, permissions []string, roles []string) (token, jti string, expiresIn int, err error) {
	now := time.Now()
	expiresAt := now.Add(ts.jwtTTL)
	jti = uuid.New().String()

	claims := AccessTokenClaims{
		TenantID:    tenantID.String(),
		Scopes:      scopes,
		Permissions: permissions,
		Roles:       roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.jwtIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{ts.jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	method := jwtSigningMethod(ts.algorithm)
	jwtToken := jwt.NewWithClaims(method, claims)
	jwtToken.Header["kid"] = ts.keyID

	signed, err := jwtToken.SignedString(ts.provider.Signer())
	if err != nil {
		return "", "", 0, fmt.Errorf("sign access token: %w", err)
	}

	return signed, jti, int(ts.jwtTTL.Seconds()), nil
}

// IssueRefreshToken creates a new opaque refresh token, stores its hash in Redis
// and the DB, and returns the plaintext token.
func (ts *TokenService) IssueRefreshToken(ctx context.Context, tenantID, userID, sessionID uuid.UUID) (string, error) {
	plaintext, err := ggidcrypto.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	token := &domain.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: hashToken(plaintext),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := ts.refreshRepo.Create(ctx, token); err != nil {
		return "", fmt.Errorf("persist refresh token: %w", err)
	}

	redisKey := refreshTokenKey(token.TokenHash)
	ttl := time.Until(token.ExpiresAt)
	if err := ts.rdb.Set(ctx, redisKey, token.ID.String(), ttl).Err(); err != nil {
		return "", fmt.Errorf("cache refresh token: %w", err)
	}

	return plaintext, nil
}

// RefreshAccessToken validates a refresh token, rotates it, and returns new tokens.
func (ts *TokenService) RefreshAccessToken(ctx context.Context, plaintext string) (string, *domain.RefreshToken, error) {
	return ts.refreshToken(ctx, plaintext)
}

// RotateRefreshToken is an alias for RefreshAccessToken for backward compatibility.
func (ts *TokenService) RotateRefreshToken(ctx context.Context, plaintext string) (string, *domain.RefreshToken, error) {
	return ts.refreshToken(ctx, plaintext)
}

func (ts *TokenService) refreshToken(ctx context.Context, plaintext string) (string, *domain.RefreshToken, error) {
	if plaintext == "" {
		return "", nil, fmt.Errorf("refresh token is empty")
	}

	tokenHash := hashToken(plaintext)

	// Look up in Redis first; if we have a cached ID, validate by hash anyway to avoid
	// depending on a RefreshTokenRepo.Get method that may not exist.
	redisKey := refreshTokenKey(tokenHash)
	idStr, _ := ts.rdb.Get(ctx, redisKey).Result()
	if idStr != "" {
		if _, parseErr := uuid.Parse(idStr); parseErr == nil {
			oldToken, lookupErr := ts.refreshRepo.FindByHash(ctx, tokenHash)
			if lookupErr == nil && oldToken != nil && oldToken.IsActive() {
				return ts.rotateToken(ctx, oldToken)
			}
		}
	}

	// Fallback to DB lookup
	oldToken, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		return "", nil, fmt.Errorf("find refresh token: %w", err)
	}
	if oldToken == nil {
		return "", nil, fmt.Errorf("invalid refresh token")
	}
	if !oldToken.IsActive() {
		_ = ts.refreshRepo.RevokeAllForSession(ctx, oldToken.SessionID)
		return "", nil, fmt.Errorf("refresh token replay detected — session revoked")
	}

	return ts.rotateToken(ctx, oldToken)
}

func (ts *TokenService) rotateToken(ctx context.Context, oldToken *domain.RefreshToken) (string, *domain.RefreshToken, error) {
	// Revoke the old token
	if err := ts.refreshRepo.Revoke(ctx, oldToken.ID); err != nil {
		return "", nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// Issue new token
	newPlaintext, err := ggidcrypto.GenerateRandomToken(32)
	if err != nil {
		return "", nil, fmt.Errorf("generate new refresh token: %w", err)
	}
	newHash := hashToken(newPlaintext)

	newToken := &domain.RefreshToken{
		ID:          uuid.New(),
		TenantID:    oldToken.TenantID,
		UserID:      oldToken.UserID,
		SessionID:   oldToken.SessionID,
		TokenHash:   newHash,
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
		RotatedFrom: &oldToken.ID,
		CreatedAt:   time.Now(),
	}

	if err := ts.refreshRepo.Create(ctx, newToken); err != nil {
		return "", nil, fmt.Errorf("persist new refresh token: %w", err)
	}

	newRedisKey := refreshTokenKey(newHash)
	ttl := time.Until(newToken.ExpiresAt)
	if err := ts.rdb.Set(ctx, newRedisKey, newToken.ID.String(), ttl).Err(); err != nil {
		return "", nil, fmt.Errorf("cache new refresh token: %w", err)
	}

	return newPlaintext, newToken, nil
}

// RevokeRefreshToken revokes a refresh token by its plaintext value.
func (ts *TokenService) RevokeRefreshToken(ctx context.Context, plaintext string) error {
	tokenHash := hashToken(plaintext)

	// Delete from Redis
	ts.rdb.Del(ctx, refreshTokenKey(tokenHash))

	// Revoke in DB
	rt, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		return err
	}
	if rt == nil {
		return nil // already gone
	}
	return ts.refreshRepo.Revoke(ctx, rt.ID)
}

// RevokeAllForSession revokes all refresh tokens for a session.
func (ts *TokenService) RevokeAllForSession(ctx context.Context, sessionID uuid.UUID) error {
	return ts.refreshRepo.RevokeAllForSession(ctx, sessionID)
}

// RevokeAllForUser revokes all refresh tokens for a user (global logout).
func (ts *TokenService) RevokeAllForUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	return ts.refreshRepo.RevokeAllForUser(ctx, tenantID, userID)
}

// PublicKey returns the RSA public key for JWT verification (backward compatibility).
// Returns nil if the underlying key is not RSA.
func (ts *TokenService) PublicKey() *rsa.PublicKey {
	return ts.provider.Public().(*rsa.PublicKey)
}

// KeyID returns the key identifier used in JWT headers and JWKS.
func (ts *TokenService) KeyID() string {
	return ts.keyID
}

// Provider returns the underlying KeyProvider.
func (ts *TokenService) Provider() ggidcrypto.KeyProvider {
	return ts.provider
}

// --- helpers ---

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func refreshTokenKey(hash string) string {
	return "ggid:rt:" + hash
}

func jwtSigningMethod(alg ggidcrypto.KeyAlgorithm) jwt.SigningMethod {
	switch alg {
	case ggidcrypto.RS256:
		return jwt.SigningMethodRS256
	case ggidcrypto.RS384:
		return jwt.SigningMethodRS384
	case ggidcrypto.RS512:
		return jwt.SigningMethodRS512
	case ggidcrypto.ES256:
		return jwt.SigningMethodES256
	case ggidcrypto.ES384:
		return jwt.SigningMethodES384
	case ggidcrypto.ES512:
		return jwt.SigningMethodES512
	case ggidcrypto.EdDSA:
		return jwt.SigningMethodEdDSA
	case ggidcrypto.SM2SM3:
		return ggidcrypto.SigningMethodSM2
	default:
		return jwt.SigningMethodRS256
	}
}

// compile-time assertion
var _ crypto.Signer = (*rsa.PrivateKey)(nil)

// parsePublicKey parses an RSA public key from PEM-encoded PKIX or PKCS1 data.
func parsePublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		rsaPub, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		return rsaPub, nil
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaPub, nil
}

// parsePrivateKey parses an RSA private key from PEM-encoded PKCS1 or PKCS8 data.
func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		keyAny, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %w (pkcs8: %v)", err, err2)
		}
		rsaKey, ok := keyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	}
	return key, nil
}

// loadOrCreatePrivateKey loads an RSA private key from disk, generating one if missing.
func loadOrCreatePrivateKey(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return parsePrivateKey(data)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	pubPath := derivePublicKeyPath(path)
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pubData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	_ = os.WriteFile(pubPath, pubData, 0o644)
	return key, nil
}

func derivePublicKeyPath(privateKeyPath string) string {
	if len(privateKeyPath) > 11 && privateKeyPath[len(privateKeyPath)-11:] == "private.pem" {
		return privateKeyPath[:len(privateKeyPath)-11] + "public.pem"
	}
	if len(privateKeyPath) > 4 && privateKeyPath[len(privateKeyPath)-4:] == ".pem" {
		return privateKeyPath[:len(privateKeyPath)-4] + ".pub"
	}
	return privateKeyPath + ".pub"
}
