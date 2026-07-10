# OAuth Grant Type Validation and Per-Client Authorization

> Research document for the GGID IAM Suite. Covers security properties of each
> OAuth 2.0 / OIDC grant type, per-client authorization enforcement, token
> endpoint validation flow, and a full audit of the existing GGID OAuth service
> (`services/oauth/`).

---

## Table of Contents

1. [Grant Type Security Implications](#1-grant-type-security-implications)
2. [Per-Client Grant Type Allowlist](#2-per-client-grant-type-allowlist)
3. [Grant Type Validation at Token Endpoint](#3-grant-type-validation-at-token-endpoint)
4. [authorization_code Grant Security](#4-authorization_code-grant-security)
5. [refresh_token Grant Security](#5-refresh_token-grant-security)
6. [client_credentials Grant Security](#6-client_credentials-grant-security)
7. [device_code Grant Security](#7-device_code-grant-security)
8. [Custom Grant Types](#8-custom-grant-types)
9. [Grant Type Misuse Attack Scenarios](#9-grant-type-misuse-attack-scenarios)
10. [GGID Grant Type Audit](#10-ggid-grant-type-audit)
11. [Gap Analysis and Recommendations](#11-gap-analysis-and-recommendations)

---

## 1. Grant Type Security Implications

Each OAuth 2.0 grant type was designed for a specific interaction model and
carries different security assumptions. Using a grant type outside its intended
context weakens the overall security posture.

### Risk Matrix

| Grant Type | User Involvement | Credential Exposure | Consent | PKCE | Risk Level | When to Use |
|---|---|---|---|---|---|---|
| `authorization_code` | Browser redirect | None on token endpoint | Required | Recommended for all, mandatory for public | **Low** | Web apps, SPAs, mobile |
| `refresh_token` | None (existing session) | None | Not required | N/A | **Low-Medium** | All clients needing long-lived sessions |
| `client_credentials` | None (machine identity) | Client secret | Not applicable | N/A | **Medium** | Server-to-server APIs |
| `urn:ietf:params:oauth:grant-type:device_code` | Secondary device | None on token endpoint | User approves on second device | N/A | **Medium** | TVs, IoT, CLI tools |
| `password` (Resource Owner Password) | Direct credentials | User password exposed to client | Skipped | N/A | **High** | Deprecated (OAuth 2.1 removes it) |
| `urn:ietf:params:oauth:grant-type:jwt-bearer` | None (assertion) | Signed JWT assertion | Not required | N/A | **Medium-High** | Federation, cross-domain trust |

### Key Security Properties

**authorization_code** — The gold standard. The user's credentials never touch
the client application. The browser redirect + short-lived code + PKCE creates a
defense-in-depth chain. An attacker would need to intercept the redirect *and*
know the PKCE verifier to redeem the code.

**refresh_token** — Extends a session without re-authentication. If stolen, an
attacker can impersonate the user until expiry. Rotation and reuse detection are
critical mitigations (see Section 5).

**client_credentials** — No user context. The token represents the client
itself, not a person. The security boundary is the client secret. If the secret
is compromised, the attacker gets a machine identity with potentially broad
scopes.

**device_code** — Designed for input-constrained devices. The device shows a
code on screen; the user enters it on a separate device (phone/laptop). The
device polls the token endpoint. Risk: if the device is compromised, an attacker
can observe the user code and approve on the user's behalf before they do.

**password (ROPC)** — Removed in OAuth 2.1. The client directly handles user
credentials, breaking the OAuth security model. Should only be used for
first-party migration scenarios and only if no alternative exists.

```go
// RiskLevel categorizes grant types by inherent security risk.
type RiskLevel string

const (
	RiskLow       RiskLevel = "low"
	RiskMedium    RiskLevel = "medium"
	RiskHigh      RiskLevel = "high"
)

// GrantSecurityProfile describes the security characteristics of a grant type.
type GrantSecurityProfile struct {
	GrantType          string
	RiskLevel          RiskLevel
	RequiresUserConsent bool
	SupportsPKCE        bool
	InvolvesCredentials bool // does the client see user credentials?
	RecommendedFor      []string
}

// StandardGrantProfiles returns the security profile for all standard grants.
func StandardGrantProfiles() map[string]GrantSecurityProfile {
	return map[string]GrantSecurityProfile{
		"authorization_code": {
			GrantType:           "authorization_code",
			RiskLevel:           RiskLow,
			RequiresUserConsent: true,
			SupportsPKCE:        true,
			InvolvesCredentials: false,
			RecommendedFor:      []string{"web_app", "spa", "mobile", "native"},
		},
		"refresh_token": {
			GrantType:           "refresh_token",
			RiskLevel:           RiskMedium,
			RequiresUserConsent: false,
			SupportsPKCE:        false,
			InvolvesCredentials: false,
			RecommendedFor:      []string{"all_clients"},
		},
		"client_credentials": {
			GrantType:           "client_credentials",
			RiskLevel:           RiskMedium,
			RequiresUserConsent: false,
			SupportsPKCE:        false,
			InvolvesCredentials: false,
			RecommendedFor:      []string{"service_account", "machine_to_machine"},
		},
		"urn:ietf:params:oauth:grant-type:device_code": {
			GrantType:           "urn:ietf:params:oauth:grant-type:device_code",
			RiskLevel:           RiskMedium,
			RequiresUserConsent: true, // approved on second device
			SupportsPKCE:        false,
			InvolvesCredentials: false,
			RecommendedFor:      []string{"tv", "iot", "cli"},
		},
		"password": {
			GrantType:           "password",
			RiskLevel:           RiskHigh,
			RequiresUserConsent: false, // skipped
			SupportsPKCE:        false,
			InvolvesCredentials: true,
			RecommendedFor:      []string{}, // deprecated — do not recommend
		},
	}
}
```

---

## 2. Per-Client Grant Type Allowlist

### Why Not All Clients Should Use All Grants

The token endpoint accepts any client ID + grant type combination in many
implementations. This is a critical security flaw. Each client should be
registered with an explicit **allowlist** of grant types it may use. Without
per-client enforcement, an attacker can:

- Use `client_credentials` with a public client that has no secret
- Use `password` grant to bypass the consent screen of `authorization_code`
- Use `refresh_token` with a code stolen from a different flow

### Client Type to Grant Type Mapping

| Client Type | Allowed Grants | Forbidden Grants | Rationale |
|---|---|---|---|
| **SPA (public)** | `authorization_code` + PKCE, `refresh_token` | `client_credentials`, `password` | No secret storage; PKCE replaces it |
| **Web app (confidential)** | `authorization_code`, `refresh_token` | `password` (unless first-party) | Has secret; consent flow available |
| **Service account** | `client_credentials` | All user-based grants | Machine identity, no user |
| **Mobile (public)** | `authorization_code` + PKCE, `refresh_token`, `device_code` | `client_credentials`, `password` | Limited input, no secret storage |
| **CLI tool** | `device_code`, `authorization_code` + PKCE | `client_credentials`, `password` | No browser may be available |
| **First-party app** | May additionally use `password` | — | Only for trusted first-party migration |

```go
// ValidateClientGrants checks that the grant types requested during client
// registration are compatible with the client type. Returns an error if any
// grant type is incompatible.
func ValidateClientGrants(clientType ClientType, grantTypes []string) error {
	allowedByType := map[ClientType]map[string]bool{
		ClientTypePublic: {
			"authorization_code": true,
			"refresh_token":      true,
			"urn:ietf:params:oauth:grant-type:device_code": true,
		},
		ClientTypeConfidential: {
			"authorization_code": true,
			"refresh_token":      true,
			"client_credentials": true,
			"urn:ietf:params:oauth:grant-type:device_code": true,
			"urn:ietf:params:oauth:grant-type:jwt-bearer":  true,
		},
	}

	allowed, ok := allowedByType[clientType]
	if !ok {
		return fmt.Errorf("unknown client type: %s", clientType)
	}

	for _, gt := range grantTypes {
		if !allowed[gt] {
			return fmt.Errorf("grant type %q is not allowed for %s clients", gt, clientType)
		}
	}
	return nil
}

// CheckGrantAllowed verifies that a specific client is authorized to use a
// specific grant type at the token endpoint. This is the per-request check.
func CheckGrantAllowed(client *OAuthClient, grantType string) error {
	if !client.Enabled {
		return ErrClientDisabled
	}
	if len(client.GrantTypes) == 0 {
		// No allowlist configured — fail closed (deny by default).
		return fmt.Errorf("client has no registered grant types")
	}
	for _, allowed := range client.GrantTypes {
		if allowed == grantType {
			return nil
		}
	}
	return fmt.Errorf("grant type %q is not authorized for client %s", grantType, client.ClientID)
}
```

### GGID Current State

The GGID `domain.OAuthClient` struct stores `GrantTypes []string` and has a
`SupportsGrantType(gt string)` helper (see `domain/models.go:57-64`). The
`refresh_token` and `client_credentials` service methods *do* call
`SupportsGrantType` before proceeding. However, the `authorization_code` handler
in `server.go` and the `device_code` handler do **not** perform this check.
This is a gap (see Section 10).

---

## 3. Grant Type Validation at Token Endpoint

The token endpoint (`/oauth/token`) is the central point where all grant types
converge. A robust validation pipeline is essential.

### Validation Pipeline

```
Request → Client Auth → Extract grant_type → Check client's allowlist
  → Validate grant-specific params → Issue token → Response
```

**Step 1: Client Authentication**
- Confidential clients: verify `client_secret` via Basic auth or POST body.
- Public clients: verify `client_id` exists and is public.

**Step 2: Extract grant_type**
- Must be present. Empty or missing → `invalid_request`.
- Unknown value → `unsupported_grant_type` (HTTP 400).

**Step 3: Per-Client Grant Allowlist Check**
- Look up the client. Check `client.SupportsGrantType(grantType)`.
- Not authorized → `unauthorized_client` (HTTP 400).

**Step 4: Grant-Specific Parameter Validation**
- `authorization_code`: requires `code`, `redirect_uri`, optional `code_verifier`.
- `refresh_token`: requires `refresh_token`.
- `client_credentials`: requires valid client secret.
- `device_code`: requires `device_code`.

**Step 5: Issue Token**
- Generate access token with appropriate claims.
- Return token response with `Cache-Control: no-store`.

```go
// GrantHandler processes a specific grant type at the token endpoint.
type GrantHandler interface {
	// Handle processes the grant request and returns a token response or error.
	Handle(ctx context.Context, client *OAuthClient, form url.Values) (*TokenResponse, *GrantError)
}

// GrantError wraps OAuth2 error codes for the token endpoint.
type GrantError struct {
	Code        string // OAuth2 error code (e.g., "unsupported_grant_type")
	Description string
	HTTPStatus  int
}

func (e *GrantError) Error() string { return e.Code + ": " + e.Description }

// GrantRegistry routes grant types to their handlers.
type GrantRegistry struct {
	handlers map[string]GrantHandler
}

func NewGrantRegistry() *GrantRegistry {
	return &GrantRegistry{handlers: make(map[string]GrantHandler)}
}

func (r *GrantRegistry) Register(grantType string, handler GrantHandler) {
	r.handlers[grantType] = handler
}

// ProcessTokenRequest is the complete token endpoint validation flow.
func (r *GrantRegistry) ProcessTokenRequest(
	ctx context.Context,
	clientRepo ClientRepository,
	form url.Values,
) (*TokenResponse, *GrantError) {
	// Step 1: Extract grant_type.
	grantType := form.Get("grant_type")
	if grantType == "" {
		return nil, &GrantError{
			Code: "invalid_request", Description: "grant_type is required",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// Step 2: Resolve the handler.
	handler, ok := r.handlers[grantType]
	if !ok {
		return nil, &GrantError{
			Code: "unsupported_grant_type",
			Description: fmt.Sprintf("grant type %q is not supported", grantType),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// Step 3: Client authentication.
	clientID := form.Get("client_id")
	clientSecret := form.Get("client_secret")
	client, err := clientRepo.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, &GrantError{
			Code: "invalid_client", Description: "client authentication failed",
			HTTPStatus: http.StatusUnauthorized,
		}
	}
	if !client.Enabled {
		return nil, &GrantError{
			Code: "invalid_client", Description: "client is disabled",
			HTTPStatus: http.StatusUnauthorized,
		}
	}
	if client.IsConfidential() {
		ok, _ := crypto.VerifyPassword(clientSecret, client.ClientSecretHash)
		if !ok {
			return nil, &GrantError{
				Code: "invalid_client", Description: "invalid client credentials",
				HTTPStatus: http.StatusUnauthorized,
			}
		}
	}

	// Step 4: Per-client grant allowlist.
	if !client.SupportsGrantType(grantType) {
		return nil, &GrantError{
			Code: "unauthorized_client",
			Description: fmt.Sprintf("client is not authorized to use grant type %q", grantType),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// Step 5: Delegate to the grant-specific handler.
	return handler.Handle(ctx, client, form)
}
```

---

## 4. authorization_code Grant Security

The authorization code flow is the most secure grant type when implemented
correctly. Its security depends on four pillars:

### 4.1 One-Time Use

An authorization code must be usable exactly once. After redemption, it is
permanently consumed. If the same code is presented again, the request must be
rejected. GGID implements this via `codeRepo.ConsumeCode()` which atomically
marks the code as used.

### 4.2 Redirect URI Matching

The `redirect_uri` in the token request must **exactly match** the one used
during the authorization request. This prevents code injection attacks where an
attacker tricks the server into redirecting the code to their endpoint.

### 4.3 PKCE Verification

Proof Key for Code Exchange (RFC 7636) prevents code interception attacks. The
client sends a `code_verifier` at the token endpoint; the server verifies it
against the stored `code_challenge`.

### 4.4 Code Expiry

Codes should be short-lived: 30 seconds to 10 minutes maximum. GGID uses 10
minutes (`time.Now().Add(10 * time.Minute)`).

```go
// RedeemAuthorizationCode securely processes an authorization_code grant.
func (s *OAuthService) RedeemAuthorizationCode(
	ctx context.Context,
	client *OAuthClient,
	code string,
	redirectURI string,
	codeVerifier string,
) (*TokenResponse, error) {
	// 1. Atomically consume the code (one-time use).
	storedCode, err := s.codeRepo.ConsumeCode(ctx, hashCode(code))
	if err != nil {
		return nil, fmt.Errorf("invalid_grant: authorization code not found or already used")
	}

	// 2. Verify code belongs to this client (prevents code injection).
	if storedCode.ClientID != client.ID {
		// SECURITY: This is a code injection attempt. Log and reject.
		return nil, fmt.Errorf("invalid_grant: code was issued to a different client")
	}

	// 3. Check expiry.
	if storedCode.IsExpired() {
		return nil, fmt.Errorf("invalid_grant: authorization code has expired")
	}

	// 4. Exact redirect_uri match (RFC 6749 Section 4.1.3).
	if storedCode.RedirectURI != redirectURI {
		return nil, fmt.Errorf("invalid_grant: redirect_uri mismatch")
	}

	// 5. PKCE verification.
	if !storedCode.ValidatePKCE(codeVerifier) {
		return nil, fmt.Errorf("invalid_grant: PKCE verification failed")
	}

	// 6. Issue tokens.
	accessToken, expiresIn, err := s.issueAccessToken(storedCode.UserID, client)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(storedCode.Scope),
	}, nil
}
```

### Code Reuse Detection (RFC 6749 Section 4.1.2)

Per RFC 6749, if an authorization code is used more than once, the authorization
server **must attempt to revoke all tokens previously issued** based on that
code. This indicates a potential compromise:

```go
// ConsumeCodeWithRevocation atomically consumes a code and revokes any
// tokens that were issued from a previous (fraudulent) redemption attempt.
func (s *OAuthService) ConsumeCodeWithRevocation(
	ctx context.Context,
	codeHash string,
) (*domain.AuthorizationCode, error) {
	code, err := s.codeRepo.ConsumeCode(ctx, codeHash)
	if err != nil {
		// If the code was already consumed, this is a reuse attempt.
		// Revoke all tokens issued from the original redemption.
		originalCode, getErr := s.codeRepo.GetCodeByHash(ctx, codeHash)
		if getErr == nil && originalCode != nil {
			_ = s.tokenRepo.RevokeTokensForClient(
				ctx, originalCode.TenantID, originalCode.ClientID,
			)
		}
		return nil, fmt.Errorf("invalid_grant: code reuse detected — all tokens revoked")
	}
	return code, nil
}
```

> **Note:** GGID's `AuthorizationCodeRepository.ConsumeCode` returns an error
> if the code is already consumed, which prevents replay. However, it does not
> revoke previously issued tokens upon reuse detection. This is a gap.

---

## 5. refresh_token Grant Security

### 5.1 Token Rotation

On each refresh token use, a **new** refresh token is issued and the old one is
invalidated. This limits the window of opportunity for a stolen token.

### 5.2 Reuse Detection (Token Family Revocation)

If a previously-used (revoked) refresh token is presented again, the entire
token family must be revoked. This detects compromise: the attacker has the old
token, and the legitimate user also used it, triggering rotation. When the
attacker presents the old token, we know the chain is compromised.

GGID implements this correctly in `RefreshToken()` (oauth_service.go:774-777):

```go
// 5. Reuse detection: if the token was already used or revoked, revoke ALL tokens.
if record.Used || record.Revoked {
	_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, req.TenantID, client.ID)
	return nil, errors.Unauthenticated("refresh token reuse detected — all tokens revoked")
}
```

### 5.3 Scope Subset Enforcement

The new scope requested during refresh must be a **subset** of the original
scope. A client cannot escalate privileges during refresh:

```go
// ValidateScopeSubset ensures requested scopes do not exceed the original.
func ValidateScopeSubset(original, requested []string) error {
	originalSet := make(map[string]bool)
	for _, s := range original {
		originalSet[s] = true
	}
	for _, s := range requested {
		if !originalSet[s] {
			return fmt.Errorf("invalid_scope: requested scope %q exceeds original grant", s)
		}
	}
	return nil
}

// SecureRefreshToken issues a new token pair with rotation + reuse detection.
func (s *OAuthService) SecureRefreshToken(
	ctx context.Context,
	client *OAuthClient,
	refreshToken string,
	requestedScope []string,
) (*TokenResponse, error) {
	// 1. Verify client supports refresh_token grant.
	if !client.SupportsGrantType("refresh_token") {
		return nil, fmt.Errorf("unauthorized_client: client does not support refresh_token")
	}

	// 2. Look up the refresh token.
	tokenHash := hashTokenSHA256(refreshToken)
	record, err := s.tokenRepo.GetRefreshToken(ctx, client.TenantID, tokenHash)
	if err != nil || record == nil {
		return nil, fmt.Errorf("invalid_grant: refresh token not found")
	}

	// 3. Reuse detection — revoke entire token family.
	if record.Used || record.Revoked {
		_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, client.TenantID, client.ID)
		return nil, fmt.Errorf("invalid_grant: refresh token reuse detected")
	}

	// 4. Expiry check.
	if time.Now().After(record.ExpiresAt) {
		return nil, fmt.Errorf("invalid_grant: refresh token expired")
	}

	// 5. Scope subset enforcement.
	effectiveScope := requestedScope
	if len(requestedScope) == 0 {
		effectiveScope = record.Scope // retain original scope
	} else if err := ValidateScopeSubset(record.Scope, requestedScope); err != nil {
		return nil, err
	}

	// 6. Rotate: invalidate old, issue new.
	_ = s.tokenRepo.RevokeRefreshToken(ctx, client.TenantID, tokenHash)

	accessToken, expiresIn, _ := s.issueAccessToken(record.UserID, client.TenantID, client.ClientID)
	newRefresh, _ := crypto.GenerateRandomToken(32)
	newRecord := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  client.TenantID,
		ClientID:  client.ID,
		UserID:    record.UserID,
		TokenHash: hashTokenSHA256(newRefresh),
		Scope:     effectiveScope,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	_ = s.tokenRepo.StoreRefreshToken(ctx, newRecord)

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: newRefresh,
		Scope:        joinScopes(effectiveScope),
	}, nil
}
```

> **Note:** GGID's current `RefreshToken` implementation does **not** validate
> that the requested scope is a subset of the original. It blindly uses
> `req.Scope` (line 806). This is a scope escalation vulnerability.

---

## 6. client_credentials Grant Security

### Key Properties

1. **No user involvement** — The token represents the client, not a user. The
   `sub` claim is the client ID, not a user UUID.
2. **No refresh token** — The client re-authenticates each time. There is no
   refresh token because the client always has its credentials.
3. **Scope limitation** — The issued scope must be limited to the client's
   registered scopes, not arbitrary requested scopes.
4. **Confidential only** — Only confidential clients (those with a secret) may
   use this grant. Public clients cannot prove their identity.

```go
// SecureClientCredentials validates and issues a client_credentials token.
func (s *OAuthService) SecureClientCredentials(
	ctx context.Context,
	client *OAuthClient,
	clientSecret string,
	requestedScope []string,
) (*TokenResponse, error) {
	// 1. Only confidential clients may use client_credentials.
	if !client.IsConfidential() {
		return nil, fmt.Errorf("unauthorized_client: public clients cannot use client_credentials")
	}

	// 2. Verify client secret.
	ok, _ := crypto.VerifyPassword(clientSecret, client.ClientSecretHash)
	if !ok {
		return nil, fmt.Errorf("invalid_client: authentication failed")
	}

	// 3. Check grant allowlist.
	if !client.SupportsGrantType("client_credentials") {
		return nil, fmt.Errorf("unauthorized_client: client does not support client_credentials")
	}

	// 4. Scope enforcement: requested scope must be subset of registered scopes.
	effectiveScope := requestedScope
	if len(requestedScope) == 0 {
		effectiveScope = client.Scopes // default to all registered scopes
	} else {
		registered := make(map[string]bool)
		for _, s := range client.Scopes {
			registered[s] = true
		}
		for _, s := range requestedScope {
			if !registered[s] {
				return nil, fmt.Errorf("invalid_scope: scope %q not registered for this client", s)
			}
		}
	}

	// 5. Issue token with uuid.Nil as subject (machine identity, no user).
	accessToken, expiresIn, err := s.issueAccessToken(uuid.Nil, client.TenantID, client.ClientID)
	if err != nil {
		return nil, err
	}

	// 6. NO refresh token for client_credentials.
	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(effectiveScope),
		// RefreshToken intentionally omitted
	}, nil
}
```

> **Note:** GGID's current `ClientCredentials` method checks client secret and
> grant type support, which is good. However:
> - It does not verify that the client is confidential (public clients with
>   the grant type could theoretically use it).
> - It does not enforce scope subset against registered scopes.
> - It does not issue a refresh token (correct behavior).

---

## 7. device_code Grant Security

The device authorization flow (RFC 8628) has unique security considerations:

### 7.1 Device Code Validation

- The `device_code` must exist in the active store.
- It must not be expired (GGID uses 15 minutes).
- It must be in `pending` or `approved` status.

### 7.2 Polling Rate Enforcement

RFC 8628 requires the server to return `slow_down` if the client polls faster
than the specified `interval`. GGID implements this with a 5-second minimum
interval.

### 7.3 Status Responses

| Response | Meaning |
|---|---|
| `authorization_pending` | User hasn't approved yet |
| `slow_down` | Polling too fast — increase interval |
| `expired_token` | Device code has expired |
| `access_denied` | User explicitly denied |
| `invalid_grant` | Device code not found |

```go
// SecurePollDeviceToken validates device code with rate limiting and status checks.
func (s *OAuthService) SecurePollDeviceToken(
	ctx context.Context,
	client *OAuthClient,
	deviceCode string,
) (*TokenResponse, *DevicePollError) {
	info, ok := getDeviceCode(deviceCode)
	if !ok {
		return nil, &DevicePollError{Code: "expired_token", Description: "device code not found"}
	}

	// 1. Verify client matches the one that initiated the flow.
	if info.ClientID != client.ClientID {
		return nil, &DevicePollError{Code: "invalid_grant", Description: "device code was issued to a different client"}
	}

	// 2. Verify grant allowlist.
	if !client.SupportsGrantType("urn:ietf:params:oauth:grant-type:device_code") {
		return nil, &DevicePollError{Code: "unauthorized_client", Description: "client does not support device_code grant"}
	}

	// 3. Check expiry.
	if time.Now().After(info.ExpiresAt) {
		removeDeviceCode(deviceCode)
		return nil, &DevicePollError{Code: "expired_token", Description: "device code expired"}
	}

	// 4. Handle status.
	switch info.Status {
	case "pending":
		// Rate-limit: enforce minimum polling interval.
		if info.LastPoll != nil && time.Since(*info.LastPoll) < 5*time.Second {
			return nil, &DevicePollError{Code: "slow_down", Description: "polling too fast"}
		}
		now := time.Now()
		info.LastPoll = &now
		return nil, &DevicePollError{Code: "authorization_pending", Description: "user has not yet approved"}

	case "denied":
		return nil, &DevicePollError{Code: "access_denied", Description: "user denied the request"}

	case "approved":
		if info.UserID == nil {
			return nil, &DevicePollError{Code: "authorization_pending", Description: "approved but no user"}
		}
		token, expiresIn, err := s.issueDeviceAccessToken(info.TenantID, *info.UserID)
		if err != nil {
			return nil, &DevicePollError{Code: "server_error", Description: err.Error()}
		}
		removeDeviceCode(deviceCode)
		return &TokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			Scope:       joinScopes(info.Scope),
		}, nil

	default:
		return nil, &DevicePollError{Code: "invalid_grant", Description: "unknown status"}
	}
}

type DevicePollError struct {
	Code        string
	Description string
}
```

> **Note:** GGID's `PollDeviceToken` correctly handles all status transitions
> and slow_down enforcement. However, it does **not** check the grant type
> allowlist and does **not** verify that the requesting `client_id` matches the
> one that initiated the device flow.

---

## 8. Custom Grant Types

### 8.1 Safe Registration

Custom grant types (e.g., `urn:ggid:biometric`, `urn:ietf:params:oauth:grant-type:jwt-bearer`)
follow the URN naming convention. They must be:

1. **Explicitly registered** per client (not globally available).
2. **Subject to the same validation pipeline** as standard grants.
3. **Audited** — any custom grant must log all invocations.

```go
// ExtensibleGrantRegistry allows runtime registration of custom grant handlers.
type ExtensibleGrantRegistry struct {
	handlers    map[string]GrantHandler
	mu          sync.RWMutex
	auditLogger AuditLogger
}

// RegisterGrant adds a new grant type to the registry.
func (r *ExtensibleGrantRegistry) RegisterGrant(
	grantType string,
	handler GrantHandler,
	requireConfidential bool,
) error {
	if !strings.HasPrefix(grantType, "urn:") && !isStandardGrant(grantType) {
		return fmt.Errorf("custom grant types must use URN notation (urn:...)")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[grantType]; exists {
		return fmt.Errorf("grant type %q already registered", grantType)
	}
	r.handlers[grantType] = &auditingHandler{
		inner:               handler,
		grantType:           grantType,
		requireConfidential: requireConfidential,
		auditLogger:         r.auditLogger,
	}
	return nil
}

// auditingHandler wraps a grant handler with audit logging and client-type checks.
type auditingHandler struct {
	inner               GrantHandler
	grantType           string
	requireConfidential bool
	auditLogger         AuditLogger
}

func (h *auditingHandler) Handle(ctx context.Context, client *OAuthClient, form url.Values) (*TokenResponse, *GrantError) {
	// Enforce client type requirement.
	if h.requireConfidential && !client.IsConfidential() {
		return nil, &GrantError{
			Code: "unauthorized_client",
			Description: fmt.Sprintf("grant %q requires a confidential client", h.grantType),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// Log invocation.
	h.auditLogger.Log(ctx, AuditEvent{
		GrantType: h.grantType,
		ClientID:  client.ClientID,
		Action:    "grant_invocation",
	})

	resp, err := h.inner.Handle(ctx, client, form)
	if err != nil {
		h.auditLogger.Log(ctx, AuditEvent{
			GrantType: h.grantType,
			ClientID:  client.ClientID,
			Action:    "grant_failed",
			Detail:    err.Code,
		})
	}
	return resp, err
}

func isStandardGrant(gt string) bool {
	switch gt {
	case "authorization_code", "refresh_token", "client_credentials":
		return true
	default:
		return false
	}
}
```

### 8.2 Security Implications

- **Biometric grants** (`urn:ggid:biometric`) should require hardware-backed
  attestation. The grant handler must verify the attestation before issuing a
  token.
- **JWT-bearer grants** (`urn:ietf:params:oauth:grant-type:jwt-bearer`) must
  verify the assertion signature against a trusted issuer's JWKS, not just
  parse it unverified as GGID currently does.
- Custom grants should never bypass consent — if user data is involved, consent
  must be collected beforehand and stored.

---

## 9. Grant Type Misuse Attack Scenarios

### Scenario A: Grant Switching to Bypass Consent

**Attack:** A malicious third-party app is registered with `authorization_code`
grant (which requires user consent). The attacker sends the user's credentials
using `password` grant instead, bypassing the consent screen entirely.

**Mitigation:** Per-client grant allowlist. If the client is not registered for
`password` grant, the request is rejected with `unauthorized_client`.

```go
// PreventGrantSwitching ensures the same client cannot switch grant types
// to bypass security controls.
func (s *OAuthService) ValidateGrantRequest(
	ctx context.Context,
	client *OAuthClient,
	grantType string,
) error {
	// Strict per-client allowlist.
	if !client.SupportsGrantType(grantType) {
		// Log suspicious activity.
		s.auditLogger.Log(ctx, AuditEvent{
			ClientID:  client.ClientID,
			Action:    "grant_type_mismatch",
			Detail:    fmt.Sprintf("client attempted %q but only allows %v", grantType, client.GrantTypes),
		})
		return fmt.Errorf("unauthorized_client: grant type not permitted for this client")
	}

	// Additional check: if the client was issued a code via authorization_code,
	// reject any attempt to use a different grant with the same client session.
	if grantType == "password" && client.IsThirdParty() {
		return fmt.Errorf("unauthorized_client: third-party clients may not use password grant")
	}

	return nil
}
```

### Scenario B: Cross-Grant Token Replay

**Attack:** An attacker intercepts an authorization code. Instead of using it
with `authorization_code` grant, they try to use it as a `refresh_token` grant,
hoping the server confuses the two.

**Mitigation:** Grant-specific token stores. Authorization codes and refresh
tokens must be stored in separate tables/namespaces. A code cannot be used as a
refresh token and vice versa.

```go
// IsolationTokenStore ensures tokens are never interchangeable across grants.
type IsolationTokenStore struct {
	codeStore    AuthorizationCodeRepository
	refreshStore RefreshTokenRepository
}

// ValidateTokenForGrant checks that a token is valid only for its original grant.
func (s *IsolationTokenStore) ValidateTokenForGrant(
	token string,
	expectedGrant string,
) error {
	switch expectedGrant {
	case "authorization_code":
		if _, err := s.codeStore.ConsumeCode(ctx, hashCode(token)); err != nil {
			return fmt.Errorf("invalid_grant: not a valid authorization code")
		}
	case "refresh_token":
		if _, err := s.refreshStore.Get(ctx, hashTokenSHA256(token)); err != nil {
			return fmt.Errorf("invalid_grant: not a valid refresh token")
		}
	default:
		return fmt.Errorf("unsupported_grant_type")
	}
	return nil
}
```

### Scenario C: Public Client Using client_credentials

**Attack:** A public client (no secret) attempts `client_credentials` grant to
get a machine-scoped token with elevated permissions.

**Mitigation:** The `client_credentials` handler must verify `client.IsConfidential()`.
GGID's current implementation checks the secret hash but does not explicitly
reject public clients.

---

## 10. GGID Grant Type Audit

### What Exists

| Feature | Status | Location |
|---|---|---|
| Grant type switch in token handler | Implemented | `server.go:325-386` |
| `authorization_code` grant | Implemented | `oauth_service.go:298-359` |
| `refresh_token` grant | Implemented | `oauth_service.go:746-817` |
| `client_credentials` grant | Implemented | `oauth_service.go:830-867` |
| `device_code` grant (RFC 8628) | Implemented | `oauth_service.go:1232-1290` |
| `jwt-bearer` grant (RFC 7523) | Implemented | `oauth_service.go:1450-1525` |
| `password` grant | **Not implemented** | (correctly omitted) |
| Unknown grant → `unsupported_grant_type` | Implemented | `server.go:383-385` |
| `SupportsGrantType` on client | Implemented | `domain/models.go:57-64` |
| Refresh token rotation + reuse detection | Implemented | `oauth_service.go:774-808` |
| PKCE validation | Implemented | `domain/models.go:122-139` |
| Code one-time use (ConsumeCode) | Implemented | `oauth_service.go:314-317` |
| Redirect URI exact match | Implemented | `oauth_service.go:325-327` |
| Auth code expiry (10 min) | Implemented | `oauth_service.go:259` |
| Device code slow_down enforcement | Implemented | `oauth_service.go:1251-1253` |
| Client registration with grant_types field | Implemented | `server.go:766`, `oauth_service.go:77-107` |
| Dynamic client registration with grant_types | Implemented | `oauth_service.go:1036-1037` |
| Discovery `grant_types_supported` | Implemented | `oauth_service.go:375` |

### What Is Missing (Gaps)

| Gap | Severity | Description |
|---|---|---|
| **G1: No grant allowlist check for `authorization_code`** | High | The token handler does not call `client.SupportsGrantType("authorization_code")` before issuing a token. A client registered with only `client_credentials` could redeem auth codes. |
| **G2: No grant allowlist check for `device_code`** | Medium | `PollDeviceToken` does not verify the client supports `device_code` grant. |
| **G3: No scope subset validation in refresh_token** | High | `RefreshToken` uses `req.Scope` directly without checking it is a subset of the original scope. Scope escalation possible. |
| **G4: No scope subset validation in client_credentials** | Medium | `ClientCredentials` does not verify requested scope is a subset of the client's registered scopes. |
| **G5: No confidential-only check for client_credentials** | Medium | Public clients are not explicitly rejected from using `client_credentials`. Secret check happens but public clients have no secret hash. |
| **G6: JWT-bearer assertion not signature-verified** | Critical | `JWTBearerGrant` uses `ParseUnverified` — the assertion is never validated against a trusted issuer. Anyone can forge a JWT. |
| **G7: No code reuse detection with token revocation** | Medium | While `ConsumeCode` prevents replay, if the same code is used twice (race), previously issued tokens are not revoked. |
| **G8: No client_id match in device flow** | Low | `PollDeviceToken` does not verify the polling `client_id` matches the one that initiated the device flow. |
| **G9: Token endpoint lacks client auth for public clients on some grants** | Low | The token handler trusts `client_id` from POST body without verifying the client is public. |
| **G10: `password` grant absent from discovery** | Informational | Correctly not advertised, but no explicit blocklist prevents future addition. |

---

## 11. Gap Analysis and Recommendations

### Prioritized Action Items

| # | Action | Gap(s) Addressed | Effort | Priority |
|---|---|---|---|---|
| 1 | **Add `SupportsGrantType` check in `ExchangeAuthorizationCode`** | G1 | 1 hour | P0 |
| 2 | **Add scope subset validation in `RefreshToken`** | G3 | 2 hours | P0 |
| 3 | **Verify JWT-bearer assertion signature against trusted JWKS** | G6 | 1 day | P0 |
| 4 | **Add `IsConfidential()` check in `ClientCredentials`** | G5 | 30 min | P1 |
| 5 | **Add scope subset validation in `ClientCredentials`** | G4 | 1 hour | P1 |
| 6 | **Add grant allowlist + client_id match in `PollDeviceToken`** | G2, G8 | 2 hours | P1 |
| 7 | **Implement `GrantRegistry` pattern for centralized validation** | G1, G2, G9 | 1 day | P2 |

### Effort Summary

- **P0 items** (3): ~1.5 days — fix critical security gaps
- **P1 items** (3): ~4 hours — harden existing flows
- **P2 items** (1): 1 day — architectural improvement

### Long-Term Recommendations

1. **Adopt the GrantRegistry pattern** — Move from the current `switch` statement
   in `server.go` to a registry-based approach. Each grant handler encapsulates
   its own validation logic. This makes it trivial to add new grants safely and
   ensures consistent enforcement of per-client allowlists.

2. **Enforce grant types at registration time** — When a client is created via
   `CreateClient` or dynamic registration, validate that the requested grant
   types are compatible with the client type (see Section 2). Currently, any
   client can register any grant types without validation.

3. **Audit logging for all grant invocations** — Log every token request with
   the grant type, client ID, and outcome. This enables detection of grant type
   misuse patterns.

4. **OAuth 2.1 alignment** — Consider removing support for the `password` grant
   type permanently and documenting this in the OAuth 2.1 migration plan. The
   `implicit` flow should also be deprecated.

5. **Rate limiting per grant type** — Apply differentiated rate limits: tighter
   on `authorization_code` (to prevent code brute-force), looser on
   `client_credentials` (legitimate high-frequency machine traffic).

---

## References

- RFC 6749 — The OAuth 2.0 Authorization Framework
- RFC 7636 — Proof Key for Code Exchange by OAuth Public Clients (PKCE)
- RFC 7523 — JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication and Authorization Grants
- RFC 8628 — OAuth 2.0 Device Authorization Grant
- RFC 8693 — OAuth 2.0 Token Exchange
- RFC 7009 — OAuth 2.0 Token Revocation
- RFC 7662 — OAuth 2.0 Token Introspection
- OAuth 2.1 Draft — https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/
- OIDC Core 1.0 — https://openid.net/specs/openid-connect-core-1_0.html
