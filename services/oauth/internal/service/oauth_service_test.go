package service

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- Mock Repositories ---

type mockClientRepo struct {
	clients map[string]*domain.OAuthClient // keyed by client_id
}

func newMockClientRepo() *mockClientRepo {
	return &mockClientRepo{clients: make(map[string]*domain.OAuthClient)}
}

func (m *mockClientRepo) CreateClient(_ context.Context, client *domain.OAuthClient) error {
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()
	m.clients[client.ClientID] = client
	return nil
}

func (m *mockClientRepo) GetClientByID(_ context.Context, _ uuid.UUID, clientID string) (*domain.OAuthClient, error) {
	c, ok := m.clients[clientID]
	if !ok {
		return nil, errors.NotFound("client", clientID)
	}
	return c, nil
}

func (m *mockClientRepo) ListClients(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.OAuthClient, int, error) {
	var result []*domain.OAuthClient
	for _, c := range m.clients {
		result = append(result, c)
	}
	return result, len(result), nil
}

func (m *mockClientRepo) UpdateClient(_ context.Context, _ uuid.UUID, clientID string, client *domain.OAuthClient) (*domain.OAuthClient, error) {
	existing, ok := m.clients[clientID]
	if !ok {
		return nil, errors.NotFound("client", clientID)
	}
	existing.Name = client.Name
	existing.RedirectURIs = client.RedirectURIs
	existing.Scopes = client.Scopes
	existing.Enabled = client.Enabled
	existing.ClientSecretHash = client.ClientSecretHash
	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (m *mockClientRepo) DeleteClient(_ context.Context, _ uuid.UUID, clientID string) error {
	delete(m.clients, clientID)
	return nil
}

type mockCodeRepo struct {
	codes map[string]*domain.AuthorizationCode // keyed by code_hash
}

func newMockCodeRepo() *mockCodeRepo {
	return &mockCodeRepo{codes: make(map[string]*domain.AuthorizationCode)}
}

func (m *mockCodeRepo) CreateCode(_ context.Context, code *domain.AuthorizationCode) error {
	code.CreatedAt = time.Now()
	m.codes[code.CodeHash] = code
	return nil
}

func (m *mockCodeRepo) ConsumeCode(_ context.Context, codeHash string) (*domain.AuthorizationCode, error) {
	code, ok := m.codes[codeHash]
	if !ok || code.Used || code.IsExpired() {
		return nil, errors.InvalidArgument("invalid or expired authorization code")
	}
	code.Used = true
	return code, nil
}

func (m *mockCodeRepo) ResolveTenantFromCode(_ context.Context, codeHash string) (uuid.UUID, error) {
	code, ok := m.codes[codeHash]
	if !ok || code.Used || code.IsExpired() {
		return uuid.Nil, errors.InvalidArgument("invalid or expired authorization code")
	}
	return code.TenantID, nil
}

type mockTokenRepo struct {
	tokens         []*domain.IDTokenRecord
	refreshTokens  []*domain.RefreshTokenRecord
}

func (m *mockTokenRepo) RecordIDToken(_ context.Context, record *domain.IDTokenRecord) error {
	m.tokens = append(m.tokens, record)
	return nil
}
func (m *mockTokenRepo) StoreRefreshToken(_ context.Context, record *domain.RefreshTokenRecord) error {
	m.refreshTokens = append(m.refreshTokens, record)
	return nil
}
func (m *mockTokenRepo) GetRefreshToken(_ context.Context, _ uuid.UUID, tokenHash string) (*domain.RefreshTokenRecord, error) {
	for _, rt := range m.refreshTokens {
		if rt.TokenHash == tokenHash {
			return rt, nil
		}
	}
	return nil, fmt.Errorf("refresh token not found")
}
func (m *mockTokenRepo) RevokeRefreshToken(_ context.Context, _ uuid.UUID, tokenHash string) error {
	for _, rt := range m.refreshTokens {
		if rt.TokenHash == tokenHash {
			rt.Revoked = true
		}
	}
	return nil
}
func (m *mockTokenRepo) RevokeAllRefreshTokens(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }

// --- Mock KeyProvider ---

type mockKeyProvider struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	kid  string
}

var (
	mockKeyOnce sync.Once
	cachedPriv  *rsa.PrivateKey
)

// newMockKeyProvider returns a key provider backed by a single cached RSA
// key. Generating a fresh 2048-bit key on every call starves the CPU when
// dozens of tests run together, causing intermittent panics/timeouts.
func newMockKeyProvider() *mockKeyProvider {
	mockKeyOnce.Do(func() {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			// Use a deterministic key if randomness fails.
			priv = &rsa.PrivateKey{
				PublicKey: rsa.PublicKey{N: big.NewInt(1), E: 65537},
				D:         big.NewInt(1),
			}
		}
		cachedPriv = priv
	})
	return &mockKeyProvider{priv: cachedPriv, pub: &cachedPriv.PublicKey, kid: "test-kid"}
}

func (kp *mockKeyProvider) PublicKey() *rsa.PublicKey   { return kp.pub }
func (kp *mockKeyProvider) PrivateKey() *rsa.PrivateKey { return kp.priv }
func (kp *mockKeyProvider) KeyID() string                { return kp.kid }

// pkg/crypto.KeyProvider implementation.
func (kp *mockKeyProvider) Metadata() ggidcrypto.KeyMetadata {
	return ggidcrypto.KeyMetadata{KeyID: kp.kid, Algorithm: ggidcrypto.RS256, Use: "sig"}
}
func (kp *mockKeyProvider) Public() crypto.PublicKey { return kp.pub }
func (kp *mockKeyProvider) Signer() crypto.Signer    { return kp.priv }
func (kp *mockKeyProvider) Close() error                { return nil }

// --- Helpers ---

var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000020")

func newTestOAuthService() (*OAuthService, *mockClientRepo, *mockCodeRepo, *mockTokenRepo) {
	clientRepo := newMockClientRepo()
	codeRepo := newMockCodeRepo()
	tokenRepo := &mockTokenRepo{}
	kp := newMockKeyProvider()
	svc := NewOAuthService(clientRepo, codeRepo, tokenRepo, kp, "https://test.ggid.dev")
	return svc, clientRepo, codeRepo, tokenRepo
}

// --- Tests ---

func TestCreateClient_Confidential(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "Test App",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code", "refresh_token"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://app.example.com/callback"},
		Scopes:        []string{"openid", "profile", "email"},
	})
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}

	if result.Client.ClientID == "" {
		t.Error("expected non-empty client_id")
	}
	if len(result.Client.ClientID) < 10 || result.Client.ClientID[:5] != "gcid_" {
		t.Errorf("expected client_id to start with 'gcid_', got '%s'", result.Client.ClientID)
	}
	if result.ClientSecret == "" {
		t.Error("expected non-empty client_secret for confidential client")
	}
	if result.Client.ClientSecretHash == "" {
		t.Error("expected non-empty secret hash")
	}
	if result.Client.Type != domain.ClientTypeConfidential {
		t.Errorf("expected confidential, got %s", result.Client.Type)
	}

	// Verify the secret can be verified.
	ok, _ := ggidcrypto.VerifyPassword(result.ClientSecret, result.Client.ClientSecretHash)
	if !ok {
		t.Error("client secret verification failed")
	}
}

func TestCreateClient_Public(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      testTenantID,
		Name:          "SPA App",
		Type:          domain.ClientTypePublic,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://spa.example.com/callback"},
	})
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}

	if result.ClientSecret != "" {
		t.Error("public client should not have a secret")
	}
	if result.Client.ClientSecretHash != "" {
		t.Error("public client should not have a secret hash")
	}
}

func TestCreateClient_DefaultAuthMethod(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID: testTenantID,
		Name:     "App",
		Type:     domain.ClientTypeConfidential,
	})

	if result.Client.TokenEndpointAuthMethod != "client_secret_basic" {
		t.Errorf("expected default auth method 'client_secret_basic', got '%s'", result.Client.TokenEndpointAuthMethod)
	}
}

func TestClientSupportsGrantType(t *testing.T) {
	client := &domain.OAuthClient{
		GrantTypes: []string{"authorization_code", "refresh_token"},
	}

	if !client.SupportsGrantType("authorization_code") {
		t.Error("expected to support authorization_code")
	}
	if client.SupportsGrantType("password") {
		t.Error("should not support password grant")
	}
}

func TestClientValidateRedirectURI(t *testing.T) {
	client := &domain.OAuthClient{
		RedirectURIs: []string{"https://app.example.com/callback", "http://localhost:3000/callback"},
	}

	if !client.ValidateRedirectURI("https://app.example.com/callback") {
		t.Error("expected registered URI to be valid")
	}
	if client.ValidateRedirectURI("https://evil.example.com/callback") {
		t.Error("unregistered URI should be invalid")
	}
}

// --- Authorization Code Flow ---

func TestCreateAuthorizationCode_Success(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	// Pre-create a client.
	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:            uuid.New(),
		TenantID:      testTenantID,
		ClientID:      "test_client",
		Type:          domain.ClientTypeConfidential,
		RedirectURIs:  []string{"https://app.example.com/callback"},
		ResponseTypes: []string{"code"},
		Enabled:       true,
	}

	code, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:            testTenantID,
		ClientID:            "test_client",
		RedirectURI:         "https://app.example.com/callback",
		ResponseType:        "code",
		Scope:               []string{"openid", "profile"},
		State:               "xyz123",
		Nonce:               "nonce-abc",
		CodeChallenge:       "e916xcaQ",
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode failed: %v", err)
	}

	if code == "" {
		t.Error("expected non-empty code")
	}
}

func TestCreateAuthorizationCode_InvalidRedirectURI(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:           uuid.New(),
		TenantID:     testTenantID,
		ClientID:     "test_client",
		RedirectURIs: []string{"https://app.example.com/callback"},
		Enabled:      true,
	}

	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "test_client",
		RedirectURI: "https://evil.example.com/callback",
	})
	if err == nil {
		t.Fatal("expected error for invalid redirect_uri")
	}
}

func TestCreateAuthorizationCode_ClientNotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "nonexistent",
		RedirectURI: "https://app.example.com/callback",
	})
	if err == nil {
		t.Fatal("expected error for non-existent client")
	}
}

func TestPKCE_Validate_S256(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// Known S256 challenge for this verifier.
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	code := &domain.AuthorizationCode{
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	}

	if !code.ValidatePKCE(verifier) {
		t.Error("PKCE S256 validation should succeed with correct verifier")
	}
	if code.ValidatePKCE("wrong-verifier") {
		t.Error("PKCE S256 validation should fail with wrong verifier")
	}
}

func TestPKCE_Validate_Plain(t *testing.T) {
	code := &domain.AuthorizationCode{
		CodeChallenge:       "my-challenge",
		CodeChallengeMethod: "plain",
	}

	if !code.ValidatePKCE("my-challenge") {
		t.Error("PKCE plain validation should succeed")
	}
	if code.ValidatePKCE("wrong") {
		t.Error("PKCE plain validation should fail with wrong verifier")
	}
}

func TestPKCE_NotRequired(t *testing.T) {
	code := &domain.AuthorizationCode{
		CodeChallenge: "", // no PKCE
	}

	// Any verifier should pass when PKCE is not required.
	if !code.ValidatePKCE("anything") {
		t.Error("PKCE should be optional")
	}
	if !code.ValidatePKCE("") {
		t.Error("PKCE should pass with empty verifier when not required")
	}
}

// --- Token Exchange ---

func TestExchangeAuthorizationCode_Success(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	userID := uuid.New()
	clientDBID := uuid.New()

	// Pre-create a confidential client with a known secret.
	secretHash, _ := ggidcrypto.HashPassword("test-secret")
	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:               clientDBID,
		TenantID:         testTenantID,
		ClientID:         "test_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/callback"},
		Enabled:          true,
	}

	// Create an authorization code.
	plainCode, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "test_client",
		RedirectURI: "https://app.example.com/callback",
		ResponseType: "code",
		State:       "test-state",
		Scope:       []string{"openid", "profile"},
		Nonce:       "test-nonce",
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("CreateAuthorizationCode failed: %v", err)
	}

	// Exchange the code.
	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://app.example.com/callback",
		ClientID:     "test_client",
		ClientSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode failed: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got '%s'", resp.TokenType)
	}
	if resp.ExpiresIn <= 0 {
		t.Error("expected positive expires_in")
	}
	if resp.IDToken == "" {
		t.Error("expected non-empty id_token (openid scope present)")
	}
}

func TestExchangeAuthorizationCode_WrongClientSecret(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := ggidcrypto.HashPassword("correct-secret")
	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "test_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/callback"},
		Enabled:          true,
	}

	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         "fake-code",
		ClientID:     "test_client",
		ClientSecret: "wrong-secret",
		RedirectURI:  "https://app.example.com/callback",
	})
	if err == nil {
		t.Fatal("expected error for wrong client secret")
	}

	ge, ok := errors.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T", err)
	}
	if ge.Code != errors.ErrUnauthenticated {
		t.Errorf("expected ErrUnauthenticated, got %s", ge.Code)
	}
}

func TestExchangeAuthorizationCode_CodeReplayPrevented(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := ggidcrypto.HashPassword("secret")
	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "test_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/callback"},
		Enabled:          true,
	}

	plainCode, _ := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "test_client",
		RedirectURI: "https://app.example.com/callback",
		ResponseType: "code",
		State:       "test-state",
		Scope:       []string{"openid"},
		UserID:      uuid.New(),
	})

	// First exchange succeeds.
	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://app.example.com/callback",
		ClientID:     "test_client",
		ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("first exchange failed: %v", err)
	}

	// Replay should fail.
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://app.example.com/callback",
		ClientID:     "test_client",
		ClientSecret: "secret",
	})
	if err == nil {
		t.Fatal("expected error for code replay")
	}
}

func TestExchangeAuthorizationCode_RedirectURIMismatch(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	secretHash, _ := ggidcrypto.HashPassword("secret")
	clientRepo.clients["test_client"] = &domain.OAuthClient{
		ID:               uuid.New(),
		TenantID:         testTenantID,
		ClientID:         "test_client",
		ClientSecretHash: secretHash,
		Type:             domain.ClientTypeConfidential,
		RedirectURIs:     []string{"https://app.example.com/callback"},
		Enabled:          true,
	}

	plainCode, _ := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{
		TenantID:    testTenantID,
		ClientID:    "test_client",
		RedirectURI: "https://app.example.com/callback",
		ResponseType: "code",
		State:       "test-state",
		UserID:      uuid.New(),
	})

	_, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plainCode,
		RedirectURI:  "https://different.example.com/callback", // mismatch!
		ClientID:     "test_client",
		ClientSecret: "secret",
	})
	if err == nil {
		t.Fatal("expected error for redirect_uri mismatch")
	}
}

// --- OIDC Discovery ---

func TestGetDiscoveryConfig(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	config := svc.GetDiscoveryConfig()

	if config.Issuer != "https://test.ggid.dev" {
		t.Errorf("expected issuer 'https://test.ggid.dev', got '%s'", config.Issuer)
	}
	if config.AuthorizationEndpoint == "" {
		t.Error("expected non-empty authorization_endpoint")
	}
	if config.JwksURI == "" {
		t.Error("expected non-empty jwks_uri")
	}

	// Verify required OIDC fields.
	found := false
	for _, alg := range config.IDTokenSigningAlgValues {
		if alg == "RS256" {
			found = true
		}
	}
	if !found {
		t.Error("RS256 should be in id_token_signing_alg_values_supported")
	}
}

// --- JWKS ---

func TestGetJWKS(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()

	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}

	key := jwks.Keys[0]
	if key.KTY != "RSA" {
		t.Errorf("expected kty RSA, got %s", key.KTY)
	}
	if key.Use != "sig" {
		t.Errorf("expected use sig, got %s", key.Use)
	}
	if key.Alg != "RS256" {
		t.Errorf("expected alg RS256, got %s", key.Alg)
	}

	// Verify N is valid base64.
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		t.Fatalf("failed to decode N: %v", err)
	}
	if len(nBytes) == 0 {
		t.Error("expected non-empty modulus")
	}
}

// --- Utility ---

func TestGenerateClientID_Format(t *testing.T) {
	id := generateClientID()
	if len(id) < 15 || id[:5] != "gcid_" {
		t.Errorf("expected ID starting with 'gcid_', got '%s'", id)
	}

	// Two calls should produce different IDs.
	id2 := generateClientID()
	if id == id2 {
		t.Error("expected different client IDs on successive calls")
	}
}

func TestGenerateClientSecret_Format(t *testing.T) {
	secret := generateClientSecret()
	if len(secret) < 15 || secret[:4] != "gcs_" {
		t.Errorf("expected secret starting with 'gcs_', got '%s'", secret)
	}
}

func TestHashCode_Deterministic(t *testing.T) {
	h1 := hashCode("test-code")
	h2 := hashCode("test-code")
	if h1 != h2 {
		t.Error("hashCode should be deterministic for same input")
	}

	h3 := hashCode("different-code")
	if h1 == h3 {
		t.Error("hashCode should differ for different inputs")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"openid", "profile", "email"}

	if !contains(slice, "openid") {
		t.Error("expected to find 'openid'")
	}
	if contains(slice, "offline") {
		t.Error("should not find 'offline'")
	}
	if contains(nil, "anything") {
		t.Error("nil slice should not contain anything")
	}
}

func TestJoinScopes(t *testing.T) {
	result := joinScopes([]string{"openid", "profile"})
	if result != "openid profile" {
		t.Errorf("expected 'openid profile', got '%s'", result)
	}

	result = joinScopes([]string{"openid"})
	if result != "openid" {
		t.Errorf("expected 'openid', got '%s'", result)
	}

	result = joinScopes([]string{})
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

// Suppress unused imports.
var _ = x509.MarshalPKIXPublicKey

// === Client Secret Rotation ===

func TestRotateClientSecret_Success(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	// Create a confidential client.
	result, err := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:      tenantID,
		Name:          "test-client",
		Type:          domain.ClientTypeConfidential,
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		RedirectURIs:  []string{"https://example.com/cb"},
		Scopes:        []string{"openid"},
	})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	oldSecret := result.ClientSecret

	// Rotate the secret.
	newSecret, err := svc.RotateClientSecret(context.Background(), tenantID, result.Client.ClientID, oldSecret)
	if err != nil {
		t.Fatalf("RotateClientSecret: %v", err)
	}
	if newSecret == "" {
		t.Error("expected non-empty new secret")
	}
	if newSecret == oldSecret {
		t.Error("new secret should differ from old")
	}

	// Verify old secret no longer works.
	stored := clientRepo.clients[result.Client.ClientID]
	ok, _ := ggidcrypto.VerifyPassword(oldSecret, stored.ClientSecretHash)
	if ok {
		t.Error("old secret should no longer match after rotation")
	}
	ok, _ = ggidcrypto.VerifyPassword(newSecret, stored.ClientSecretHash)
	if !ok {
		t.Error("new secret should match after rotation")
	}
}

func TestRotateClientSecret_NotFound(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	_, err := svc.RotateClientSecret(context.Background(), tenantID, "nonexistent", "secret")
	if err == nil {
		t.Error("expected error for non-existent client")
	}
}

// === ClaimRulesEngine ===

func TestClaimRulesEngine_ApplyRules(t *testing.T) {
	engine := NewClaimRulesEngine([]ClaimRule{
		{ClaimName: "department", SourceAttr: "dept", Default: "engineering"},
		{ClaimName: "cost_center", SourceAttr: "cc", Default: "1000"},
	})

	claims := jwt.MapClaims{}
	userAttrs := map[string]any{
		"dept": "sales",
		// cc is missing — should use default.
	}

	engine.ApplyRules(claims, userAttrs)

	if claims["department"] != "sales" {
		t.Errorf("expected department='sales', got %v", claims["department"])
	}
	if claims["cost_center"] != "1000" {
		t.Errorf("expected cost_center='1000', got %v", claims["cost_center"])
	}
}

func TestClaimRulesEngine_NoOverwrite(t *testing.T) {
	engine := NewClaimRulesEngine([]ClaimRule{
		{ClaimName: "sub", SourceAttr: "uid", Default: "default"},
	})

	claims := jwt.MapClaims{"sub": "existing"}
	engine.ApplyRules(claims, map[string]any{"uid": "override"})

	if claims["sub"] != "existing" {
		t.Error("should not overwrite existing claims")
	}
}

func TestClaimRulesEngine_AddRule(t *testing.T) {
	engine := NewClaimRulesEngine(nil)
	engine.AddRule(ClaimRule{ClaimName: "custom", Default: "val"})

	claims := jwt.MapClaims{}
	engine.ApplyRules(claims, nil)

	if claims["custom"] != "val" {
		t.Error("AddRule should add a working rule")
	}
}

// TestRevokeToken_Empty moved to coverage_boost4_test.go to avoid duplicate.

func TestIsTokenRevoked_NotRevoked(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	if svc.IsTokenRevoked("some-random-token") {
		t.Error("token should not be revoked")
	}
}

// === Client Credentials Grant ===

func TestClientCredentials_Success(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   tenantID,
		Name:       "m2m-client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
		Scopes:     []string{"read", "write"},
	})

	resp, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
		Scope:        []string{"read"},
	})
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", resp.TokenType)
	}
}

func TestClientCredentials_WrongSecret(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   tenantID,
		Name:       "m2m-client",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
	})

	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: "wrong-secret",
	})
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestClientCredentials_UnsupportedGrant(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   tenantID,
		Name:       "auth-code-only",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"authorization_code"}, // no client_credentials
	})

	_, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
	})
	if err == nil {
		t.Error("expected error for unsupported grant type")
	}
}

// === Refresh Token Grant ===

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID:     tenantID,
		RefreshToken: "invalid",
		ClientID:     "nonexistent",
	})
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

// === SAML Token ===

func TestIssueSAMLToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	token, _, err := svc.IssueSAMLToken(tenantID, "nameid@example.com", "user@example.com", "John Doe")
	if err != nil {
		t.Fatalf("IssueSAMLToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty SAML token")
	}
}

// === DefaultIfEmpty ===

func TestDefaultIfEmpty(t *testing.T) {
	if defaultIfEmpty("", "default") != "default" {
		t.Error("expected 'default' for empty string")
	}
	if defaultIfEmpty("value", "default") != "value" {
		t.Error("expected 'value' for non-empty string")
	}
}

// === issueIDToken test ===

func TestIssueIDToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()
	userID := uuid.New()

	token, err := svc.issueIDToken(userID, tenantID, "test-client", "nonce123", nil)
	if err != nil {
		t.Fatalf("issueIDToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty ID token")
	}
}
