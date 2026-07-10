package authprovider

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- mapGroupsToRoles ---

func TestMapGroupsToRoles_MultiGroup(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins,DC=corp,DC=local", Role: "admin"},
		{GroupDN: "CN=Users,DC=corp,DC=local", Role: "user"},
	}
	p, _ := NewLDAPProvider(cfg)

	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": []string{
			"CN=Admins,DC=corp,DC=local",
			"CN=Users,DC=corp,DC=local",
		},
	})
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d: %v", len(roles), roles)
	}
}

func TestMapGroupsToRoles_StringGroup(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins,DC=corp,DC=local", Role: "admin"},
	}
	p, _ := NewLDAPProvider(cfg)

	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": "CN=Admins,DC=corp,DC=local",
	})
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("expected [admin], got %v", roles)
	}
}

func TestMapGroupsToRoles_NoMemberOf(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	roles := p.mapGroupsToRoles(map[string]any{})
	if len(roles) != 0 {
		t.Errorf("expected 0 roles, got %v", roles)
	}
}

func TestMapGroupsToRoles_NoMatchingGroups(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins", Role: "admin"},
	}
	p, _ := NewLDAPProvider(cfg)
	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": []string{"CN=Guests"},
	})
	if len(roles) != 0 {
		t.Errorf("expected 0 roles, got %v", roles)
	}
}

func TestMapGroupsToRoles_CaseInsensitive(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "cn=admins,dc=corp,dc=local", Role: "admin"},
	}
	p, _ := NewLDAPProvider(cfg)
	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": []string{"CN=ADMINS,DC=CORP,DC=LOCAL"},
	})
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("expected case-insensitive match [admin], got %v", roles)
	}
}

func TestMapGroupsToRoles_Dedup(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins", Role: "admin"},
		{GroupDN: "CN=SuperAdmins", Role: "admin"},
	}
	p, _ := NewLDAPProvider(cfg)
	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": []string{"CN=Admins", "CN=SuperAdmins"},
	})
	if len(roles) != 1 {
		t.Errorf("expected dedup to 1, got %v", roles)
	}
}

func TestMapGroupsToRoles_InvalidType(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins", Role: "admin"},
	}
	p, _ := NewLDAPProvider(cfg)
	roles := p.mapGroupsToRoles(map[string]any{
		"memberOf": 12345,
	})
	if len(roles) != 0 {
		t.Errorf("expected 0 roles for invalid type, got %v", roles)
	}
}

// --- dial (unreachable servers → error paths) ---

func TestDial_UnreachableLDAP(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.ServerURL = "ldap://127.0.0.1:1"
	cfg.ConnTimeout = 100 * time.Millisecond
	p, _ := NewLDAPProvider(cfg)
	_, err := p.dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error")
	}
}

func TestDial_UnreachableLDAPS(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.ServerURL = "ldaps://127.0.0.1:1"
	cfg.ConnTimeout = 100 * time.Millisecond
	p, _ := NewLDAPProvider(cfg)
	_, err := p.dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error for LDAPS")
	}
}

func TestDial_StartTLSUnreachable(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.ServerURL = "ldap://127.0.0.1:1"
	cfg.StartTLS = true
	cfg.ConnTimeout = 100 * time.Millisecond
	p, _ := NewLDAPProvider(cfg)
	_, err := p.dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error with StartTLS")
	}
}

// --- pool management ---

func TestClose_DrainsPool(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	conn1 := &mockLDAPConn{}
	conn2 := &mockLDAPConn{}
	p.pool <- conn1
	p.pool <- conn2
	p.Close()
	if !conn1.closed || !conn2.closed {
		t.Error("expected pool connections to be closed after Close")
	}
	if len(p.pool) != 0 {
		t.Errorf("expected empty pool, got %d", len(p.pool))
	}
}

func TestGetConn_UsesDialFunc(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = func(_ context.Context) (ldapConn, error) {
		return &mockLDAPConn{}, nil
	}
	conn, err := p.getConn(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn")
	}
}

func TestPutConn_HealthyReturnsToPool(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	conn := &mockLDAPConn{}
	p.putConn(conn, true)
	if len(p.pool) != 1 {
		t.Errorf("expected 1 in pool, got %d", len(p.pool))
	}
}

func TestPutConn_UnhealthyClosed(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	conn := &mockLDAPConn{}
	p.putConn(conn, false)
	if len(p.pool) != 0 {
		t.Errorf("expected 0 in pool for unhealthy conn, got %d", len(p.pool))
	}
	if !conn.closed {
		t.Error("expected unhealthy conn to be closed")
	}
}

func TestPutConn_ClosingConnClosed(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	conn := &mockLDAPConn{closing: true}
	p.putConn(conn, true) // healthy=true but closing
	if len(p.pool) != 0 {
		t.Errorf("expected 0 in pool for closing conn, got %d", len(p.pool))
	}
}

func TestPutConn_PoolFull(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.PoolSize = 1
	p, _ := NewLDAPProvider(cfg)
	p.pool <- &mockLDAPConn{} // fill pool
	extra := &mockLDAPConn{}
	p.putConn(extra, true) // pool full, should be closed
	if len(p.pool) != 1 {
		t.Errorf("expected pool at capacity 1, got %d", len(p.pool))
	}
}

// --- tlsConfig ---

func TestTLSConfig_Default(t *testing.T) {
	p, _ := NewLDAPProvider(validLDAPConfig())
	tc := p.tlsConfig()
	if tc.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 min, got %x", tc.MinVersion)
	}
}

func TestTLSConfig_Custom(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}
	p, _ := NewLDAPProvider(cfg)
	tc := p.tlsConfig()
	if tc.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3, got %x", tc.MinVersion)
	}
}

// --- Chain ---

func TestChain_EmptyProviders(t *testing.T) {
	chain := NewChain()
	_, err := chain.Authenticate(context.Background(), Credentials{
		Username: "x", Password: "y",
	})
	if err == nil {
		t.Fatal("expected error with no providers")
	}
	if !strings.Contains(err.Error(), "no auth providers") {
		t.Errorf("expected 'no auth providers' in error, got %v", err)
	}
}

// --- WithTenantContext ---

func TestWithTenantContext_RoundTrip(t *testing.T) {
	tid := uuid.New()
	ctx := WithTenantContext(context.Background(), tid)
	got, err := resolveTenantID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tid {
		t.Errorf("expected %s, got %s", tid, got)
	}
}
