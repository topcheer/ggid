# Token Exchange (RFC 8693) — Implementation Patterns for IAM Systems

> **Status**: Implementation Research & GGID Gap Analysis
> **Companion Document**: [RFC 8693 Full Spec Analysis](token-exchange-rfc8693.md) (1293 lines — covers protocol flow, parameters, token types, delegation semantics, act/azp claims)
> **Scope**: This document focuses on **Go implementation patterns**, audit trail design, security enforcement, and GGID-specific gap analysis. Protocol-level details are not duplicated.
> **Date**: 2025-07

---

## Table of Contents

1. [Impersonation vs Delegation](#1-impersonation-vs-delegation)
2. [Subject Token Validation](#2-subject-token-validation)
3. [Actor Token Requirements](#3-actor-token-requirements)
4. [Token Exchange Audit Trail](#4-token-exchange-audit-trail)
5. [Scope Downgrade and Upgrade](#5-scope-downgrade-and-upgrade)
6. [Audience Binding for Exchanged Tokens](#6-audience-binding-for-exchanged-tokens)
7. [Multi-Hop Delegation Security](#7-multi-hop-delegation-security)
8. [GGID Token Exchange Gap Analysis](#8-ggid-token-exchange-gap-analysis)
9. [Recommendations & Action Items](#9-recommendations--action-items)

---

## 1. Impersonation vs Delegation

The fundamental distinction in token exchange: **who is the system attributing actions to?**

### 1.1 Impersonation

In impersonation, the caller obtains a token whose `sub` claim is the **subject's** identity, with **no `act` claim**. The resulting token is indistinguishable from one the subject would have obtained directly.

```
Admin exchanges subject_token for user_token:
  sub: user@example.com     ← the subject
  // NO act claim           ← actor is invisible
  scope: read:profile
```

**When appropriate:**
- Administrative "login as user" for support workflows
- Service-to-service calls where the downstream system expects a user-scoped token and has no actor claim consumer
- Legacy system compatibility where `act` is not understood

**Security risks:**
- Actions are attributable only to the subject, not the actor — breaks accountability
- If the exchanged token leaks, forensic analysis cannot distinguish it from a token the user obtained directly
- Makes insider threat detection significantly harder

### 1.2 Delegation

In delegation, the caller obtains a token whose `sub` is the subject but an **`act` claim records the actor**. The resulting token carries a cryptographic proof that it was issued via exchange.

```
Service exchanges user_token for downstream_token:
  sub: user@example.com           ← the subject (user)
  act: { sub: order-service }     ← the actor (calling service)
  scope: read:orders
  aud: inventory-api              ← restricted audience
```

**When appropriate:**
- Microservice chains where each hop needs a scoped token
- "On behalf of" flows (user authorizes app to call API, app calls downstream API)
- Any scenario requiring attribution and audit trails

### 1.3 Go Implementation: Token Structures

The following Go code shows how to model both patterns using `github.com/golang-jwt/jwt/v5`, which is already a dependency in GGID's auth and OAuth services:

```go
// ActorClaim represents the nested "act" claim for delegation tokens.
// RFC 8693 §4.1: act is a JSON object containing claims about the actor.
type ActorClaim struct {
	Sub    string `json:"sub"`              // actor subject identifier
	Iss    string `json:"iss,omitempty"`    // actor issuer
	Act    *ActorClaim `json:"act,omitempty"` // nested actor for multi-hop
}

// ExchangeTokenClaims extends standard JWT claims with token-exchange fields.
type ExchangeTokenClaims struct {
	TenantID string      `json:"tenant_id"`
	Scopes   []string    `json:"scope"`
	Act      *ActorClaim `json:"act,omitempty"` // nil = impersonation; non-nil = delegation
	jwt.RegisteredClaims
}

// IssueImpersonationToken issues a token with the subject's identity but NO act claim.
// WARNING: The resulting token is indistinguishable from the subject's own token.
// Use only when audit trail can be established through other means.
func IssueImpersonationToken(
	key *rsa.PrivateKey,
	kid string,
	issuer string,
	subjectID string,
	tenantID string,
	scopes []string,
	audience string,
	ttl time.Duration,
) (string, error) {
	now := time.Now()
	claims := ExchangeTokenClaims{
		TenantID: tenantID,
		Scopes:   scopes,
		// Act is deliberately nil — this is impersonation
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subjectID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}

// IssueDelegationToken issues a token with the subject's identity AND an act claim.
// The actor's identity is permanently embedded in the token.
func IssueDelegationToken(
	key *rsa.PrivateKey,
	kid string,
	issuer string,
	subjectID string,
	tenantID string,
	scopes []string,
	audience string,
	actorSub string,
	actorIss string,
	parentAct *ActorClaim, // non-nil for multi-hop (see §7)
	ttl time.Duration,
) (string, error) {
	now := time.Now()

	// Build the act claim. If parentAct is present, nest it for chain-of-custody.
	act := &ActorClaim{
		Sub: actorSub,
		Iss: actorIss,
	}
	if parentAct != nil {
		act.Act = parentAct
	}

	claims := ExchangeTokenClaims{
		TenantID: tenantID,
		Scopes:   scopes,
		Act:      act,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subjectID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}
```

### 1.4 Audit Trail Implications

| Aspect | Impersonation | Delegation |
|--------|--------------|------------|
| Token-level attribution | None — `act` absent | Full — `act.sub` present |
| Forensic value | Must rely on exchange logs | Embedded in token + exchange logs |
| Token replay detection | Impossible from token alone | Actor visible in `act` claim |
| Compliance (SOC2/GDPR) | Requires compensating controls | Self-documenting |

**Recommendation**: Default to delegation (`act` present) for all token exchange flows. Reserve impersonation for explicitly flagged admin workflows with mandatory out-of-band audit logging.

---

## 2. Subject Token Validation

The subject token is the input to the exchange — it represents the entity whose identity will be propagated. Validation failures must be hard rejections.

### 2.1 Validation Requirements

1. **Signature valid** — the subject token must be cryptographically verified against the issuing IdP's keys
2. **Unexpired** — `exp` claim must be in the future
3. **Active** — if using opaque tokens, introspection must return `active: true`; for JWTs, check revocation list
4. **Correct issuer** — `iss` must match a trusted issuer in the tenant's federation config
5. **Not-before check** — `nbf` (if present) must be in the past
6. **Audience check** — the token exchange endpoint must be a valid audience for the subject token (prevents stolen tokens from other clients)

### 2.2 Go Implementation

```go
// SubjectTokenValidator validates tokens presented for exchange.
type SubjectTokenValidator struct {
	keyProvider KeyProvider
	issuer      string
	revocationChecker RevocationChecker // interface to Redis or DB
	trustedIssuers   map[string]bool   // federated issuers per tenant
}

// ValidatedSubject contains the parsed subject identity after validation.
type ValidatedSubject struct {
	SubjectID string
	TenantID  string
	Issuer    string
	Scopes    []string
	Audience  []string
	ExpiresAt time.Time
}

// ValidateSubjectToken validates a JWT subject token for exchange.
func (v *SubjectTokenValidator) ValidateSubjectToken(
	ctx context.Context,
	rawToken string,
) (*ValidatedSubject, error) {
	// Parse and verify signature
	token, err := jwt.ParseWithClaims(rawToken, &ExchangeTokenClaims{},
		func(t *jwt.Token) (interface{}, error) {
			// Enforce RS256
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return v.keyProvider.PublicKey(), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("subject token signature invalid: %w", err)
	}

	claims, ok := token.Claims.(*ExchangeTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("subject token claims invalid")
	}

	// Check expiration (jwt/v5 validates exp automatically, but be explicit for audit)
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("subject token expired")
	}

	// Check NBF if present
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return nil, fmt.Errorf("subject token not yet valid (nbf: %s)", claims.NotBefore.Time)
	}

	// Verify issuer is trusted
	if !v.trustedIssuers[claims.Issuer] {
		return nil, fmt.Errorf("untrusted subject token issuer: %s", claims.Issuer)
	}

	// Check revocation (jti-based for JWTs)
	if claims.ID != "" {
		revoked, err := v.revocationChecker.IsRevoked(ctx, claims.ID)
		if err != nil {
			return nil, fmt.Errorf("revocation check failed: %w", err)
		}
		if revoked {
			return nil, fmt.Errorf("subject token has been revoked (jti: %s)", claims.ID)
		}
	}

	return &ValidatedSubject{
		SubjectID: claims.Subject,
		TenantID:  claims.TenantID,
		Issuer:    claims.Issuer,
		Scopes:    claims.Scopes,
		Audience:  claims.Audience,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
```

### 2.3 Authorization Check: Who Can Exchange?

Not every authenticated caller may exchange a subject token. The authorization layer must enforce:

```go
// ExchangePolicy defines who is allowed to exchange tokens.
type ExchangePolicy struct {
	AllowedActors   map[string]bool // service accounts allowed to exchange
	MaxImpersonators map[string]bool // subjects allowed to be impersonated by admins
}

// AuthorizeExchange checks whether the actor is permitted to exchange
// a token for the given subject.
func (p *ExchangePolicy) AuthorizeExchange(
	actorID string,
	actorScopes []string,
	subject *ValidatedSubject,
	impersonation bool, // true = no act claim; false = delegation
) error {
	// Impersonation requires elevated privilege
	if impersonation {
		hasAdminScope := false
		for _, s := range actorScopes {
			if s == "admin:impersonate" || s == "system:impersonate" {
				hasAdminScope = true
				break
			}
		}
		if !hasAdminScope {
			return fmt.Errorf("actor %s lacks admin:impersonate scope for impersonation", actorID)
		}
	}

	// Delegation: actor must be in allowed set OR hold token-exchange scope
	if !impersonation {
		hasExchangeScope := false
		for _, s := range actorScopes {
			if s == "token:exchange" {
				hasExchangeScope = true
				break
			}
		}
		if !p.AllowedActors[actorID] && !hasExchangeScope {
			return fmt.Errorf("actor %s not authorized for token exchange", actorID)
		}
	}

	return nil
}
```

**Key principle**: Impersonation should require a scope (`admin:impersonate`) that is never granted to regular users or services. Only break-glass admin accounts and specifically designated service accounts should hold it.

---

## 3. Actor Token Requirements

The actor token authenticates the **party performing the exchange**. Even when the subject token is valid, the actor must independently prove its identity.

### 3.1 Actor Authentication Methods

| Method | When to Use | Security Properties |
|--------|-------------|-------------------|
| `client_secret_basic` | Confidential clients (web apps) | Shared secret over TLS |
| `client_secret_post` | Legacy compatibility | Less secure — secret in body |
| `private_key_jwt` | Service-to-service (highest security) | Asymmetric signature, no shared secret |
| `tls_client_auth` (mTLS) | Zero-trust service mesh | Certificate-bound, strongest transport auth |
| `none` (public client) | Only with PKCE-validated subject | Weakest — use sparingly |

### 3.2 Binding Actor to the Exchange Request

The actor token must be bound to this specific exchange request to prevent token replay or injection:

```go
// ActorValidator validates the actor credential and binds it to the request.
type ActorValidator struct {
	clientRepo ClientRepository
	trustedCAs *x509.CertPool // for mTLS
}

// ValidateActor validates the calling actor and returns its identity.
func (av *ActorValidator) ValidateActor(
	ctx context.Context,
	r *http.Request,
	subjectTenantID string,
) (*ActorIdentity, error) {
	// Determine actor authentication method
	switch {
	case r.TLS != nil && len(r.TLS.PeerCertificates) > 0:
		return av.validateMTLS(ctx, r, subjectTenantID)

	case r.Header.Get("Authorization") != "":
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			return av.validateBearerAssertion(ctx, r, subjectTenantID)
		}
		return av.validateClientSecret(ctx, r, subjectTenantID)

	case r.FormValue("client_assertion") != "":
		return av.validatePrivateKeyJWT(ctx, r, subjectTenantID)

	default:
		return nil, fmt.Errorf("no actor authentication provided")
	}
}

// validateMTLS validates mTLS certificate and binds it to the client.
func (av *ActorValidator) validateMTLS(
	ctx context.Context,
	r *http.Request,
	tenantID string,
) (*ActorIdentity, error) {
	cert := r.TLS.PeerCertificates[0]

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:     av.trustedCAs,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Verify(opts); err != nil {
		return nil, fmt.Errorf("mTLS certificate verification failed: %w", err)
	}

	// Extract client identity from cert subject
	clientID := cert.Subject.CommonName // or URI SANs
	if clientID == "" {
		return nil, fmt.Errorf("mTLS certificate missing CN")
	}

	// Verify this client_id is registered and mTLS-bound
	client, err := av.clientRepo.GetByClientID(ctx, tenantID, clientID)
	if err != nil {
		return nil, fmt.Errorf("unregistered mTLS client: %w", err)
	}
	if client.TokenEndpointAuthMethod != "tls_client_auth" {
		return nil, fmt.Errorf("client not configured for mTLS auth")
	}

	// Verify certificate fingerprint matches registered cert
	fp := sha256.Sum256(cert.Raw)
	registeredFP := client.Metadata["tls_client_cert_sha256"].(string)
	if hex.EncodeToString(fp[:]) != registeredFP {
		return nil, fmt.Errorf("mTLS certificate fingerprint mismatch")
	}

	return &ActorIdentity{
		ClientID: clientID,
		Scopes:   client.Scopes,
		AuthMethod: "tls_client_auth",
	}, nil
}

// validatePrivateKeyJWT validates a private_key_jwt client assertion.
func (av *ActorValidator) validatePrivateKeyJWT(
	ctx context.Context,
	r *http.Request,
	tenantID string,
) (*ActorIdentity, error) {
	assertion := r.FormValue("client_assertion")
	assertionType := r.FormValue("client_assertion_type")
	if assertionType != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
		return nil, fmt.Errorf("unsupported client_assertion_type")
	}

	// Parse without verification first to get client_id
	unverified := jwt.RegisteredClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(assertion, &unverified)
	if err != nil {
		return nil, fmt.Errorf("parse client assertion: %w", err)
	}

	// Look up client by iss claim
	client, err := av.clientRepo.GetByClientID(ctx, tenantID, unverified.Issuer)
	if err != nil {
		return nil, fmt.Errorf("unknown client in assertion: %w", err)
	}

	// Parse and verify with client's public key
	token, err := jwt.ParseWithClaims(assertion, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) {
			// Load client's registered public key (JWKS)
			return client.PublicKey()
		},
		jwt.WithIssuer(client.ClientID),
		jwt.WithSubject(client.ClientID),
		jwt.WithAudience(av.issuer), // must target our token endpoint
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid client assertion: %w", err)
	}

	return &ActorIdentity{
		ClientID:   client.ClientID,
		Scopes:     client.Scopes,
		AuthMethod: "private_key_jwt",
	}, nil
}

// ActorIdentity is the resolved actor after authentication.
type ActorIdentity struct {
	ClientID  string
	Scopes    []string
	AuthMethod string
}
```

**Key binding principle**: The actor's `client_id` becomes the `act.sub` in the issued token, creating an immutable link between the authenticated caller and the resulting delegation token.

---

## 4. Token Exchange Audit Trail

Every token exchange must generate an audit event. Without it, the `act` claim in the token is the only forensic trail — and tokens expire, get deleted, or may be unreadable by downstream audit systems.

### 4.1 Why Audit Every Exchange

| Risk Without Audit | Consequence |
|---|---|
| Actor identity lost after token expiry | Cannot investigate who performed action on behalf of user |
| No exchange rate limiting data | Undetected token laundering attacks |
| No scope reduction trail | Cannot prove least-privilege compliance |
| No multi-hop chain visibility | Blind to circular delegation or abuse |

### 4.2 Audit Event Structure

GGID's existing `audit.Event` struct (in `pkg/audit/publisher.go`) maps cleanly to token exchange events:

```go
// buildExchangeAuditEvent creates an audit event for a token exchange.
func buildExchangeAuditEvent(
	actor *ActorIdentity,
	subject *ValidatedSubject,
	requestedScopes []string,
	grantedScopes []string,
	audience string,
	impersonation bool,
	result string, // "success" or "failure"
	ipAddress string,
) *audit.Event {
	metadata := map[string]any{
		"grant_type":    "urn:ietf:params:oauth:grant-type:token-exchange",
		"requested_scope":  requestedScopes,
		"granted_scope":    grantedScopes,
		"audience":         audience,
		"impersonation":    impersonation,
		"subject_issuer":   subject.Issuer,
		"actor_auth_method": actor.AuthMethod,
	}

	action := "token.exchange"
	if impersonation {
		action = "token.impersonate"
	}

	return &audit.Event{
		ID:           uuid.New(),
		TenantID:     mustParseUUID(subject.TenantID),
		ActorType:    "api_client",
		ActorID:      mustParseUUID(actor.ClientID),
		ActorName:    actor.ClientID,
		Action:       action,
		ResourceType: "oauth_token",
		ResourceID:   mustParseUUID(subject.SubjectID),
		ResourceName: subject.SubjectID,
		Result:       result,
		IPAddress:    ipAddress,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
	}
}
```

### 4.3 Chain-of-Custody for Multi-Hop Delegation

When delegation chains across multiple services (A → B → C), each exchange generates an independent audit event. The `exchange_chain_id` links them:

```go
// ExchangeChainTracker tracks the delegation chain across hops.
type ExchangeChainTracker struct {
	publisher *audit.Publisher
}

// LogExchange logs a single hop in the delegation chain.
func (t *ExchangeChainTracker) LogExchange(
	ctx context.Context,
	chainID string,      // propagated through act.act.act... chain
	hopNumber int,       // 1 for first exchange, 2 for second, etc.
	actor *ActorIdentity,
	subject *ValidatedSubject,
	parentChainID string, // chain ID from the incoming subject token's act claim, if any
) error {
	event := buildExchangeAuditEvent(actor, subject, nil, nil, "", false, "success", "")
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	event.Metadata["exchange_chain_id"] = chainID
	event.Metadata["hop_number"] = hopNumber
	event.Metadata["parent_chain_id"] = parentChainID

	return t.publisher.Publish(ctx, event)
}

// DeriveChainID extracts or creates a chain ID from the subject token's act claim.
// If the subject token already has an act claim, we inherit its chain context.
func DeriveChainID(subjectAct *ActorClaim) string {
	if subjectAct != nil {
		// Multi-hop: derive from parent chain
		// In production, use a hash of the root actor's sub + creation timestamp
		return fmt.Sprintf("chain-%s-%d", subjectAct.Sub, time.Now().UnixMilli())
	}
	return fmt.Sprintf("chain-%s-%d", uuid.New().String(), time.Now().UnixMilli())
}
```

The audit system can then reconstruct the full delegation chain by querying events with the same `exchange_chain_id`, ordered by `hop_number`.

---

## 5. Scope Downgrade and Upgrade

### 5.1 The Principle

The exchanged token must have **equal or fewer scopes** than the subject token. This is the "monotonic scope reduction" principle:

```
subject_token.scope = [read:profile, write:orders, admin:users]
                        ↓ exchange (requested: [read:orders])
exchanged_token.scope = [read:orders]   ← MUST be subset of subject scopes
```

### 5.2 Enforcement

```go
// ValidateScopeDowngrade ensures the requested scope is a valid subset of subject scopes.
// Returns the scope set to grant (which may be further reduced by policy).
func ValidateScopeDowngrade(
	subjectScopes []string,
	requestedScopes []string,
) ([]string, error) {
	subjectSet := toSet(subjectScopes)

	// If no scope requested, grant a minimal default (NOT all subject scopes)
	if len(requestedScopes) == 0 {
		return []string{}, nil // no scopes — caller gets identity only
	}

	granted := make([]string, 0, len(requestedScopes))
	for _, s := range requestedScopes {
		if !subjectSet[s] {
			return nil, fmt.Errorf(
				"scope upgrade attempt: requested '%s' not in subject token scopes %v",
				s, subjectScopes,
			)
		}
		granted = append(granted, s)
	}

	return granted, nil
}

// ApplyScopePolicy further restricts granted scopes based on resource policy.
// For example, a resource server may only accept a whitelist of scopes.
func ApplyScopePolicy(
	grantedScopes []string,
	allowedByPolicy []string,
) []string {
	policySet := toSet(allowedByPolicy)
	result := make([]string, 0, len(grantedScopes))
	for _, s := range grantedScopes {
		if policySet[s] {
			result = append(result, s)
		}
	}
	return result
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, i := range items {
		m[i] = true
	}
	return m
}
```

### 5.3 Why Scope Reduction Matters

| Scenario | Without Reduction | With Reduction |
|---|---|---|
| Frontend → API gateway → backend | Backend gets full user scopes (write, admin) | Backend gets `read:orders` only |
| Mobile app → BFF → microservice | Microservice gets all app scopes | Microservice gets exactly what it needs |
| Cross-tenant service call | All tenant scopes leak to downstream | Only requested scopes propagate |

**Security invariant**: The set of scopes in any exchanged token must always be a subset of the scopes in the subject token. This must be enforced at issuance time — never trust the requested scope blindly.

---

## 6. Audience Binding for Exchanged Tokens

### 6.1 Why Audience Matters

Without audience binding, an exchanged token can be replayed against any service that trusts the issuer:

```
Attacker obtains exchanged token (scope: read:orders)
  → replays it against /api/v1/admin/users (different service)
  → if no audience check, the admin API trusts the token
```

### 6.2 Implementation

```go
// AudienceValidator ensures the requested audience is valid for token exchange.
type AudienceValidator struct {
	registeredAudiences map[string]bool // resource servers registered for token exchange
}

// ValidateAudience checks that the requested audience is registered and the
// subject token's client is allowed to target it.
func (av *AudienceValidator) ValidateAudience(
	requestedAudience string,
	subjectClientID string,
) error {
	if requestedAudience == "" {
		return fmt.Errorf("audience is required for token exchange (RFC 8693 §2.1)")
	}

	if !av.registeredAudiences[requestedAudience] {
		return fmt.Errorf("unregistered audience: %s", requestedAudience)
	}

	// Optionally: check if subjectClientID is authorized to target this audience
	// This prevents a service from obtaining tokens for unrelated services
	return nil
}

// IssueAudienceBoundToken issues a token restricted to a single audience.
func IssueAudienceBoundToken(
	key *rsa.PrivateKey,
	kid string,
	issuer string,
	subject *ValidatedSubject,
	actor *ActorIdentity,
	scopes []string,
	audience string,
	ttl time.Duration,
) (string, error) {
	// Enforce maximum TTL for exchanged tokens (should be short)
	maxTTL := 15 * time.Minute
	if ttl > maxTTL {
		ttl = maxTTL
	}

	now := time.Now()
	claims := ExchangeTokenClaims{
		TenantID: subject.TenantID,
		Scopes:   scopes,
		Act: &ActorClaim{
			Sub: actor.ClientID,
			Iss: issuer,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject.SubjectID,
			Audience:  jwt.ClaimStrings{audience}, // single audience
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(key)
}
```

### 6.3 Resource Server Validation

The downstream resource server must validate the `aud` claim:

```go
// ValidateAudienceForResource checks that the token's audience matches this service.
func ValidateAudienceForResource(claims *ExchangeTokenClaims, resourceID string) error {
	for _, aud := range claims.Audience {
		if aud == resourceID {
			return nil
		}
	}
	return fmt.Errorf("token audience %v does not include %s", claims.Audience, resourceID)
}
```

---

## 7. Multi-Hop Delegation Security

In microservice architectures, delegation can chain: Service A calls Service B, which calls Service C. Each hop should exchange tokens, creating a nested `act` chain.

### 7.1 Act Claim Nesting

```
Original token (user → Service A):
  sub: user@example.com
  act: null

After first exchange (A → B):
  sub: user@example.com
  act: { sub: service-a }

After second exchange (B → C):
  sub: user@example.com
  act: { sub: service-b, act: { sub: service-a } }
```

### 7.2 Depth Limiting and Circular Detection

```go
const MaxDelegationDepth = 3

// ValidateMultiHop validates the delegation chain depth and detects cycles.
func ValidateMultiHop(act *ActorClaim) error {
	depth := 0
	visited := make(map[string]bool)
	current := act

	for current != nil {
		depth++
		if depth > MaxDelegationDepth {
			return fmt.Errorf(
				"delegation chain exceeds max depth %d (RFC 8693 security recommendation)",
				MaxDelegationDepth,
			)
		}

		// Cycle detection
		if visited[current.Sub] {
			return fmt.Errorf("circular delegation detected: %s appears twice in act chain", current.Sub)
		}
		visited[current.Sub] = true

		current = current.Act
	}

	return nil
}

// CountHops returns the number of actors in the delegation chain.
func CountHops(act *ActorClaim) int {
	count := 0
	current := act
	for current != nil {
		count++
		current = current.Act
	}
	return count
}

// ExtractActorChain returns all actors in the delegation chain as a slice.
func ExtractActorChain(act *ActorClaim) []string {
	var chain []string
	current := act
	for current != nil {
		chain = append(chain, current.Sub)
		current = current.Act
	}
	return chain
}
```

### 7.3 Multi-Hop Exchange Handler

```go
// HandleMultiHopExchange processes a token exchange request with chain validation.
func HandleMultiHopExchange(
	ctx context.Context,
	subjectValidator *SubjectTokenValidator,
	actorValidator *ActorValidator,
	publisher *audit.Publisher,
	rawSubjectToken string,
	actor *ActorIdentity,
	requestedScopes []string,
	audience string,
	r *http.Request,
) (string, error) {
	// 1. Validate subject token
	subject, err := subjectValidator.ValidateSubjectToken(ctx, rawSubjectToken)
	if err != nil {
		return "", fmt.Errorf("subject validation: %w", err)
	}

	// 2. Parse existing act chain from subject token (if delegation)
	parentAct := extractActFromToken(rawSubjectToken)

	// 3. Validate multi-hop depth and detect cycles
	if parentAct != nil {
		if err := ValidateMultiHop(parentAct); err != nil {
			return "", err
		}
		if CountHops(parentAct) >= MaxDelegationDepth {
			return "", fmt.Errorf("max delegation depth reached")
		}
	}

	// 4. Validate scope downgrade
	granted, err := ValidateScopeDowngrade(subject.Scopes, requestedScopes)
	if err != nil {
		return "", err
	}

	// 5. Issue delegation token with nested act chain
	token, err := IssueDelegationToken(
		key, kid, issuer,
		subject.SubjectID, subject.TenantID,
		granted, audience,
		actor.ClientID, issuer,
		parentAct, // nest the parent chain
		15*time.Minute,
	)
	if err != nil {
		return "", err
	}

	// 6. Audit the exchange
	chainID := DeriveChainID(parentAct)
	hopNumber := CountHops(parentAct) + 1
	publisher.Publish(ctx, &audit.Event{
		ID:        uuid.New(),
		TenantID:  mustParseUUID(subject.TenantID),
		ActorType: "api_client",
		ActorID:   mustParseUUID(actor.ClientID),
		Action:    "token.exchange",
		Result:    "success",
		Metadata: map[string]any{
			"exchange_chain_id": chainID,
			"hop_number":        hopNumber,
			"actor_chain":       ExtractActorChain(parentAct),
			"subject_id":        subject.SubjectID,
			"granted_scopes":    granted,
		},
		CreatedAt: time.Now(),
	})

	return token, nil
}
```

---

## 8. GGID Token Exchange Gap Analysis

### 8.1 Current State

A thorough review of the GGID codebase reveals **no token exchange support**:

| Component | Status | Details |
|-----------|--------|---------|
| OAuth token endpoint (`server.go:325`) | Missing | Only `authorization_code`, `refresh_token`, `client_credentials` handled. Unknown grant types return `unsupported_grant_type` |
| OAuth client model (`domain/models.go`) | Missing | `SupportsGrantType()` exists but no `urn:ietf:params:oauth:grant-type:token-exchange` registered |
| Auth token service (`token_service.go`) | Missing | `AccessTokenClaims` has no `act` field, no actor/delegation concepts |
| Discovery endpoint | Missing | `GrantTypesSupported` in `OIDCDiscoveryConfig` does not include token exchange |
| Audit events | Partially ready | `audit.Event` struct can accommodate exchange events (via `Action` + `Metadata`), but no `token.exchange` action type is defined |
| Subject token validation | Missing | No JWT validation/revocation checking infrastructure in OAuth service |
| Actor authentication | Partially ready | Client secret auth exists; mTLS and `private_key_jwt` are referenced in `jar_mtls.go` but not wired to token endpoint |
| Scope validation | Missing | No scope intersection or downgrade logic exists |
| Audience binding | Missing | `AccessTokenClaims` sets audience to a single static config value, not per-request |

### 8.2 What Exists That Can Be Leveraged

1. **RSA key management** — Both auth and OAuth services load/create RSA keys. Token exchange can reuse the same `KeyProvider` interface.
2. **Client management** — `OAuthClient` model with `GrantTypes`, `Scopes`, `TokenEndpointAuthMethod` fields is ready to register token-exchange clients.
3. **Audit publisher** — `pkg/audit` NATS publisher with `Event.Metadata` map is flexible enough for exchange audit events.
4. **JWKS endpoint** — Already exposes public keys for token verification by downstream services.
5. **mTLS infrastructure** — `jar_mtls.go` has certificate validation code that can be extended for actor authentication.

### 8.3 Missing Components (Build List)

| Component | Priority | Complexity |
|-----------|----------|------------|
| Token exchange grant type handler in `server.go` | P0 | Medium |
| `ExchangeTokenClaims` struct with `act` field | P0 | Low |
| `SubjectTokenValidator` with signature/exp/revocation checks | P0 | Medium |
| `ExchangePolicy` for authorization (impersonation vs delegation) | P0 | Medium |
| `ActorValidator` with mTLS + private_key_jwt support | P1 | High |
| Scope downgrade enforcement (`ValidateScopeDowngrade`) | P1 | Low |
| Audience binding validation | P1 | Low |
| Multi-hop depth limiting + cycle detection | P1 | Low |
| Token exchange audit event generation | P1 | Low |
| Discovery endpoint update (add grant type) | P2 | Trivial |
| Database migration for token exchange audit trail | P2 | Low |
| Console UI for viewing delegation chains | P3 | Medium |

---

## 9. Recommendations & Action Items

### 9.1 Phased Implementation Plan

**Phase 1 — Foundation (1-2 sprints, ~5 days effort)**

| # | Action Item | Effort | Files |
|---|-------------|--------|-------|
| 1 | Add `ExchangeTokenClaims` struct with `act` field and `IssueDelegationToken()` / `IssueImpersonationToken()` to `token_service.go` | 1 day | `services/auth/internal/service/token_service.go` |
| 2 | Add token exchange grant type case in `server.go` token handler switch | 0.5 days | `services/oauth/internal/server/server.go` |
| 3 | Implement `SubjectTokenValidator` with signature, expiry, issuer, and revocation (jti) checks | 1.5 days | New file: `services/oauth/internal/service/token_exchange.go` |
| 4 | Implement `ExchangePolicy` authorization — require `admin:impersonate` scope for impersonation, `token:exchange` for delegation | 1 day | `services/oauth/internal/service/token_exchange.go` |
| 5 | Add token exchange audit event generation | 1 day | `services/oauth/internal/service/token_exchange.go` |

**Phase 2 — Hardening (1 sprint, ~3 days effort)**

| # | Action Item | Effort |
|---|-------------|--------|
| 6 | Implement scope downgrade validation (`ValidateScopeDowngrade`) — reject any requested scope not in subject token | 0.5 days |
| 7 | Implement audience binding — require `audience` parameter, validate against registered resource servers | 0.5 days |
| 8 | Add multi-hop depth limiting (max 3 hops) and circular delegation detection | 1 day |
| 9 | Wire `ActorValidator` for mTLS and `private_key_jwt` — extend existing `jar_mtls.go` patterns | 1 day |

**Phase 3 — Compliance (1 sprint, ~2 days effort)**

| # | Action Item | Effort |
|---|-------------|--------|
| 10 | Update discovery endpoint to advertise token exchange grant type | 0.5 days |
| 11 | Add database migration for token exchange audit trail (store `exchange_chain_id`, `hop_number`) | 0.5 days |
| 12 | Write integration tests: delegation flow, impersonation flow, scope downgrade rejection, multi-hop depth limit, circular detection | 1 day |

### 9.2 Security Decision Points

1. **Default to delegation, not impersonation**: Every exchange should produce an `act` claim by default. Impersonation (no `act`) should require an explicit `may_act` parameter or admin scope, and always log a `token.impersonate` audit event with elevated severity.

2. **Short TTL for exchanged tokens**: Exchanged tokens should have a maximum TTL of 15 minutes, significantly shorter than the standard access token TTL. This limits the blast radius if an exchanged token leaks.

3. **Mandatory audience parameter**: Unlike the base OAuth flow where audience is optional, token exchange should **require** the `audience` parameter. This prevents the "audience-less token" problem where exchanged tokens are replayable across services.

4. **Rate limiting per actor**: Each actor should be rate-limited on exchange requests (e.g., 100 exchanges/minute). Abnormal exchange volume is a strong indicator of compromise or abuse.

5. **Deny self-exchange**: An actor should not be able to exchange a token where `act.sub == subject.sub` (i.e., exchange a token for itself). This prevents infinite loops and is a common misconfiguration.

### 9.3 GGID-Specific Considerations

- **Multi-tenancy**: The `tenant_id` from the subject token must be propagated to the exchanged token. Cross-tenant exchanges should be denied by default unless explicitly federated.
- **RLS integration**: The exchanged token's `tenant_id` claim enables PostgreSQL Row-Level Security — the downstream service automatically sees only the correct tenant's data.
- **NATS audit pipeline**: Token exchange events flow through the existing NATS JetStream audit pipeline. No new infrastructure needed.
- **Gateway passthrough**: The API gateway (`services/gateway/`) must be updated to allow the token-exchange grant type through its token endpoint proxy. Currently it forwards to the OAuth service, so this should work with no changes if the OAuth service handles the new grant type.

---

## References

- [RFC 8693 — OAuth 2.0 Token Exchange](https://datatracker.ietf.org/doc/html/rfc8693)
- [GGID RFC 8693 Full Spec Analysis](token-exchange-rfc8693.md) — companion document with complete protocol details
- [RFC 8705 — OAuth 2.0 Mutual-TLS Client Authentication](https://datatracker.ietf.org/doc/html/rfc8705) — for actor mTLS
- [RFC 7523 — JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication](https://datatracker.ietf.org/doc/html/rfc7523) — for `private_key_jwt`
- GGID source: `services/oauth/internal/server/server.go` (token endpoint handler)
- GGID source: `services/auth/internal/service/token_service.go` (JWT claims and signing)
- GGID source: `pkg/audit/publisher.go` (audit event structure)
- GGID source: `services/oauth/internal/domain/models.go` (OAuth client model)
- GGID source: `services/oauth/internal/service/jar_mtls.go` (mTLS patterns)
