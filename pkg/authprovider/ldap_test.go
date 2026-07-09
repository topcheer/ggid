package authprovider

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// --- Mock ldapConn ---

type mockLDAPConn struct {
	bindFunc      func(username, password string) error
	searchResult  *ldap.SearchResult
	searchErr     error
	closing       bool
	closed        bool
	startTLSFunc  func(config *tls.Config) error
}

func (m *mockLDAPConn) Bind(username, password string) error {
	if m.bindFunc != nil {
		return m.bindFunc(username, password)
	}
	return nil
}

func (m *mockLDAPConn) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return &ldap.SearchResult{}, nil
}

func (m *mockLDAPConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockLDAPConn) IsClosing() bool {
	return m.closing
}

func (m *mockLDAPConn) StartTLS(config *tls.Config) error {
	if m.startTLSFunc != nil {
		return m.startTLSFunc(config)
	}
	return nil
}

// --- Helpers ---

type kv struct{ Key, Val string }

func newMockSearchResult(dn string, attrs ...kv) *ldap.SearchResult {
	entry := &ldap.Entry{DN: dn}
	for _, a := range attrs {
		entry.Attributes = append(entry.Attributes, &ldap.EntryAttribute{
			Name:   a.Key,
			Values: []string{a.Val},
		})
	}
	return &ldap.SearchResult{Entries: []*ldap.Entry{entry}}
}

func validLDAPConfig() LDAPConfig {
	return LDAPConfig{
		ServerURL:    "ldap://dc01.corp.local:389",
		BindDN:       "CN=admin,DC=corp,DC=local",
		BindPassword: "adminPass",
		BaseDN:       "DC=corp,DC=local",
	}
}

// --- NewLDAPProvider tests ---

func TestNewLDAPProvider_ValidConfig(t *testing.T) {
	p, err := NewLDAPProvider(validLDAPConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type() != ProviderLDAP {
		t.Errorf("expected type %s, got %s", ProviderLDAP, p.Type())
	}
	if p.Name() != "ldap" {
		t.Errorf("expected name 'ldap', got '%s'", p.Name())
	}
}

func TestNewLDAPProvider_MissingServerURL(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.ServerURL = ""
	_, err := NewLDAPProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing server URL")
	}
}

func TestNewLDAPProvider_MissingBaseDN(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.BaseDN = ""
	_, err := NewLDAPProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing base DN")
	}
}

func TestNewLDAPProvider_Defaults(t *testing.T) {
	cfg := validLDAPConfig()
	// Leave optional fields empty.
	p, err := NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.cfg.UserFilter == "" {
		t.Error("expected default UserFilter to be set")
	}
	if p.cfg.PoolSize != 5 {
		t.Errorf("expected default PoolSize 5, got %d", p.cfg.PoolSize)
	}
	if p.cfg.ConnTimeout != 10*time.Second {
		t.Errorf("expected default ConnTimeout 10s, got %v", p.cfg.ConnTimeout)
	}
}

func TestNewLDAPProvider_LDAPSURL(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.ServerURL = "ldaps://dc01.corp.local:636"
	p, err := NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.cfg.ServerURL != "ldaps://dc01.corp.local:636" {
		t.Errorf("unexpected server URL: %s", p.cfg.ServerURL)
	}
}

// --- Authenticate tests with mocked connections ---

func TestLDAPProvider_Authenticate_EmptyCredentials(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())

	tests := []struct {
		name   string
		creds  Credentials
	}{
		{"empty username", Credentials{Password: "pw"}},
		{"empty password", Credentials{Username: "user"}},
		{"both empty", Credentials{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Authenticate(context.Background(), tt.creds)
			if err == nil {
				t.Fatal("expected error for empty credentials")
			}
		})
	}
}

func TestLDAPProvider_Authenticate_ServiceBindFails(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		return &mockLDAPConn{
			bindFunc: func(username, password string) error {
				return errors.New("invalid credentials")
			},
		}, nil
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "userpass",
	})
	if err == nil {
		t.Fatal("expected error when service bind fails")
	}
}

func TestLDAPProvider_Authenticate_UserNotFound(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())

	bindCount := 0
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		bindCount++
		return &mockLDAPConn{
			bindFunc: func(username, password string) error {
				if bindCount == 1 {
					return nil // service bind succeeds
				}
				return errors.New("should not reach user bind")
			},
			searchResult: &ldap.SearchResult{Entries: nil}, // no entries
		}, nil
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "ghost",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for user not found")
	}
}

func TestLDAPProvider_Authenticate_SearchError(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		return &mockLDAPConn{
			searchErr: errors.New("LDAP search timeout"),
		}, nil
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for search failure")
	}
}

func TestLDAPProvider_Authenticate_UserBindFails(t *testing.T) {
	cfg := validLDAPConfig()
	p, _ := NewLDAPProvider(cfg)

	callCount := 0
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		callCount++
		if callCount == 1 {
			// Service bind + search connection.
			return &mockLDAPConn{
				searchResult: newMockSearchResult("CN=testuser,DC=corp,DC=local",
					kv{"mail", "testuser@corp.local"},
					kv{"displayName", "Test User"},
				),
			}, nil
		}
		// User bind connection — fail.
		return &mockLDAPConn{
			bindFunc: func(username, password string) error {
				return errors.New("invalid credentials")
			},
		}, nil
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "wrongpass",
	})
	if err == nil {
		t.Fatal("expected error for user bind failure")
	}
}

func TestLDAPProvider_Authenticate_Success_AutoProvision(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.AutoProvision = true
	p, _ := NewLDAPProvider(cfg)

	callCount := 0
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		callCount++
		if callCount == 1 {
			return &mockLDAPConn{
				searchResult: newMockSearchResult("CN=jdoe,DC=corp,DC=local",
					kv{"mail", "jdoe@corp.local"},
					kv{"displayName", "John Doe"},
					kv{"sAMAccountName", "jdoe"},
				),
			}, nil
		}
		return &mockLDAPConn{}, nil // user bind succeeds (default no-op)
	}

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "correctpass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	if result.ExternalID != "CN=jdoe,DC=corp,DC=local" {
		t.Errorf("expected ExternalID 'CN=jdoe,DC=corp,DC=local', got '%s'", result.ExternalID)
	}
	if result.Provider != ProviderLDAP {
		t.Errorf("expected provider LDAP, got %s", result.Provider)
	}
	if !result.NewUser {
		t.Error("expected NewUser=true with AutoProvision")
	}
	if result.MustLink {
		t.Error("expected MustLink=false with AutoProvision")
	}
	// Verify attributes were extracted.
	if result.Attributes["mail"] != "jdoe@corp.local" {
		t.Errorf("expected mail attribute, got %v", result.Attributes["mail"])
	}
	if result.Attributes["displayName"] != "John Doe" {
		t.Errorf("expected displayName attribute, got %v", result.Attributes["displayName"])
	}
}

func TestLDAPProvider_Authenticate_Success_MustLink(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.AutoProvision = false // no JIT
	p, _ := NewLDAPProvider(cfg)

	callCount := 0
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		callCount++
		if callCount == 1 {
			return &mockLDAPConn{
				searchResult: newMockSearchResult("CN=jdoe,DC=corp,DC=local",
					kv{"mail", "jdoe@corp.local"},
				),
			}, nil
		}
		return &mockLDAPConn{}, nil
	}

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "correctpass",
	})
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if result.NewUser {
		t.Error("expected NewUser=false without AutoProvision")
	}
	if !result.MustLink {
		t.Error("expected MustLink=true without AutoProvision")
	}
}

func TestLDAPProvider_Authenticate_DialFails(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		return nil, errors.New("connection refused")
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error when dial fails")
	}
}

func TestLDAPProvider_Authenticate_DialFailsForUserBind(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	callCount := 0
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		callCount++
		if callCount == 1 {
			return &mockLDAPConn{
				searchResult: newMockSearchResult("CN=testuser,DC=corp,DC=local"),
			}, nil
		}
		return nil, errors.New("connection refused for user bind")
	}

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error when user bind dial fails")
	}
}

func TestLDAPProvider_Close(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())

	// Close should not panic on empty pool.
	p.Close()

	// Close should be idempotent.
	p.Close()
}

func TestLDAPProvider_tlsConfig_Default(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	tc := p.tlsConfig()
	if tc.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 min, got %x", tc.MinVersion)
	}
}

func TestLDAPProvider_tlsConfig_Custom(t *testing.T) {
	cfg := validLDAPConfig()
	customTLS := &tls.Config{MinVersion: tls.VersionTLS13}
	cfg.TLSConfig = customTLS

	p, _ := NewLDAPProvider(cfg)
	tc := p.tlsConfig()
	if tc.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 min from custom config, got %x", tc.MinVersion)
	}
}
