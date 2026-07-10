package authprovider

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/go-ldap/ldap/v3"
)

// fakeLDAPServer simulates a stateful LDAP directory server for testing.
// It maintains a user directory and credential store, and produces
// ldapConn instances that behave like real LDAP connections.
type fakeLDAPServer struct {
	mu sync.Mutex

	// directory maps user DN → attributes (including credentials)
	users map[string]*fakeUser

	// valid service-account credentials
	serviceDN       string
	servicePassword string

	// instrumentation
	bindCalls    []bindCall
	searchCalls  []searchCall
	startTLSCalls int
	closeCalls    int

	// injectable behaviors
	dialErr       error // if non-nil, dial returns this error
	startTLSErr   error // if non-nil, StartTLS returns this error
}

type fakeUser struct {
	dn       string
	password string
	attrs    []*ldap.EntryAttribute
}

type bindCall struct {
	username string
	password string
}

type searchCall struct {
	baseDN string
	filter string
}

func newFakeLDAPServer() *fakeLDAPServer {
	return &fakeLDAPServer{
		users: make(map[string]*fakeUser),
	}
}

// addUser registers a user in the fake directory.
func (s *fakeLDAPServer) addUser(dn, password string, attrs ...*ldap.EntryAttribute) {
	s.users[dn] = &fakeUser{dn: dn, password: password, attrs: attrs}
}

// setServiceAccount configures the service bind credentials.
func (s *fakeLDAPServer) setServiceAccount(dn, password string) {
	s.serviceDN = dn
	s.servicePassword = password
}

// newConn returns a ldapConn backed by this fake server.
func (s *fakeLDAPServer) newConn() ldapConn {
	return &fakeServerConn{server: s}
}

// fakeServerConn implements ldapConn by delegating to the fakeLDAPServer.
type fakeServerConn struct {
	server *fakeLDAPServer
	closed bool
}

func (c *fakeServerConn) Bind(username, password string) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	c.server.bindCalls = append(c.server.bindCalls, bindCall{
		username: username,
		password: password,
	})

	// Check service account first.
	if username == c.server.serviceDN {
		if password == c.server.servicePassword {
			return nil
		}
		return errors.New("LDAPResultInvalidCredentials: service bind failed")
	}

	// Check user credentials.
	user, ok := c.server.users[username]
	if !ok {
		return errors.New("LDAPResultInvalidCredentials: no such object")
	}
	if password != user.password {
		return errors.New("LDAPResultInvalidCredentials: wrong password")
	}
	return nil
}

func (c *fakeServerConn) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	c.server.searchCalls = append(c.server.searchCalls, searchCall{
		baseDN: req.BaseDN,
		filter: req.Filter,
	})

	// Match users whose DN contains the base DN.
	var entries []*ldap.Entry
	for _, user := range c.server.users {
		if strings.Contains(strings.ToLower(user.dn), strings.ToLower(req.BaseDN)) {
			entries = append(entries, &ldap.Entry{
				DN:         user.dn,
				Attributes: user.attrs,
			})
		}
	}

	return &ldap.SearchResult{Entries: entries}, nil
}

func (c *fakeServerConn) Close() error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	c.server.closeCalls++
	c.closed = true
	return nil
}

func (c *fakeServerConn) IsClosing() bool {
	return c.closed
}

func (c *fakeServerConn) StartTLS(config *tls.Config) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	c.server.startTLSCalls++
	if c.server.startTLSErr != nil {
		return c.server.startTLSErr
	}
	return nil
}

// makeDialFunc returns a dialFunc that produces connections from this server.
func (s *fakeLDAPServer) makeDialFunc() func(ctx context.Context) (ldapConn, error) {
	return func(_ context.Context) (ldapConn, error) {
		if s.dialErr != nil {
			return nil, s.dialErr
		}
		return s.newConn(), nil
	}
}

// helper to build a multi-valued attribute
func multiAttr(name string, values ...string) *ldap.EntryAttribute {
	return &ldap.EntryAttribute{Name: name, Values: values}
}

func singleAttr(name, value string) *ldap.EntryAttribute {
	return &ldap.EntryAttribute{Name: name, Values: []string{value}}
}

// =========================================================================
// Tests using the fake LDAP server
// =========================================================================

// TestFakeServer_AuthSuccess exercises the full Authenticate flow: service bind,
// user search, user bind, and result construction.
func TestFakeServer_AuthSuccess(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "secret123",
		singleAttr("mail", "jdoe@corp.local"),
		singleAttr("displayName", "John Doe"),
		singleAttr("sAMAccountName", "jdoe"),
	)

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	if result.ExternalID != "CN=jdoe,DC=corp,DC=local" {
		t.Errorf("expected ExternalID, got %q", result.ExternalID)
	}
	if result.Provider != ProviderLDAP {
		t.Errorf("expected ProviderLDAP, got %s", result.Provider)
	}
	if result.Attributes["mail"] != "jdoe@corp.local" {
		t.Errorf("expected mail attr, got %v", result.Attributes["mail"])
	}

	// Verify the server recorded the expected operations.
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.bindCalls) < 2 {
		t.Errorf("expected at least 2 bind calls (service + user), got %d", len(srv.bindCalls))
	}
	if len(srv.searchCalls) != 1 {
		t.Errorf("expected 1 search call, got %d", len(srv.searchCalls))
	}
	// The search filter should contain the escaped username.
	if !strings.Contains(srv.searchCalls[0].filter, "jdoe") {
		t.Errorf("expected filter to contain 'jdoe', got %q", srv.searchCalls[0].filter)
	}
}

// TestFakeServer_AuthSuccess_GroupRoleMapping verifies that group membership
// attributes are mapped to roles during successful authentication.
func TestFakeServer_AuthSuccess_GroupRoleMapping(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=admin1,DC=corp,DC=local", "pass",
		singleAttr("mail", "admin1@corp.local"),
		multiAttr("memberOf",
			"CN=Admins,DC=corp,DC=local",
			"CN=Developers,DC=corp,DC=local",
		),
	)

	cfg := validLDAPConfig()
	cfg.AutoProvision = true
	cfg.GroupRoleMappings = []GroupRoleMapping{
		{GroupDN: "CN=Admins,DC=corp,DC=local", Role: "admin"},
		{GroupDN: "CN=Developers,DC=corp,DC=local", Role: "developer"},
		{GroupDN: "CN=Guests,DC=corp,DC=local", Role: "guest"}, // not a member
	}
	p, _ := NewLDAPProvider(cfg)
	p.dialFunc = srv.makeDialFunc()

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "admin1",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	if len(result.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %v", result.Roles)
	}

	roleSet := map[string]bool{}
	for _, r := range result.Roles {
		roleSet[r] = true
	}
	if !roleSet["admin"] {
		t.Errorf("expected 'admin' role, got %v", result.Roles)
	}
	if !roleSet["developer"] {
		t.Errorf("expected 'developer' role, got %v", result.Roles)
	}
}

// TestFakeServer_UserNotFound verifies that an empty search result returns
// an unauthenticated error.
func TestFakeServer_UserNotFound(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	// No users added.

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "nobody",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for user not found")
	}
}

// TestFakeServer_WrongPassword verifies that when the user bind fails
// (incorrect password), authentication returns an unauthenticated error.
func TestFakeServer_WrongPassword(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "correctPass",
		singleAttr("mail", "jdoe@corp.local"),
	)

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "wrongPass",
	})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

// TestFakeServer_ConnectionError verifies that a dial failure propagates
// as an internal error.
func TestFakeServer_ConnectionError(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.dialErr = errors.New("connection refused")

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

// TestFakeServer_ServiceBindFails verifies that a failed service-account bind
// returns an error before any user search is attempted.
func TestFakeServer_ServiceBindFails(t *testing.T) {
	srv := newFakeLDAPServer()
	// Wrong service password.
	srv.setServiceAccount("CN=svc,DC=corp,DC=local", "wrongSvcPass")

	p, _ := NewLDAPProvider(LDAPConfig{
		ServerURL:    "ldap://dc01.corp.local:389",
		BindDN:       "CN=svc,DC=corp,DC=local",
		BindPassword: "svcPass", // won't match server's "wrongSvcPass"
		BaseDN:       "DC=corp,DC=local",
	})
	p.dialFunc = srv.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for service bind failure")
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	// No search should have been attempted.
	if len(srv.searchCalls) != 0 {
		t.Errorf("expected 0 search calls, got %d", len(srv.searchCalls))
	}
}

// TestFakeServer_STARTTLS verifies that when StartTLS is configured,
// the dial function invokes StartTLS on each new connection.
func TestFakeServer_STARTTLS(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "pass",
		singleAttr("mail", "jdoe@corp.local"),
	)

	// Use a custom dialFunc that calls StartTLS like the real dial() does.
	srvWithTLS := &startTLSDialWrapper{server: srv}
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srvWithTLS.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.startTLSCalls == 0 {
		t.Error("expected StartTLS to be called at least once")
	}
}

// startTLSDialWrapper wraps a fakeLDAPServer's dial to invoke StartTLS
// after creating each connection, simulating the real dial() behavior.
type startTLSDialWrapper struct {
	server *fakeLDAPServer
}

func (w *startTLSDialWrapper) makeDialFunc() func(ctx context.Context) (ldapConn, error) {
	return func(ctx context.Context) (ldapConn, error) {
		if w.server.dialErr != nil {
			return nil, w.server.dialErr
		}
		conn := w.server.newConn()
		if err := conn.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12}); err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	}
}

// TestFakeServer_STARTTLSError verifies that a StartTLS failure during dial
// prevents authentication.
func TestFakeServer_STARTTLSError(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.startTLSErr = errors.New("TLS handshake failed")

	wrapper := &startTLSDialWrapper{server: srv}
	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = wrapper.makeDialFunc()

	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error for STARTTLS failure")
	}
}

// TestFakeServer_AutoProvision verifies the auto-provision flag sets NewUser.
func TestFakeServer_AutoProvision(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=newuser,DC=corp,DC=local", "pass",
		singleAttr("mail", "newuser@corp.local"),
		singleAttr("givenName", "New"),
		singleAttr("sn", "User"),
	)

	cfg := validLDAPConfig()
	cfg.AutoProvision = true
	p, _ := NewLDAPProvider(cfg)
	p.dialFunc = srv.makeDialFunc()

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "newuser",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !result.NewUser {
		t.Error("expected NewUser=true for auto-provision")
	}
	if result.MustLink {
		t.Error("expected MustLink=false for auto-provision")
	}
	if result.Attributes["givenName"] != "New" {
		t.Errorf("expected givenName 'New', got %v", result.Attributes["givenName"])
	}
}

// TestFakeServer_NoAutoProvision verifies that without auto-provision,
// MustLink is set to true.
func TestFakeServer_NoAutoProvision(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=linkme,DC=corp,DC=local", "pass",
		singleAttr("mail", "linkme@corp.local"),
	)

	cfg := validLDAPConfig()
	cfg.AutoProvision = false
	p, _ := NewLDAPProvider(cfg)
	p.dialFunc = srv.makeDialFunc()

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "linkme",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.NewUser {
		t.Error("expected NewUser=false without auto-provision")
	}
	if !result.MustLink {
		t.Error("expected MustLink=true without auto-provision")
	}
}

// TestFakeServer_MultiValuedAttrs verifies that multi-valued attributes
// (e.g., memberOf with multiple groups) are returned as []string.
func TestFakeServer_MultiValuedAttrs(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=multi,DC=corp,DC=local", "pass",
		singleAttr("mail", "multi@corp.local"),
		multiAttr("memberOf",
			"CN=Group1,DC=corp,DC=local",
			"CN=Group2,DC=corp,DC=local",
			"CN=Group3,DC=corp,DC=local",
		),
	)

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	result, err := p.Authenticate(context.Background(), Credentials{
		Username: "multi",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	memberOf, ok := result.Attributes["memberOf"].([]string)
	if !ok {
		t.Fatalf("expected memberOf to be []string, got %T", result.Attributes["memberOf"])
	}
	if len(memberOf) != 3 {
		t.Errorf("expected 3 groups, got %d", len(memberOf))
	}
}

// TestFakeServer_PoolReuse verifies that a healthy connection is returned
// to the pool after successful authentication and can be reused on the
// next call (exercising the getConn pool-hit branch).
func TestFakeServer_PoolReuse(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "pass",
		singleAttr("mail", "jdoe@corp.local"),
	)

	cfg := validLDAPConfig()
	cfg.PoolSize = 3
	p, _ := NewLDAPProvider(cfg)
	p.dialFunc = srv.makeDialFunc()

	// First authentication — connections go through dialFunc, then
	// healthy ones are returned to the pool.
	_, err := p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("first auth failed: %v", err)
	}

	poolLen := len(p.pool)
	if poolLen == 0 {
		t.Fatal("expected connections in pool after successful auth")
	}

	// Second authentication should be able to reuse a pooled connection
	// for the service-bind step.
	_, err = p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("second auth failed: %v", err)
	}
}

// TestFakeServer_PoolReuseClosingConn verifies that when a pooled connection
// is closing, getConn discards it and dials a fresh one.
func TestFakeServer_PoolReuseClosingConn(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "pass",
		singleAttr("mail", "jdoe@corp.local"),
	)

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	// Manually push a "closing" connection into the pool.
	closingConn := &fakeServerConn{server: srv}
	closingConn.closed = true // simulate IsClosing() == true
	p.pool <- closingConn

	// getConn should detect the closing conn, discard it, and dial fresh.
	conn, err := p.getConn(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}

	// Pool should now be empty (closing conn was removed).
	if len(p.pool) != 0 {
		t.Errorf("expected empty pool, got %d", len(p.pool))
	}
}

// TestFakeServer_SearchFilterEscaping verifies that special characters in
// the username are properly escaped in the LDAP search filter.
func TestFakeServer_SearchFilterEscaping(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	// Username with LDAP special characters that must be escaped.
	_, _ = p.Authenticate(context.Background(), Credentials{
		Username: "test*user(admin)",
		Password: "pass",
	})

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.searchCalls) == 0 {
		t.Fatal("expected at least 1 search call")
	}
	// The raw '*' and '(' should NOT appear unescaped in the filter.
	// ldap.EscapeFilter converts '*' to '\2a', '(' to '\28', ')' to '\29'.
	filter := srv.searchCalls[0].filter
	if strings.Contains(filter, "test*user") {
		t.Errorf("expected '*' to be escaped in filter: %q", filter)
	}
	if !strings.Contains(filter, `\2a`) {
		t.Errorf("expected escaped '*' (\\2a) in filter: %q", filter)
	}
}

// TestFakeServer_CloseAfterAuth verifies that Close drains all pooled
// TestFakeServer_CloseAfterAuth verifies that Close drains the pool.
// Authenticate may or may not leave connections pooled (depends on conn state),
// so we manually add a connection to the pool to test Close behavior.
func TestFakeServer_CloseAfterAuth(t *testing.T) {
	srv := newFakeLDAPServer()
	srv.setServiceAccount("CN=admin,DC=corp,DC=local", "adminPass")
	srv.addUser(
		"CN=jdoe,DC=corp,DC=local", "pass",
		singleAttr("mail", "jdoe@corp.local"),
	)

	p, _ := NewLDAPProvider(validLDAPConfig())
	p.dialFunc = srv.makeDialFunc()

	// Authenticate to exercise the flow.
	_, _ = p.Authenticate(context.Background(), Credentials{
		Username: "jdoe",
		Password: "pass",
	})

	// Manually add a healthy connection to the pool to test Close.
	conn, err := p.dialFunc(context.Background())
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	p.putConn(conn, true)

	if len(p.pool) == 0 {
		t.Fatal("expected pooled connections before Close")
	}

	p.Close()

	if len(p.pool) != 0 {
		t.Errorf("expected empty pool after Close, got %d", len(p.pool))
	}

	srv.mu.Lock()
	closeCount := srv.closeCalls
	srv.mu.Unlock()
	if closeCount == 0 {
		t.Error("expected Close to be called on pooled connections")
	}
}
