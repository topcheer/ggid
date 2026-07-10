package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type errClientRepo3 struct{ err error }

func (m *errClientRepo3) CreateClient(context.Context, *domain.OAuthClient) error {
	return m.err
}
func (m *errClientRepo3) GetClientByID(context.Context, uuid.UUID, string) (*domain.OAuthClient, error) {
	return nil, m.err
}
func (m *errClientRepo3) ListClients(context.Context, uuid.UUID, int, int) ([]*domain.OAuthClient, int, error) {
	return nil, 0, m.err
}
func (m *errClientRepo3) UpdateClient(context.Context, uuid.UUID, string, *domain.OAuthClient) (*domain.OAuthClient, error) {
	return nil, m.err
}
func (m *errClientRepo3) DeleteClient(context.Context, uuid.UUID, string) error { return m.err }

type errCodeRepo3 struct{ err error }

func (m *errCodeRepo3) CreateCode(context.Context, *domain.AuthorizationCode) error { return m.err }
func (m *errCodeRepo3) ConsumeCode(context.Context, string) (*domain.AuthorizationCode, error) {
	return nil, m.err
}

type halfWorkingClientRepo3 struct {
	clients   map[string]*domain.OAuthClient
	updateErr error
}

func (m *halfWorkingClientRepo3) CreateClient(_ context.Context, c *domain.OAuthClient) error {
	m.clients[c.ClientID] = c
	return nil
}
func (m *halfWorkingClientRepo3) GetClientByID(_ context.Context, _ uuid.UUID, clientID string) (*domain.OAuthClient, error) {
	c, ok := m.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}
func (m *halfWorkingClientRepo3) ListClients(context.Context, uuid.UUID, int, int) ([]*domain.OAuthClient, int, error) {
	return nil, 0, nil
}
func (m *halfWorkingClientRepo3) UpdateClient(context.Context, uuid.UUID, string, *domain.OAuthClient) (*domain.OAuthClient, error) {
	return nil, m.updateErr
}
func (m *halfWorkingClientRepo3) DeleteClient(context.Context, uuid.UUID, string) error { return nil }

func TestCreateClient_RepoError3(t *testing.T) {
	svc := NewOAuthService(&errClientRepo3{err: fmt.Errorf("db down")}, newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")
	_, err := svc.CreateClient(context.Background(), &CreateClientInput{TenantID: testTenantID, Name: "t", Type: domain.ClientTypePublic})
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestCreateAuthorizationCode_CodeRepoError3(t *testing.T) {
	cr := newMockClientRepo()
	cr.clients["cr3"] = &domain.OAuthClient{ID: uuid.New(), TenantID: testTenantID, ClientID: "cr3", RedirectURIs: []string{"https://app.example.com/cb"}, Enabled: true}
	svc := NewOAuthService(cr, &errCodeRepo3{err: fmt.Errorf("db down")}, &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")
	_, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{TenantID: testTenantID, ClientID: "cr3", RedirectURI: "https://app.example.com/cb", ResponseType: "code"})
	if err == nil {
		t.Fatal("expected code repo error")
	}
}

func TestRotateClientSecret_UpdateError3(t *testing.T) {
	h, _ := crypto.HashPassword("old")
	svc := NewOAuthService(&halfWorkingClientRepo3{clients: map[string]*domain.OAuthClient{"re3": {ID: uuid.New(), TenantID: testTenantID, ClientID: "re3", ClientSecretHash: h, Type: domain.ClientTypeConfidential, Enabled: true}}, updateErr: fmt.Errorf("up")}, newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")
	_, err := svc.RotateClientSecret(context.Background(), testTenantID, "re3", "old")
	if err == nil {
		t.Fatal("expected update error")
	}
}

func TestDynamicClientRegister_RepoError3(t *testing.T) {
	svc := NewOAuthService(&errClientRepo3{err: fmt.Errorf("db down")}, newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")
	ctx := tenant.WithContext(context.Background(), &tenant.Context{TenantID: testTenantID, IsolationLevel: tenant.IsolationShared})
	_, err := svc.DynamicClientRegister(ctx, &DynamicRegistrationRequest{ClientName: "D", RedirectURIs: []string{"https://app.example.com/cb"}})
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestParseAccessToken_WrongSigningMethod3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "u", "exp": time.Now().Add(time.Hour).Unix()})
	s, _ := tok.SignedString([]byte("x"))
	_, err := svc.ParseAccessToken(s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExchangeToken_ProperlySignedNoSub3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	kp := svc.keyProvider
	now := time.Now()
	c := jwt.MapClaims{"iss": "https://test.ggid.dev", "aud": "tc", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	tok.Header["kid"] = kp.KeyID()
	s, _ := tok.SignedString(kp.PrivateKey())
	_, err := svc.ExchangeToken(context.Background(), &TokenExchangeRequestRFC8693{TenantID: testTenantID, SubjectToken: s, SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token"})
	if err == nil {
		t.Fatal("expected error for missing sub")
	}
	if !strings.Contains(err.Error(), "sub") {
		t.Errorf("got: %s", err.Error())
	}
}

func TestPollDeviceToken_Denied3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{TenantID: testTenantID, ClientID: "dc", Scope: []string{"openid"}, Issuer: "https://test.ggid.dev"})
	deviceCodeMu.Lock()
	deviceCodeStore[r.DeviceCode].Status = "denied"
	deviceCodeMu.Unlock()
	_, err := svc.PollDeviceToken(context.Background(), r.DeviceCode, "dc")
	if err == nil {
		t.Fatal("expected access_denied")
	}
}

func TestPollDeviceToken_UnknownStatus3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{TenantID: testTenantID, ClientID: "dc"})
	deviceCodeMu.Lock()
	deviceCodeStore[r.DeviceCode].Status = "weird"
	deviceCodeMu.Unlock()
	_, err := svc.PollDeviceToken(context.Background(), r.DeviceCode, "dc")
	if err == nil {
		t.Fatal("expected pending")
	}
}

func TestApproveDeviceCode_DeviceCodeMissing3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	deviceCodeMu.Lock()
	userCodeIndex["TM3"] = "no-such-code"
	delete(deviceCodeStore, "no-such-code")
	deviceCodeMu.Unlock()
	err := svc.ApproveDeviceCode("TM3", uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetInt64Claim_JSONNumber3(t *testing.T) {
	c := jwt.MapClaims{"n": json.Number("12345")}
	if v := getInt64Claim(c, "n"); v != 12345 {
		t.Errorf("got %d", v)
	}
	c2 := jwt.MapClaims{"n": json.Number("bad")}
	if v := getInt64Claim(c2, "n"); v != 0 {
		t.Errorf("got %d", v)
	}
}

func TestIntrospectToken_InvalidToken3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	r := svc.IntrospectToken("bad")
	if r.Active {
		t.Error("expected false")
	}
}

func TestIntrospectToken_ScopeNonString3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	kp := svc.keyProvider
	now := time.Now()
	c := jwt.MapClaims{"iss": "https://test.ggid.dev", "aud": "tc", "sub": "u", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(), "scope": 12345}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	tok.Header["kid"] = kp.KeyID()
	s, _ := tok.SignedString(kp.PrivateKey())
	r := svc.IntrospectToken(s)
	if !r.Active {
		t.Error("expected active")
	}
	if r.Scope != "" {
		t.Errorf("expected empty scope, got %s", r.Scope)
	}
}

func TestIntrospectToken_StringScope3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	kp := svc.keyProvider
	now := time.Now()
	c := jwt.MapClaims{"iss": "https://test.ggid.dev", "aud": "tc", "sub": "u", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(), "scope": "openid profile"}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	tok.Header["kid"] = kp.KeyID()
	s, _ := tok.SignedString(kp.PrivateKey())
	r := svc.IntrospectToken(s)
	if !r.Active {
		t.Error("expected active")
	}
	if r.Scope != "openid profile" {
		t.Errorf("got %s", r.Scope)
	}
	if r.ClientID != "tc" {
		t.Errorf("got %s", r.ClientID)
	}
}

func TestGetJWKS_KeyIDRotation3(t *testing.T) {
	kp1 := newMockKeyProvider()
	s1 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp1, "https://test.ggid.dev")
	if s1.GetJWKS().Keys[0].KID != "test-kid" {
		t.Error("expected test-kid")
	}
	kp2 := &mockKeyProvider{priv: kp1.priv, pub: kp1.pub, kid: "rotated"}
	s2 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp2, "https://test.ggid.dev")
	if s2.GetJWKS().Keys[0].KID != "rotated" {
		t.Error("expected rotated kid")
	}
}

func TestGetDiscoveryConfig_AllFields3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	cfg := svc.GetDiscoveryConfig()
	for _, f := range []string{cfg.AuthorizationEndpoint, cfg.TokenEndpoint, cfg.UserInfoEndpoint, cfg.JwksURI, cfg.RevocationEndpoint, cfg.IntrospectionEndpoint} {
		if f == "" || !strings.HasPrefix(f, "https://test.ggid.dev") {
			t.Errorf("bad endpoint: %s", f)
		}
	}
	if len(cfg.CodeChallengeMethodsSupported) < 2 {
		t.Error("need >= 2 challenge methods")
	}
}

func TestExchangeAuthorizationCode_PKCEVerifierMismatch3(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	cr := newMockClientRepo()
	svc.clientRepo = cr
	h, _ := crypto.HashPassword("secret")
	cr.clients["pkce3"] = &domain.OAuthClient{ID: uuid.New(), TenantID: testTenantID, ClientID: "pkce3", ClientSecretHash: h, Type: domain.ClientTypeConfidential, RedirectURIs: []string{"https://app.example.com/cb"}, Enabled: true}
	v := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h2 := sha256.Sum256([]byte(v))
	ch := base64.RawURLEncoding.EncodeToString(h2[:])
	pc, err := svc.CreateAuthorizationCode(context.Background(), &AuthorizeRequest{TenantID: testTenantID, ClientID: "pkce3", RedirectURI: "https://app.example.com/cb", ResponseType: "code", Scope: []string{"openid"}, CodeChallenge: ch, CodeChallengeMethod: "S256", UserID: uuid.New()})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{TenantID: testTenantID, GrantType: "authorization_code", Code: pc, RedirectURI: "https://app.example.com/cb", ClientID: "pkce3", ClientSecret: "secret", CodeVerifier: "wrong"})
	if err == nil {
		t.Fatal("expected PKCE failure")
	}
}
