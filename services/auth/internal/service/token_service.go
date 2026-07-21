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
