package authprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
)

// LDAPConfig holds the configuration for an LDAP/Active Directory backend.
type LDAPConfig struct {
	ServerURL    string        // e.g. ldap://dc01.corp.local:389 or ldaps://...:636
	BindDN       string        // service account DN for initial search
	BindPassword string        // service account password
	BaseDN       string        // search base, e.g. dc=corp,dc=local
	UserFilter   string        // filter template with %s placeholder, e.g. (&(objectClass=user)(sAMAccountName=%s))
	GroupFilter  string        // optional group filter for role sync
	GroupBaseDN  string        // optional group search base
	AutoProvision bool         // JIT: create local user on first login
	StartTLS     bool          // issue StartTLS after connecting
	TLSConfig    *tls.Config   // optional TLS config; nil means use defaults
	PoolSize     int           // max pooled connections (default 5)
	ConnTimeout  time.Duration // dial + bind timeout
	GroupRoleMappings []GroupRoleMapping // optional: map LDAP groups to roles
}

// GroupRoleMapping maps an LDAP group DN to an application role.
type GroupRoleMapping struct {
	GroupDN string // e.g. "cn=admins,dc=corp,dc=local"
	Role    string // e.g. "admin"
}

// ldapConn is the subset of *ldap.Conn methods used by LDAPProvider.
// This interface enables unit testing without a real LDAP server.
type ldapConn interface {
	Bind(username, password string) error
	Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
	Close() error
	IsClosing() bool
	StartTLS(config *tls.Config) error
}

// LDAPProvider authenticates users against an LDAP or Active Directory server.
type LDAPProvider struct {
	cfg      LDAPConfig
	pool     chan ldapConn
	mu       sync.Mutex
	dialFunc func(ctx context.Context) (ldapConn, error) // injectable for tests
	caPool   *x509.CertPool // optional custom CA pool
}

// NewLDAPProvider creates a new LDAPProvider with a connection pool.
func NewLDAPProvider(cfg LDAPConfig) (*LDAPProvider, error) {
	if cfg.ServerURL == "" {
		return nil, errors.InvalidArgument("LDAP server URL is required")
	}
	if cfg.BaseDN == "" {
		return nil, errors.InvalidArgument("LDAP base DN is required")
	}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(&(objectClass=user)(sAMAccountName=%s))"
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 5
	}
	if cfg.ConnTimeout == 0 {
		cfg.ConnTimeout = 10 * time.Second
	}

	p := &LDAPProvider{
		cfg:  cfg,
		pool: make(chan ldapConn, cfg.PoolSize),
	}
	p.dialFunc = p.dial
	return p, nil
}

// Type returns the provider type.
func (p *LDAPProvider) Type() ProviderType { return ProviderLDAP }

// Name returns the human-readable name.
func (p *LDAPProvider) Name() string { return "ldap" }

// Authenticate binds to LDAP with the user's credentials and retrieves attributes.
func (p *LDAPProvider) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	if creds.Username == "" || creds.Password == "" {
		return nil, errors.Unauthenticated("username and password are required")
	}

	conn, err := p.getConn(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "failed to get LDAP connection", err)
	}

	// Step 1: Bind with service account to search for the user DN.
	if err := conn.Bind(p.cfg.BindDN, p.cfg.BindPassword); err != nil {
		p.putConn(conn, false)
		return nil, errors.Wrap(errors.ErrInternal, "LDAP service bind failed", err)
	}

	// Step 2: Search for the user entry.
	userDN, attributes, err := p.searchUser(conn, creds.Username)
	if err != nil {
		p.putConn(conn, false)
		return nil, err
	}
	if userDN == "" {
		p.putConn(conn, true)
		return nil, errors.Unauthenticated("user not found in directory")
	}

	// Step 3: Re-bind as the user to verify the password.
	// We need a fresh connection for the user bind because the service bind state is on conn.
	userConn, err := p.dialFunc(ctx)
	if err != nil {
		p.putConn(conn, true)
		return nil, errors.Wrap(errors.ErrInternal, "failed to connect for user bind", err)
	}

	if err := userConn.Bind(userDN, creds.Password); err != nil {
		p.putConn(conn, true)
		closeQuietly(userConn)
		return nil, errors.Unauthenticated("LDAP authentication failed")
	}

	// User bind succeeded — return the connection to the pool.
	p.putConn(conn, true)
	p.putConn(userConn, true)

	result := &AuthResult{
		ExternalID: userDN,
		Provider:   ProviderLDAP,
		Attributes: attributes,
		NewUser:    p.cfg.AutoProvision,
	}

	// Map LDAP groups to roles if a mapping is configured.
	if len(p.cfg.GroupRoleMappings) > 0 {
		result.Roles = p.mapGroupsToRoles(attributes)
	}

	// If not auto-provisioning, the caller must link the user manually.
	if !p.cfg.AutoProvision {
		result.MustLink = true
	}

	return result, nil
}

// searchUser finds the user DN and attributes matching the filter.
func (p *LDAPProvider) searchUser(conn ldapConn, username string) (string, map[string]any, error) {
	filter := fmt.Sprintf(p.cfg.UserFilter, ldap.EscapeFilter(username))

	searchReq := ldap.NewSearchRequest(
		p.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"mail", "displayName", "memberOf", "sAMAccountName", "givenName", "sn"},
		nil,
	)

	sr, err := conn.Search(searchReq)
	if err != nil {
		return "", nil, errors.Wrap(errors.ErrInternal, "LDAP search failed", err)
	}
	if len(sr.Entries) == 0 {
		return "", nil, nil
	}

	entry := sr.Entries[0]
	attrs := map[string]any{}
	for _, attr := range entry.Attributes {
		if len(attr.Values) == 1 {
			attrs[attr.Name] = attr.Values[0]
		} else {
			attrs[attr.Name] = attr.Values
		}
	}
	return entry.DN, attrs, nil
}

// getConn retrieves a connection from the pool or dials a new one.
func (p *LDAPProvider) getConn(ctx context.Context) (ldapConn, error) {
	select {
	case conn := <-p.pool:
		if conn.IsClosing() {
			closeQuietly(conn)
			return p.dialFunc(ctx)
		}
		return conn, nil
	default:
		return p.dialFunc(ctx)
	}
}

// putConn returns a healthy connection to the pool or closes it.
func (p *LDAPProvider) putConn(conn ldapConn, healthy bool) {
	if !healthy || conn.IsClosing() {
		closeQuietly(conn)
		return
	}
	select {
	case p.pool <- conn:
	default:
		closeQuietly(conn)
	}
}

// dial establishes a new LDAP connection.
func (p *LDAPProvider) dial(ctx context.Context) (ldapConn, error) {
	opts := []ldap.DialOpt{
		ldap.DialWithDialer(&net.Dialer{Timeout: p.cfg.ConnTimeout}),
	}

	// For ldaps:// pass TLS config via DialOpt
	if strings.HasPrefix(p.cfg.ServerURL, "ldaps://") {
		opts = append(opts, ldap.DialWithTLSConfig(p.tlsConfig()))
	}

	conn, err := ldap.DialURL(p.cfg.ServerURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", p.cfg.ServerURL, err)
	}

	if p.cfg.StartTLS && !strings.HasPrefix(p.cfg.ServerURL, "ldaps://") {
		if err := conn.StartTLS(p.tlsConfig()); err != nil {
			closeQuietly(conn)
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}

	return conn, nil
}

func (p *LDAPProvider) tlsConfig() *tls.Config {
	if p.cfg.TLSConfig != nil {
		return p.cfg.TLSConfig
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if p.caPool != nil {
		cfg.RootCAs = p.caPool
	}
	return cfg
}

// SetCAPool sets a custom CA certificate pool for TLS verification.
// This allows the LDAP provider to connect to servers using self-signed
// or private CA certificates.
func (p *LDAPProvider) SetCAPool(pool *x509.CertPool) {
	p.caPool = pool
}

// Close drains the pool and closes all idle connections.
func (p *LDAPProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		select {
		case conn := <-p.pool:
			closeQuietly(conn)
		default:
			return
		}
	}
}

func closeQuietly(conn ldapConn) {
	_ = conn.Close()
}

// resolveTenantID extracts the tenant ID from the context.
// It is shared between providers to avoid circular imports on pkg/tenant.
func resolveTenantID(ctx context.Context) (uuid.UUID, error) {
	// Use the tenant package via context value.
	// We avoid importing pkg/tenant here to prevent a cycle;
	// the caller always attaches tenant info.
	type tenantCtx interface {
		GetTenantID() uuid.UUID
	}
	if tc, ok := ctx.Value(tenantCtxKey{}).(tenantCtx); ok {
		return tc.GetTenantID(), nil
	}
	return uuid.Nil, errors.New(errors.ErrFailedPrecondition, "no tenant context for authentication")
}

type tenantCtxKey struct{}

// WithTenantContext attaches a tenant-resolver to the context for use by providers.
// This avoids a circular import between authprovider and tenant packages.
func WithTenantContext(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tenantResolver{tenantID})
}

type tenantResolver struct {
	id uuid.UUID
}

func (t tenantResolver) GetTenantID() uuid.UUID { return t.id }

// ChainEnhanced extends Chain with the ability to skip providers that
// do not match a given type, and to collect per-provider errors for diagnostics.
type ChainEnhanced struct {
	providers []Provider
	only      map[ProviderType]struct{} // if non-nil, only try these types
}

// NewChainEnhanced creates an enhanced chain.
func NewChainEnhanced(providers ...Provider) *ChainEnhanced {
	return &ChainEnhanced{providers: providers}
}

// OnlyTypes restricts authentication to the given provider types.
func (c *ChainEnhanced) OnlyTypes(types ...ProviderType) *ChainEnhanced {
	c.only = make(map[ProviderType]struct{}, len(types))
	for _, t := range types {
		c.only[t] = struct{}{}
	}
	return c
}

// Authenticate tries each provider in order.
func (c *ChainEnhanced) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	var lastErr error
	for _, p := range c.providers {
		if c.only != nil {
			if _, ok := c.only[p.Type()]; !ok {
				continue
			}
		}
		result, err := p.Authenticate(ctx, creds)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.Unauthenticated("no auth providers configured")
	}
	return nil, lastErr
}

// ProviderTypes returns the types of all providers in the chain.
func (c *ChainEnhanced) ProviderTypes() []ProviderType {
	types := make([]ProviderType, len(c.providers))
	for i, p := range c.providers {
		types[i] = p.Type()
	}
	return types
}

// ProviderNames returns the names of all providers.
func (c *ChainEnhanced) ProviderNames() string {
	names := make([]string, len(c.providers))
	for i, p := range c.providers {
		names[i] = p.Name()
	}
	return strings.Join(names, ", ")
}

// mapGroupsToRoles extracts the user's LDAP groups from attributes and
// maps them to application roles using the configured GroupRoleMappings.
func (p *LDAPProvider) mapGroupsToRoles(attrs map[string]any) []string {
	// Extract group memberships from attributes.
	var groups []string
	if memberOf, ok := attrs["memberOf"]; ok {
		switch v := memberOf.(type) {
		case []string:
			groups = v
		case string:
			groups = []string{v}
		}
	}

	seen := map[string]bool{}
	var roles []string
	for _, mapping := range p.cfg.GroupRoleMappings {
		for _, group := range groups {
			if strings.EqualFold(group, mapping.GroupDN) && !seen[mapping.Role] {
				roles = append(roles, mapping.Role)
				seen[mapping.Role] = true
			}
		}
	}
	return roles
}
