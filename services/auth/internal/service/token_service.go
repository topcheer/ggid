package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenService handles JWT signing and refresh-token lifecycle.
type TokenService struct {
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	keyID       string
	jwtCfg      conf.JWTConfig
	refreshRepo RefreshTokenRepo
	rdb         *redis.Client
}

// NewTokenService loads RSA keys and returns a ready token service.
func NewTokenService(cfg conf.JWTConfig, refreshRepo RefreshTokenRepo, rdb *redis.Client) (*TokenService, error) {
	privKey, err := loadOrCreatePrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	var pubKey *rsa.PublicKey
	if data, err := os.ReadFile(cfg.PublicKeyPath); err == nil {
		pubKey, err = parsePublicKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
	} else {
		// Derive public key from private key
		pubKey = &privKey.PublicKey
		// Write public key for external consumers (gateway, JWKS)
		_ = writePublicKey(cfg.PublicKeyPath, pubKey)
	}

	// Key ID is a fingerprint of the public key for JWKS identification.
	keyID := keyFingerprint(pubKey)

	return &TokenService{
		privateKey:  privKey,
		publicKey:   pubKey,
		keyID:       keyID,
		jwtCfg:      cfg,
		refreshRepo: refreshRepo,
		rdb:         rdb,
	}, nil
}

// AccessTokenClaims contains the JWT custom claims.
type AccessTokenClaims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// IssueAccessToken signs a new RS256 JWT for the given user.
func (ts *TokenService) IssueAccessToken(tenantID, userID uuid.UUID) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(ts.jwtCfg.AccessTokenTTL)

	claims := AccessTokenClaims{
		TenantID: tenantID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.jwtCfg.Issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{ts.jwtCfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = ts.keyID

	signed, err := token.SignedString(ts.privateKey)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	return signed, int(ts.jwtCfg.AccessTokenTTL.Seconds()), nil
}

// IssueRefreshToken creates a new opaque refresh token, stores its hash in Redis
// and the DB, and returns the plaintext token.
func (ts *TokenService) IssueRefreshToken(ctx context.Context, tenantID, userID, sessionID uuid.UUID) (string, error) {
	plaintext, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	tokenHash := hashToken(plaintext)
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
	}

	// Persist to DB
	if err := ts.refreshRepo.Create(ctx, rt); err != nil {
		return "", fmt.Errorf("persist refresh token: %w", err)
	}

	// Cache in Redis for fast lookup: rt:{hash} -> token_id
	redisKey := refreshTokenKey(tokenHash)
	ttl := time.Until(rt.ExpiresAt)
	if err := ts.rdb.Set(ctx, redisKey, rt.ID.String(), ttl).Err(); err != nil {
		return "", fmt.Errorf("cache refresh token: %w", err)
	}

	return plaintext, nil
}

// RotateRefreshToken revokes the old token and issues a new one linked via rotated_from.
// Returns the new plaintext token. If the old token is already revoked or expired,
// an error is returned (potential replay attack).
func (ts *TokenService) RotateRefreshToken(ctx context.Context, plaintext string) (string, *domain.RefreshToken, error) {
	tokenHash := hashToken(plaintext)

	// Fast path: check Redis
	redisKey := refreshTokenKey(tokenHash)
	if err := ts.rdb.Del(ctx, redisKey).Err(); err != nil && err != redis.Nil {
		return "", nil, fmt.Errorf("revoke refresh token cache: %w", err)
	}

	// Authoritative: check DB
	oldToken, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		return "", nil, fmt.Errorf("find refresh token: %w", err)
	}
	if oldToken == nil {
		return "", nil, fmt.Errorf("refresh token is invalid or expired")
	}

	// Replay attack detection: if the token has already been revoked,
	// an attacker may be reusing a stolen old token. Revoke ALL tokens
	// for the same session to invalidate the entire chain (RFC 6749 §10.4).
	if !oldToken.IsActive() {
		_ = ts.refreshRepo.RevokeAllForSession(ctx, oldToken.SessionID)
		return "", nil, fmt.Errorf("refresh token replay detected — session revoked")
	}

	// Revoke the old token
	if err := ts.refreshRepo.Revoke(ctx, oldToken.ID); err != nil {
		return "", nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// Issue new token
	newPlaintext, err := crypto.GenerateRandomToken(32)
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

// PublicKey returns the RSA public key for JWT verification.
func (ts *TokenService) PublicKey() *rsa.PublicKey {
	return ts.publicKey
}

// KeyID returns the key identifier used in JWT headers and JWKS.
func (ts *TokenService) KeyID() string {
	return ts.keyID
}

// --- helpers ---

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func refreshTokenKey(hash string) string {
	return "ggid:rt:" + hash
}

func keyFingerprint(pub *rsa.PublicKey) string {
	data, _ := x509.MarshalPKIXPublicKey(pub)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func loadOrCreatePrivateKey(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return parsePrivateKey(data)
	}
	// Generate new 2048-bit RSA key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	if err := writePrivateKey(path, key); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	return key, nil
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %w (pkcs8: %v)", err, err2)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}
	return key, nil
}

func writePrivateKey(path string, key *rsa.PrivateKey) error {
	_ = os.MkdirAll("configs", 0o700)
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return os.WriteFile(path, data, 0o600)
}

func parsePublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		rsaPub, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		return rsaPub, nil
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	return rsaPub, nil
}

func writePublicKey(path string, pub *rsa.PublicKey) error {
	_ = os.MkdirAll("configs", 0o700)
	data, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: data,
	})
	return os.WriteFile(path, pemData, 0o644)
}
