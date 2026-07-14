# ADR-003: Pluggable Authentication Provider Chain

**Status:** Accepted
**Date:** 2024-Q1
**Deciders:** Architecture Team

---

## Context

GGID must support multiple authentication methods simultaneously:
- **Local passwords** (bcrypt hashed in database)
- **LDAP / Active Directory** (bind to external directory)
- **OAuth / OIDC** (delegate to external IdP)
- **SAML 2.0** (enterprise federated SSO)
- **Social login** (Google, GitHub, Microsoft, etc.)

Different deployments need different combinations:
- Small company: Local passwords only
- Enterprise: Local + LDAP + SAML
- Consumer app: Local + Social login
- Hybrid: Local + LDAP + OAuth

The auth system must be flexible enough to support any combination without code changes.

### Alternatives Considered

#### Option A: Monolithic Auth Handler (Hardcoded)

```go
func Authenticate(username, password string) (*User, error) {
    // Try local DB first
    user, err := localDB.Lookup(username, password)
    if err == nil { return user, nil }

    // Try LDAP
    if ldapConfigured {
        user, err = ldap.Bind(username, password)
        if err == nil { return user, nil }
    }

    // Try OAuth... (but OAuth doesn't use username/password)
    return nil, ErrInvalidCredentials
}
```

**Pros:**
- Simple to understand
- No abstraction layer

**Cons:**
- Adding a new auth method requires modifying the auth handler
- Cannot disable/enable providers at runtime
- Cannot change provider order
- Testing requires mocking all providers
- Mixing credential-based (password) and token-based (OAuth) flows is messy

#### Option B: Strategy Pattern (Individual Config)

```go
// Configure each provider independently
localProvider := NewLocalProvider(db)
ldapProvider := NewLDAPProvider(config)

// Client code must know which provider to use
switch authMethod {
case "local": user = localProvider.Authenticate(...)
case "ldap": user = ldapProvider.Authenticate(...)
}
```

**Pros:**
- Each provider is independent
- Easy to test individually

**Cons:**
- Client code must know all providers
- No automatic fallback (try local, then LDAP)
- No central configuration
- Duplicated provider management logic

#### Option C: Chain of Responsibility (Selected)

```go
// Providers implement a common interface
type Provider interface {
    Name() string
    Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error)
}

// Chain tries providers in order
chain := NewChain(localProvider, ldapProvider)
result, err := chain.Authenticate(ctx, cred)
```

**Pros:**
- Add/remove providers without code changes
- Automatic fallback through chain
- Common interface for all providers
- Easy to test with mocks
- Runtime configuration (enable/disable via config)

**Cons:**
- Slight overhead from chain iteration
- Must handle different credential types
- Error aggregation (which provider failed?)

---

## Decision

Choose **Chain of Responsibility** with a common `Provider` interface.

### Interface Design

```go
// pkg/authprovider/provider.go

type Provider interface {
    // Name returns the provider identifier
    Name() string

    // Authenticate attempts to verify the credentials
    // Returns AuthResult on success, error on failure
    Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error)

    // Available checks if the provider is configured and reachable
    Available(ctx context.Context) bool
}

type Credentials struct {
    Username  string
    Password  string
    Token     string  // For OAuth/social flows
    Method    string  // "password", "oauth", "webauthn", "saml"
}

type AuthResult struct {
    UserID    string
    TenantID  string
    Username  string
    Email     string
    Roles     []string
    Provider  string  // Which provider authenticated
    Attributes map[string]string  // Extra attributes (LDAP groups, etc.)
}
```

### Chain Implementation

```go
// pkg/authprovider/chain.go

type Chain struct {
    providers []Provider
}

func NewChain(providers ...Provider) *Chain {
    return &Chain{providers: providers}
}

func (c *Chain) Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error) {
    var lastErr error

    for _, provider := range c.providers {
        if !provider.Available(ctx) {
            continue  // Skip unavailable providers
        }

        result, err := provider.Authenticate(ctx, cred)
        if err == nil {
            // Success — return immediately
            return result, nil
        }

        // Remember error, try next provider
        lastErr = err
    }

    // All providers failed
    return nil, fmt.Errorf("authentication failed: %w", lastErr)
}
```

### Provider Implementations

| Provider | Package | Credential Type | Auto-Provision |
|----------|---------|---------------|----------------|
| `LocalProvider` | `authprovider` | Username + password | N/A (already local) |
| `LDAPProvider` | `authprovider` | Username + password | Yes (configurable) |
| `OAuthProvider` | `oauth/service` | Authorization code | Yes |
| `SAMLProvider` | `pkg/saml` | SAML assertion | Yes |
| Social connectors | `pkg/social` | OAuth code | Yes |

### Configuration

```bash
# Auth service main.go
chain := authprovider.NewChain(
    authprovider.NewLocalProvider(db, crypto),
)

if ldapURL != "" {
    chain.Add(authprovider.NewLDAPProvider(ldapConfig))
}

// Social connectors configured separately via OAuth flow
```

### LDAP Auto-Provisioning

When `LDAP_AUTO_PROVISION=true`:
1. LDAP bind succeeds (user authenticated against directory)
2. User does not exist in local database
3. Create local user record with LDAP attributes (email, name)
4. Assign default role (`end_user`)
5. Return successful AuthResult

This allows LDAP users to have roles, permissions, and audit trails in GGID without pre-registration.

---

## Consequences

### Positive

- **Extensible**: Add a new provider by implementing the `Provider` interface — no changes to existing code.
- **Configurable**: Enable/disable providers at startup via environment variables.
- **Testable**: Mock the `Provider` interface in unit tests. No LDAP/OAuth server needed.
- **Ordered fallback**: Local → LDAP → (other). If local fails, automatically tries LDAP.
- **Attribute passthrough**: LDAP groups, OAuth scopes, SAML attributes are preserved in `AuthResult.Attributes`.

### Negative

- **Chain overhead**: Iterating providers adds ~100ns per failed provider. Negligible with 2-3 providers.
- **Mixed credential types**: Password-based and token-based auth use different flows. OAuth/SAML don't go through the chain — they have separate entry points.
- **Error masking**: If all providers fail, only the last error is returned. Makes debugging harder (which provider was closest to succeeding?).
- **State management**: LDAP connection pools, OAuth client state, and SAML metadata must be managed per provider.

### Design Note: Why Not All Providers in the Chain?

Password-based providers (Local, LDAP) use the chain because they share the same credential format (username + password). Token-based providers (OAuth, SAML, Social) have different entry points:

- **OAuth/Social**: Redirect-based flow (not username/password) — separate endpoint
- **SAML**: Assertion-based flow — separate endpoint
- **WebAuthn**: Challenge-response — separate endpoint

The chain handles the credential-based auth path. Token-based auth has its own handlers but produces the same `AuthResult` structure for downstream consistency.

---

## Future Considerations

1. **Plugin system**: Load providers dynamically as Go plugins (requires Go plugin build mode)
2. **Risk-based provider selection**: Use ABAC rules to select providers based on risk score (e.g., require LDAP for admin login)
3. **Provider health metrics**: Track success/failure rate per provider for observability
4. **Provider-specific rate limiting**: Different rate limits for LDAP vs local (LDAP is slower)

---

## References

- [Chain of Responsibility Pattern](https://refactoring.guru/design-patterns/chain-of-responsibility)
- [GGID Authentication Guide](../authentication-guide.md)
- [GGID LDAP Integration](../ldap-directory-sync.md)
- [GGID Social Login](../social-login-guide.md)
- Related: [authentication-flow.md](./authentication-flow.md)

---

*Last updated: 2025-07-11*