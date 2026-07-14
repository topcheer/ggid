# OIDC Privacy-Preserving Authorization and Authentication Context (RFC 9701)

> Research document for the GGID IAM suite.
> Topic: RFC 9701 — Privacy-Preserving Authorization and Authentication Context in OpenID Connect.
> Status: Research / Gap Analysis — not yet implemented in GGID.

---

## Table of Contents

1. [RFC 9701 Overview](#1-rfc-9701-overview)
2. [Selective Disclosure](#2-selective-disclosure)
3. [Pairwise Subject Identifiers (PPID)](#3-pairwise-subject-identifiers-ppid)
4. [Essential vs Voluntary Claims](#4-essential-vs-voluntary-claims)
5. [Authentication Context Class (ACR)](#5-authentication-context-class-acr)
6. [AMR (Authentication Methods References)](#6-amr-authentication-methods-references)
7. [Privacy-Preserving Token Design](#7-privacy-preserving-token-design)
8. [Consent-Gated Claim Release](#8-consent-gated-claim-release)
9. [GGID Privacy-Preserving Auth Gap Analysis](#9-ggid-privacy-preserving-auth-gap-analysis)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. RFC 9701 Overview

### What Is Privacy-Preserving Authorization?

Standard OIDC issues ID tokens that frequently contain more user data than the relying party (RP) actually needs. A typical ID token includes `sub`, `email`, `name`, `picture`, `preferred_username`, and `tenant_id` — regardless of whether the RP requested all of them. This constitutes a privacy violation: every RP that completes an OIDC flow receives the user's full profile, creating unnecessary data exposure and enabling cross-service user tracking.

RFC 9701 (alongside the foundational OIDC Core specification, Sections 5.5 and 8.1) defines a framework for **privacy-preserving authorization** built on four pillars:

| Pillar | Purpose | Mechanism |
|--------|---------|-----------|
| **Selective Disclosure** | Release only requested claims | `claims` request parameter |
| **Pairwise Subject Identifiers** | Prevent cross-client correlation | Per-client `sub` via HMAC |
| **Minimal Claim Tokens** | Reduce data in access tokens | Audience-scoped claims |
| **Consent-Gated Release** | User controls data sharing | Per-claim consent records |

### Problem Solved

Consider a user who authenticates to three different services via GGID:

1. **Calendar app** — only needs `sub` and `email`
2. **Photo sharing app** — needs `sub`, `name`, `picture`
3. **Banking portal** — needs `sub`, `email`, `email_verified`, plus `acr` proving MFA

Without privacy-preserving features, all three services receive the same ID token containing the full profile. The banking portal's `acr` requirement is not enforced. The photo app can correlate the user with the calendar app because both see the same `sub` value. The user has no say in what data goes where.

With RFC 9701 compliance:

- Each service receives only the claims it requested (selective disclosure)
- Each service sees a different `sub` value (PPID), preventing correlation
- The banking portal's ID token includes an `acr` claim proving MFA was used
- The user explicitly consented to each claim release

### Relationship to GGID

GGID's OAuth service currently advertises `subject_types_supported: ["public"]` and issues ID tokens with a fixed claim set (`iss`, `sub`, `aud`, `iat`, `exp`, `nonce`, `tenant_id`). It does not parse the OIDC `claims` request parameter, does not support pairwise pseudonymous identifiers, and populates ACR/AMR only structurally (the `IDTokenOptions` struct exists but is always passed `nil`). This document analyzes each gap and provides implementation guidance.

---

## 2. Selective Disclosure

### Concept

Selective disclosure means the authorization server releases **only the claims the RP explicitly requested** — no more. The OIDC Core specification defines two mechanisms for requesting specific claims:

1. **Scope-based**: Standard scopes like `profile`, `email`, `address`, `phone` map to predefined claim sets. The RP requests a scope and receives the associated claims.

2. **Claims parameter**: The RP provides a JSON object in the `claims` authorization request parameter specifying exactly which claims it wants, and whether each is essential or voluntary:

```json
{
  "userinfo": {
    "email": null,
    "email_verified": null
  },
  "id_token": {
    "acr": {"essential": true, "values": ["urn:mace:incommon:iap:silver"]}
  }
}
```

The `claims` parameter is more granular than scopes. It allows requesting a single claim from the `profile` scope without requesting all profile claims. It also distinguishes between `id_token` (claims embedded in the ID token) and `userinfo` (claims available via the UserInfo endpoint).

### Per-Scope Claim Mapping

The standard OIDC scope-to-claim mapping:

| Scope | Claims |
|-------|--------|
| `profile` | `name`, `family_name`, `given_name`, `middle_name`, `nickname`, `preferred_username`, `profile`, `picture`, `website`, `gender`, `birthdate`, `zoneinfo`, `locale`, `updated_at` |
| `email` | `email`, `email_verified` |
| `address` | `address` (JSON object with `formatted`, `street_address`, `locality`, `region`, `postal_code`, `country`) |
| `phone` | `phone_number`, `phone_number_verified` |
| `openid` | `sub` (always included) |

### Go Code: Selective Claim Release Engine

```go
package oauth

import "encoding/json"

// ScopeClaimMap defines which claims each standard OIDC scope grants.
var ScopeClaimMap = map[string][]string{
	"openid":   {"sub"},
	"profile":  {"name", "family_name", "given_name", "middle_name", "nickname",
		"preferred_username", "profile", "picture", "website", "gender",
		"birthdate", "zoneinfo", "locale", "updated_at"},
	"email":    {"email", "email_verified"},
	"address":  {"address"},
	"phone":    {"phone_number", "phone_number_verified"},
}

// ClaimsRequest represents the parsed OIDC claims request parameter.
// Each entry maps a claim name to its requirements.
type ClaimsRequestEntry struct {
	Essential bool     `json:"essential,omitempty"`
	Value     any      `json:"value,omitempty"`
	Values    []any    `json:"values,omitempty"`
}

type ClaimsRequest struct {
	IDToken  map[string]ClaimsRequestEntry `json:"id_token,omitempty"`
	UserInfo map[string]ClaimsRequestEntry `json:"userinfo,omitempty"`
}

// ParseClaimsRequest parses the JSON claims parameter from the authorization request.
func ParseClaimsRequest(raw string) (*ClaimsRequest, error) {
	if raw == "" {
		return &ClaimsRequest{}, nil
	}
	var cr ClaimsRequest
	if err := json.Unmarshal([]byte(raw), &cr); err != nil {
		return nil, fmt.Errorf("invalid claims parameter: %w", err)
	}
	return &cr, nil
}

// SelectiveClaimEngine resolves which claims to release for a given request.
type SelectiveClaimEngine struct {
	// UserClaims supplies all available claims for a user.
	// In practice, this is fetched from the identity service.
}

// ResolveClaims determines the final claim set based on scopes + claims parameter.
// It intersects the user's available data with the requested claims.
func ResolveClaims(
	userClaims map[string]any,
	scopes []string,
	claimsReq *ClaimsRequest,
	target string, // "id_token" or "userinfo"
) map[string]any {
	result := make(map[string]any)

	// 1. Always include "sub".
	if sub, ok := userClaims["sub"]; ok {
		result["sub"] = sub
	}

	// 2. Add scope-based claims.
	for _, scope := range scopes {
		for _, claim := range ScopeClaimMap[scope] {
			if val, ok := userClaims[claim]; ok {
				result[claim] = val
			}
		}
	}

	// 3. Apply claims parameter refinement for the target (id_token or userinfo).
	if claimsReq != nil {
		var requested map[string]ClaimsRequestEntry
		switch target {
		case "id_token":
			requested = claimsReq.IDToken
		case "userinfo":
			requested = claimsReq.UserInfo
		}

		// If the claims parameter is present for this target, it RESTRICTS
		// the claim set: only explicitly requested claims are released.
		if len(requested) > 0 {
			restricted := make(map[string]any)
			restricted["sub"] = result["sub"] // sub always stays
			for claimName := range requested {
				if val, ok := userClaims[claimName]; ok {
					restricted[claimName] = val
				}
			}
			result = restricted
		}
	}

	return result
}
```

### Usage in Authorization Flow

When `ExchangeAuthorizationCode` issues an ID token, the selective claim engine determines which claims are embedded:

```go
// In the token exchange handler:
userClaims := fetchUserClaims(ctx, userID)
idTokenClaims := ResolveClaims(userClaims, code.Scope, claimsReq, "id_token")
userInfoClaims := ResolveClaims(userClaims, code.Scope, claimsReq, "userinfo")

idToken := s.issueIDTokenSelective(userID, tenantID, clientID, nonce, idTokenClaims)
```

---

## 3. Pairwise Subject Identifiers (PPID)

### How PPID Works

In standard OIDC, the `sub` claim in every ID token is the user's canonical identifier (typically the UUID). This means every RP sees the same `sub` value for the same user. Two RPs can trivially correlate users by comparing `sub` values — if `sub` matches, it's the same person.

**Pairwise pseudonymous identifiers (PPID)** solve this by generating a **different `sub` for each client**. Client A sees `sub = "a1b2c3..."` while Client B sees `sub = "d4e5f6..."`. Neither value reveals the canonical user ID, and the two values are computationally unlinkable.

### Algorithm

The OIDC specification defines the algorithm as:

```
sub = HMAC-SHA-256(sector_identifier_salt, user_id || sector_identifier)
                     truncated to the appropriate length
```

Where:
- **`sector_identifier`**: Derived from the RP's `redirect_uri` domain. If the RP has multiple redirect URIs, or if a `sector_identifier_uri` is provided, it's the explicit sector identifier. Otherwise, it's the host portion of the `redirect_uri`.
- **`sector_identifier_salt`**: A per-tenant secret salt stored server-side. Different salts per tenant provide additional isolation.
- **`user_id`**: The canonical user identifier (UUID in GGID).
- **HMAC**: SHA-256, truncated to 255 bits (or the full digest, base64url-encoded).

### Sector Identifier Resolution

```go
// Sector identifier derivation rules (OIDC Core §8.1):
// 1. If sector_identifier_uri is set, fetch and validate it.
// 2. If the client has exactly one redirect_uri, use its host.
// 3. If the client has multiple redirect_uris, sector_identifier_uri is REQUIRED.
```

### Go Code: PPID Generation

```go
package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
)

// PPIDGenerator produces pairwise pseudonymous subject identifiers.
type PPIDGenerator struct {
	salt []byte // per-tenant secret salt, at least 32 bytes
}

func NewPPIDGenerator(salt []byte) *PPIDGenerator {
	return &PPIDGenerator{salt: salt}
}

// SectorIdentifier computes the sector identifier from redirect URIs.
// If the client has one redirect URI, the sector ID is its host.
// If multiple, the caller must provide sector_identifier_uri explicitly.
func SectorIdentifier(redirectURIs []string, sectorIdentifierURI string) (string, error) {
	if sectorIdentifierURI != "" {
		// Fetch and validate the sector_identifier_uri JSON array.
		// For simplicity here, return the URI's host.
		u, err := url.Parse(sectorIdentifierURI)
		if err != nil {
			return "", fmt.Errorf("invalid sector_identifier_uri: %w", err)
		}
		return u.Host, nil
	}

	if len(redirectURIs) == 1 {
		u, err := url.Parse(redirectURIs[0])
		if err != nil {
			return "", fmt.Errorf("invalid redirect_uri: %w", err)
		}
		return u.Host, nil
	}

	// Multiple redirect URIs without sector_identifier_uri is an error
	// when pairwise subject type is used.
	return "", fmt.Errorf("sector_identifier_uri required when multiple redirect_uris")
}

// Generate produces a pairwise sub for a given user and sector.
// The output is deterministic: same user + same sector + same salt = same sub.
func (g *PPIDGenerator) Generate(userID, sectorIdentifier string) string {
	mac := hmac.New(sha256.New, g.salt)
	mac.Write([]byte(userID))
	mac.Write([]byte(sectorIdentifier))
	digest := mac.Sum(nil)

	// Base64url-encode without padding (OIDC recommends ASCII-safe sub values).
	return base64.RawURLEncoding.EncodeToString(digest)
}

// Verify checks whether a given sub matches the expected pairwise value.
func (g *PPIDGenerator) Verify(sub, userID, sectorIdentifier string) bool {
	expected := g.Generate(userID, sectorIdentifier)
	return hmac.Equal([]byte(sub), []byte(expected))
}
```

### Integration with Authorization Server

The authorization server selects the subject type per-client:

```go
func (s *OAuthService) resolveSubject(userID uuid.UUID, client *Client) string {
	switch client.SubjectType {
	case "pairwise":
		sector, err := SectorIdentifier(client.RedirectURIs, client.SectorIdentifierURI)
		if err != nil {
			// Fall back to public if sector resolution fails.
			return userID.String()
		}
		return s.ppid.Generate(userID.String(), sector)
	default:
		return userID.String()
	}
}
```

### Security Considerations

- The salt must never be exposed to clients. If compromised, all PPIDs become reversible.
- The salt should be **rotated** periodically. Rotation invalidates all existing pairwise subs, forcing re-authentication.
- The salt must be **unique per tenant** to prevent cross-tenant correlation.
- The salt must be **at least 32 bytes** (256 bits) to prevent brute-force.

---

## 4. Essential vs Voluntary Claims

### Concept

The `claims` parameter allows the RP to mark each requested claim as **essential** or **voluntary**:

- **Essential** (`"essential": true`): The authorization MUST fail if the claim cannot be provided. This is used when the RP cannot function without the data. Example: a banking app needs `email_verified: true` and cannot proceed without it.

- **Voluntary** (`"essential": false` or omitted): The claim is nice-to-have. The authorization proceeds even if the claim is unavailable. Example: a photo app requests `locale` for UI localization but works fine without it.

### Request Format

```json
{
  "id_token": {
    "email": {"essential": true},
    "acr": {"essential": true, "values": ["urn:mace:incommon:iap:silver"]},
    "name": null,
    "picture": null
  }
}
```

In this example:
- `email` is essential — auth fails if the user has no email.
- `acr` is essential with a required value — auth fails if the current authentication doesn't meet the silver level.
- `name` and `picture` are voluntary (null means "optional, no constraint").

### Go Code: Claim Essentiality Validation

```go
package oauth

import "fmt"

// ValidateEssentialClaims checks that all essential claims requested
// by the RP are present and satisfy their constraints.
// Returns an error listing all missing/violated essential claims.
func ValidateEssentialClaims(
	requested map[string]ClaimsRequestEntry,
	available map[string]any,
) error {
	var violations []string

	for claimName, entry := range requested {
		if !entry.Essential {
			continue // voluntary claim, skip
		}

		value, exists := available[claimName]
		if !exists {
			violations = append(violations,
				fmt.Sprintf("essential claim %q is missing", claimName))
			continue
		}

		// Check exact value constraint.
		if entry.Value != nil {
			if !valueMatchesConstraint(value, entry.Value) {
				violations = append(violations,
					fmt.Sprintf("essential claim %q does not match required value", claimName))
			}
			continue
		}

		// Check value set constraint (one of the allowed values).
		if len(entry.Values) > 0 {
			matched := false
			for _, allowed := range entry.Values {
				if valueMatchesConstraint(value, allowed) {
					matched = true
					break
				}
			}
			if !matched {
				violations = append(violations,
					fmt.Sprintf("essential claim %q is not in allowed values", claimName))
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("essential claim validation failed: %v", violations)
	}
	return nil
}

// valueMatchesConstraint compares a user claim value with a required value.
func valueMatchesConstraint(actual, required any) bool {
	// Handle common types: string, bool, float64 (JSON numbers).
	switch a := actual.(type) {
	case string:
		if r, ok := required.(string); ok {
			return a == r
		}
	case bool:
		if r, ok := required.(bool); ok {
			return a == r
		}
	case float64:
		if r, ok := required.(float64); ok {
			return a == r
		}
	}
	return false
}
```

### Enforcement Point

Essential claim validation occurs during the authorization request — before the ID token is issued. If validation fails, the server returns an error to the RP:

```go
// In CreateAuthorizationCode:
if claimsReq != nil && claimsReq.IDToken != nil {
	if err := ValidateEssentialClaims(claimsReq.IDToken, userClaims); err != nil {
		return "", errors.InvalidArgument("essential claims not satisfied: %v", err)
	}
}
```

---

## 5. Authentication Context Class (ACR)

### Concept

The `acr` (Authentication Context Class Reference) claim in the ID token tells the RP **how strongly** the user was authenticated. RPs can request a minimum ACR level via the `acr_values` authorization request parameter or via the `claims` parameter.

This enables **step-up authentication**: if the user authenticated with a password but the RP requires MFA, the server forces a second-factor challenge before issuing the token.

### ACR Values

ACR values are identifiers (typically URIs) that represent authentication strength levels. Common examples:

| ACR Value | Meaning |
|-----------|---------|
| `urn:mace:incommon:iap:silver` | InCommon Silver — multi-factor, identity-verified |
| `urn:mace:incommon:iap:bronze` | InCommon Bronze — single-factor |
| `phr` | 21 CFR Part 11 — healthcare, strong identity proofing |
| `phrh` | 21 CFR Part 11 high — highest healthcare assurance |
| `urn:ggid:1fa` | GGID custom — single-factor (password) |
| `urn:ggid:2fa` | GGID custom — multi-factor (password + TOTP) |
| `urn:ggid:webauthn` | GGID custom — WebAuthn hardware key |

### ACR Enforcement

The server enforces ACR in two ways:

1. **Request-time**: The RP includes `acr_values=urn:ggid:2fa` in the authorization request. If the user's current session doesn't meet this level, the server prompts for additional authentication (step-up).

2. **Token-time**: The `claims` parameter marks `acr` as essential with required values. The server checks the actual ACR achieved and fails if it doesn't match.

### Go Code: ACR Validation

```go
package oauth

import "fmt"

// ACRLevel represents a ranked authentication strength.
type ACRLevel int

const (
	ACRNone      ACRLevel = 0
	ACRSingleFactor ACRLevel = 1 // password only
	ACRMultiFactor  ACRLevel = 2 // password + OTP/Hardware key
	ACRStrongProof  ACRLevel = 3 // identity-verified + MFA
)

// ACRRegistry maps ACR string values to ranked levels.
// Higher rank = stronger authentication.
var ACRRegistry = map[string]ACRLevel{
	"urn:ggid:1fa":               ACRSingleFactor,
	"urn:ggid:2fa":               ACRMultiFactor,
	"urn:ggid:webauthn":          ACRMultiFactor,
	"urn:mace:incommon:iap:bronze": ACRSingleFactor,
	"urn:mace:incommon:iap:silver": ACRMultiFactor,
	"phr":                          ACRStrongProof,
}

// ResolveACR determines the ACR value for the current authentication.
// This is called after authentication completes.
func ResolveACR(authMethods []string) string {
	hasPassword := contains(authMethods, "pwd")
	hasOTP := contains(authMethods, "otp")
	hasWebAuthn := contains(authMethods, "hwk") || contains(authMethods, "swk")
	hasFace := contains(authMethods, "face") || contains(authMethods, "fpt")

	switch {
	case hasFace:
		return "urn:ggid:webauthn"
	case hasPassword && (hasOTP || hasWebAuthn):
		return "urn:ggid:2fa"
	case hasPassword:
		return "urn:ggid:1fa"
	default:
		return "urn:ggid:1fa"
	}
}

// EnforceACR checks whether the achieved ACR meets the RP's requirement.
// Returns an error if the authentication strength is insufficient.
func EnforceACR(achievedACR string, requiredACR string) error {
	if requiredACR == "" {
		return nil // no requirement
	}

	achieved, ok := ACRRegistry[achievedACR]
	if !ok {
		return fmt.Errorf("unknown ACR value: %s", achievedACR)
	}

	required, ok := ACRRegistry[requiredACR]
	if !ok {
		return fmt.Errorf("unknown required ACR value: %s", requiredACR)
	}

	if achieved < required {
		return fmt.Errorf(
			"insufficient authentication: achieved %s (level %d), required %s (level %d)",
			achievedACR, achieved, requiredACR, required,
		)
	}

	return nil
}

// NeedsStepUp determines whether the current session needs additional auth
// to meet the requested acr_values.
func NeedsStepUp(currentACR, requestedACR string) bool {
	return EnforceACR(currentACR, requestedACR) != nil
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
```

---

## 6. AMR (Authentication Methods References)

### Concept

The `amr` (Authentication Methods References) claim in the ID token is a JSON array of strings identifying **which authentication methods were actually used** during the current authentication. Unlike ACR (which is an abstract level), AMR is concrete: it tells the RP exactly what happened.

### AMR Values Registry

The AMR values registry (maintained by IANA) includes:

| AMR Value | Meaning |
|-----------|---------|
| `pwd` | Password-based authentication |
| `otp` | One-time password (TOTP, HOTP) |
| `hwk` | Hardware key (WebAuthn with hardware token) |
| `swk` | Software key (WebAuthn with platform authenticator) |
| `face` | Facial recognition |
| `fpt` | Fingerprint |
| `geo` | Geolocation |
| `kba` | Knowledge-based authentication |
| `mfa` | Multiple-factor authentication (generic indicator) |
| `pin` | PIN or pattern |
| `rba` | Risk-based authentication |
| `sms` | SMS OTP |
| `vad` | Voice analysis |
| `ret` | Retina scan |
| `iris` | Iris scan |

### Why AMR Matters for Step-Up Auth

When the RP requests step-up authentication via `acr_values`, the server needs to know what methods the user already used. The `amr` claim in the resulting ID token proves to the RP that the step-up actually happened. Without `amr`, the RP has no cryptographic proof of which factors were used — it must trust the server's `acr` claim blindly.

Example step-up flow:

1. User logs in with password → `amr: ["pwd"]`, `acr: "urn:ggid:1fa"`
2. RP requests `acr_values=urn:ggid:2fa`
3. Server prompts for OTP
4. User enters OTP → `amr: ["pwd", "otp"]`, `acr: "urn:ggid:2fa"`
5. RP verifies `amr` contains both `pwd` and `otp`

### Go Code: AMR Population

```go
package oauth

import "time"

// AuthSession tracks the methods used during a single authentication.
type AuthSession struct {
	UserID      string
	StartedAt   time.Time
	Methods     []string // ordered list of AMR values as they occur
	StepUpDepth int      // how many factors have been completed
}

// RecordMethod adds an authentication method to the session.
func (s *AuthSession) RecordMethod(amr string) {
	// Prevent duplicates.
	for _, m := range s.Methods {
		if m == amr {
			return
		}
	}
	s.Methods = append(s.Methods, amr)
	s.StepUpDepth++
}

// AMR returns the amr claim value for the ID token.
// If multiple factors were used, "mfa" is appended as a generic indicator.
func (s *AuthSession) AMR() []string {
	amr := make([]string, len(s.Methods))
	copy(amr, s.Methods)

	// Per RFC 8176: if 2+ factors were used, include "mfa".
	if s.StepUpDepth >= 2 {
		amr = append(amr, "mfa")
	}

	return amr
}

// AuthTime returns the Unix timestamp of the initial authentication.
func (s *AuthSession) AuthTime() int64 {
	return s.StartedAt.Unix()
}

// ToIDTokenOptions converts the session to IDTokenOptions for token issuance.
func (s *AuthSession) ToIDTokenOptions() *IDTokenOptions {
	acr := ResolveACR(s.Methods)
	return &IDTokenOptions{
		AMR:      s.AMR(),
		ACR:      acr,
		AuthTime: s.AuthTime(),
	}
}
```

### Integration with GGID Auth Service

The auth service records methods as they occur:

```go
// During login:
session := &AuthSession{UserID: userID, StartedAt: time.Now()}
session.RecordMethod("pwd")

// During MFA challenge:
session.RecordMethod("otp")

// When issuing ID token:
opts := session.ToIDTokenOptions()
idToken, err := s.issueIDToken(userID, tenantID, clientID, nonce, opts)
```

---

## 7. Privacy-Preserving Token Design

### Minimizing Data in JWT Access Tokens

Access tokens are presented to resource servers (APIs). A privacy-preserving access token contains **only the claims the resource server needs** to make authorization decisions. This contrasts with a "fat token" approach where all user profile data is embedded.

### Principles

1. **No PII in access tokens**: Access tokens are passed over the network to resource servers. They should not contain email, name, or other personally identifiable information. Use opaque identifiers or pairwise subs.

2. **Audience-scoped claims**: Different resource servers need different claims. An access token for the billing API might contain `billing_tier` while a token for the storage API contains `storage_quota`. Claims are scoped to the `aud` (audience).

3. **Scope-claim intersection**: Only include claims associated with the granted scopes. If the token has `scope: "read:files"`, it should not contain billing-related claims.

4. **Reference tokens for sensitive data**: For highly sensitive operations, use opaque reference tokens that are introspected server-side, so no data is exposed in the token itself.

### Go Code: Minimal-Claim Token Issuer

```go
package oauth

import (
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AudienceClaimPolicy defines which claims each audience (resource server) is allowed to see.
type AudienceClaimPolicy map[string][]string // audience -> allowed claims

// DefaultAudiencePolicy is a sensible default for GGID microservices.
var DefaultAudiencePolicy = AudienceClaimPolicy{
	"identity-service": {"sub", "tenant_id", "scope"},
	"policy-service":   {"sub", "tenant_id", "scope", "roles"},
	"org-service":      {"sub", "tenant_id", "scope", "org_id"},
	"audit-service":    {"sub", "tenant_id", "scope"},
}

// MinimalTokenIssuer creates access tokens with the minimum necessary claims.
type MinimalTokenIssuer struct {
	privateKey *rsa.PrivateKey
	issuer     string
	policy     AudienceClaimPolicy
}

func NewMinimalTokenIssuer(key *rsa.PrivateKey, issuer string, policy AudienceClaimPolicy) *MinimalTokenIssuer {
	return &MinimalTokenIssuer{
		privateKey: key,
		issuer:     issuer,
		policy:     policy,
	}
}

// IssueAccessToken creates a minimal-claim JWT access token.
func (i *MinimalTokenIssuer) IssueAccessToken(
	sub, tenantID, audience string,
	scopes []string,
	extraClaims map[string]any,
) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       i.issuer,
		"sub":       sub, // should be PPID, not canonical user ID
		"aud":       audience,
		"iat":       now.Unix(),
		"exp":       now.Add(15 * time.Minute).Unix(),
		"tenant_id": tenantID,
		"scope":     joinScopes(scopes),
	}

	// Apply audience-scoped claim filtering.
	allowedClaims, ok := i.policy[audience]
	if ok {
		// Only include extra claims that are in the allowed list for this audience.
		for claim, value := range extraClaims {
			if contains(allowedClaims, claim) {
				claims[claim] = value
			}
		}
	} else {
		// Unknown audience: no extra claims.
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(i.privateKey)
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}
```

### Token Data Minimization Patterns

| Pattern | Description | When to Use |
|---------|-------------|-------------|
| **Scope-filtered** | Token contains only claims for granted scopes | Default for most APIs |
| **Audience-scoped** | Token contains only claims the audience is authorized to see | Microservice architecture (GGID) |
| **Opaque reference** | Token is a random string; claims fetched via introspection | High-security, cross-org |
| **DPoP-bound** | Token bound to a key proof, not a bearer token | Public clients, mobile apps |

---

## 8. Consent-Gated Claim Release

### Per-Claim Consent

Privacy regulations (GDPR, CCPA) require that users explicitly consent to data sharing. In OIDC, this means each claim release should be gated by user consent. The user sees a consent screen listing exactly which data the RP is requesting, and approves or denies each item.

### Consent Flow

1. User authenticates to GGID.
2. GGID shows consent screen: "Calendar App wants to access: email, name. Approve?"
3. User approves specific claims (or all, or denies).
4. GGID stores consent per (user, client, claim).
5. On subsequent auth requests, GGID checks stored consent:
   - If consent exists and is valid → release claims without prompting.
   - If consent is missing for a new claim → prompt for consent.
   - If consent was revoked → deny the claim.

### Consent Storage

Consent records are stored per (user_id, client_id, claim_name):

```sql
CREATE TABLE oidc_claim_consents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES identity.users(id),
    client_id   UUID NOT NULL,
    tenant_id   UUID NOT NULL,
    claim_name  VARCHAR(255) NOT NULL,
    granted     BOOLEAN NOT NULL DEFAULT FALSE,
    granted_at  TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    UNIQUE(user_id, client_id, claim_name)
);
```

### Go Code: Consent-Gated Claim Engine

```go
package oauth

import (
	"context"
	"time"
)

// ConsentRecord stores a user's consent decision for a specific claim.
type ConsentRecord struct {
	UserID    string
	ClientID  string
	TenantID  string
	ClaimName string
	Granted   bool
	GrantedAt time.Time
	RevokedAt *time.Time
	ExpiresAt *time.Time
}

// ConsentStore is the interface for consent persistence.
type ConsentStore interface {
	Get(ctx context.Context, userID, clientID, claimName string) (*ConsentRecord, error)
	Save(ctx context.Context, record *ConsentRecord) error
	Revoke(ctx context.Context, userID, clientID, claimName string) error
}

// ConsentClaimEngine gates claim release based on stored user consent.
type ConsentClaimEngine struct {
	store ConsentStore
}

func NewConsentClaimEngine(store ConsentStore) *ConsentClaimEngine {
	return &ConsentClaimEngine{store: store}
}

// FilterByConsent removes claims that the user has not consented to share.
// It returns the filtered claim map and a list of claims needing fresh consent.
func (e *ConsentClaimEngine) FilterByConsent(
	ctx context.Context,
	claims map[string]any,
	userID, clientID, tenantID string,
) (map[string]any, []string, error) {
	result := make(map[string]any)
	var needsConsent []string

	// "sub" is always allowed — it's the minimum required claim.
	if sub, ok := claims["sub"]; ok {
		result["sub"] = sub
	}

	for claimName, value := range claims {
		if claimName == "sub" {
			continue
		}

		record, err := e.store.Get(ctx, userID, clientID, claimName)
		if err != nil {
			// On error, treat as needing consent (fail closed).
			needsConsent = append(needsConsent, claimName)
			continue
		}

		if record == nil {
			// No consent record — needs fresh consent.
			needsConsent = append(needsConsent, claimName)
			continue
		}

		if !record.Granted || isRevoked(record) || isExpired(record) {
			// Consent denied or expired — needs fresh consent.
			needsConsent = append(needsConsent, claimName)
			continue
		}

		// Consent is valid — include the claim.
		result[claimName] = value
	}

	return result, needsConsent, nil
}

// GrantConsent records the user's consent decision for a claim.
func (e *ConsentClaimEngine) GrantConsent(
	ctx context.Context,
	userID, clientID, tenantID, claimName string,
	granted bool,
) error {
	record := &ConsentRecord{
		UserID:    userID,
		ClientID:  clientID,
		TenantID:  tenantID,
		ClaimName: claimName,
		Granted:   granted,
		GrantedAt: time.Now(),
	}
	// Set a 90-day expiry on consent.
	expiry := time.Now().Add(90 * 24 * time.Hour)
	record.ExpiresAt = &expiry

	return e.store.Save(ctx, record)
}

func isRevoked(r *ConsentRecord) bool {
	return r.RevokedAt != nil
}

func isExpired(r *ConsentRecord) bool {
	return r.ExpiresAt != nil && time.Now().After(*r.ExpiresAt)
}
```

### Consent Screen Integration

The authorization server pauses the flow when consent is needed:

```go
claims, needsConsent, err := consentEngine.FilterByConsent(ctx, requestedClaims, userID, clientID, tenantID)
if err != nil {
	return "", err
}
if len(needsConsent) > 0 {
	// Render consent screen, then resume.
	return s.redirectToConsentScreen(authorizationRequestID, needsConsent), nil
}
// All claims have consent — proceed to issue tokens.
```

---

## 9. GGID Privacy-Preserving Auth Gap Analysis

### Review of Current Implementation

The following analysis is based on a review of `/Users/zhanju/ggai/ggid/services/oauth/` source files.

#### What Exists

| Feature | Status | Location |
|---------|--------|----------|
| **ID Token issuance** | Implemented | `oauth_service.go:453` — `issueIDToken()` |
| **ACR/AMR struct** | Partially implemented | `oauth_service.go:447` — `IDTokenOptions` has `AMR`, `ACR`, `AuthTime` fields |
| **ACR/AMR in token** | Structurally present | `oauth_service.go:468-477` — conditionally added if `opts != nil` |
| **UserInfo endpoint** | Implemented | `oauth_service.go:522` — `GetUserInfo()` |
| **Discovery metadata** | Implemented | `oauth_service.go:365` — `GetDiscoveryConfig()` |
| **Claims supported list** | Implemented | `oauth_service.go:379` — lists `sub`, `email`, `name`, `picture`, `groups`, `preferred_username`, `updated_at` |
| **Subject types** | Declared as `["public"]` only | `oauth_service.go:376` |

#### What Is Missing

| Feature | Gap | Impact |
|---------|-----|--------|
| **Pairwise Subject Identifiers (PPID)** | No PPID generator. `sub` is always the raw user UUID. Discovery only advertises `"public"`. | Any RP can see the canonical user UUID. Two RPs can correlate users by comparing `sub` values. |
| **Claims parameter parsing** | No parsing of the `claims` authorization request parameter. `AuthorizeRequest` struct (line 190) has no `Claims` field. | RPs cannot request specific claims. All profile data is released regardless of need. |
| **Selective disclosure engine** | No scope-to-claim mapping or claim filtering. ID tokens contain a fixed set: `iss`, `sub`, `aud`, `iat`, `exp`, `nonce`, `tenant_id`. | Over-sharing: every RP gets the same claims regardless of what it requested. |
| **ACR/AMR population** | `issueIDToken` is called with `opts = nil` at line 351. The `IDTokenOptions` struct exists but is never populated. | ACR and AMR claims are never included in ID tokens, even though the code path exists. |
| **ACR enforcement** | No `acr_values` parameter handling. No ACR registry. No step-up authentication trigger. | RPs cannot require MFA or specific auth strength. The `acr` claim is absent from all issued tokens. |
| **Consent-gated release** | No consent store, no consent screen integration, no per-claim consent records. | Claims are released without user consent. No GDPR/CCPA compliance for claim sharing. |
| **Audience-scoped tokens** | Access tokens contain `sub`, `tenant_id`, `scope` — but no audience-based claim filtering. | All resource servers receive the same token data regardless of which API they serve. |
| **UserInfo selective claims** | `GetUserInfo()` returns a fixed struct (`sub`, `name`, `email`, `email_verified`, `picture`, `tenant_id`) regardless of scope. | UserInfo always returns all profile data, ignoring scope restrictions. |

#### Detailed Findings

**1. ID Token contains raw user UUID as `sub`**

```go
// oauth_service.go:459
"sub": userID.String(),
```

This means every RP sees the canonical user UUID, enabling cross-client correlation. There is no pairwise mode.

**2. `issueIDToken` always receives `nil` for opts**

```go
// oauth_service.go:351
idToken, err := s.issueIDToken(code.UserID, code.TenantID, client.ClientID, code.Nonce, nil)
```

The `IDTokenOptions` struct with `AMR` and `ACR` fields exists but is never used. The auth service does not pass authentication method information to the OAuth service.

**3. UserInfo ignores scopes**

```go
// oauth_service.go:528-535
resp := &UserInfoResponse{
    Sub:           getStringClaim(claims, "sub"),
    Name:          getStringClaim(claims, "name"),
    Email:         getStringClaim(claims, "email"),
    EmailVerified: getBoolClaim(claims, "email_verified"),
    Picture:       getStringClaim(claims, "picture"),
    TenantID:      getStringClaim(claims, "tenant_id"),
}
```

All fields are populated unconditionally. An RP with only `openid` scope receives the same data as one with `profile email` scope.

**4. No claims parameter in AuthorizeRequest**

```go
// oauth_service.go:190-199
type AuthorizeRequest struct {
    TenantID           uuid.UUID
    ClientID           string
    RedirectURI        string
    ResponseType       string
    Scope              []string
    State              string
    Nonce              string
    CodeChallenge      string
    CodeChallengeMethod string
    UserID             uuid.UUID
}
```

There is no `Claims` field, so the OIDC `claims` parameter is silently ignored.

---

## 10. Gap Analysis & Recommendations

### Priority Matrix

| # | Action Item | Priority | Effort | Impact |
|---|-------------|----------|--------|--------|
| 1 | Implement PPID support | P1 | Medium (3-5 days) | High — eliminates cross-client correlation |
| 2 | Parse `claims` parameter + selective disclosure | P1 | Medium (3-5 days) | High — enables minimal claim release |
| 3 | Populate ACR/AMR in ID tokens | P1 | Small (1-2 days) | High — enables step-up auth |
| 4 | Implement consent-gated claim release | P2 | Large (5-8 days) | Medium — GDPR/CCPA compliance |
| 5 | Add audience-scoped access tokens | P2 | Medium (3-5 days) | Medium — reduces token data exposure |

### Detailed Action Items

#### 1. Implement PPID Support (P1, 3-5 days)

- Add `SubjectType` and `SectorIdentifierURI` fields to the `Client` domain model.
- Implement `PPIDGenerator` with HMAC-SHA-256 and per-tenant salt.
- Update `issueIDToken` to use `resolveSubject()` which selects public or pairwise based on client config.
- Update discovery metadata to advertise `["public", "pairwise"]`.
- Store per-tenant salts in configuration (minimum 32 bytes each).
- Add migration to add `subject_type` and `sector_identifier_uri` columns to the `oauth_clients` table.

#### 2. Parse `claims` Parameter + Selective Disclosure (P1, 3-5 days)

- Add `Claims` field to `AuthorizeRequest`.
- Implement `ParseClaimsRequest()` to parse the JSON claims parameter.
- Implement `ResolveClaims()` engine to compute the claim set per scope + claims parameter.
- Integrate into `issueIDToken` to only include resolved claims.
- Update `GetUserInfo` to filter claims by scope.
- Add `ClaimsParameterSupported: true` to discovery metadata.

#### 3. Populate ACR/AMR in ID Tokens (P1, 1-2 days)

- Wire the auth service to pass `AuthSession` data to `issueIDToken`.
- Change line 351 from `nil` to a populated `IDTokenOptions`.
- Implement `ResolveACR()` to map authentication methods to ACR values.
- Add `acr_values` parameter handling in `AuthorizeRequest`.
- Implement `EnforceACR()` for step-up authentication triggers.
- Register GGID-specific ACR values (`urn:ggid:1fa`, `urn:ggid:2fa`, `urn:ggid:webauthn`).

#### 4. Implement Consent-Gated Claim Release (P2, 5-8 days)

- Create `oidc_claim_consents` table migration.
- Implement `ConsentStore` interface with PostgreSQL backing.
- Implement `ConsentClaimEngine` for filtering claims by stored consent.
- Add consent screen UI to the admin console (Next.js).
- Integrate consent check into the authorization flow (between auth and token issuance).
- Add consent revocation endpoint for users.

#### 5. Add Audience-Scoped Access Tokens (P2, 3-5 days)

- Define `AudienceClaimPolicy` mapping each GGID microservice to its allowed claims.
- Implement `MinimalTokenIssuer` that filters claims by audience.
- Replace the current `issueAccessToken` to use the minimal issuer.
- Add `resource` parameter support (RFC 8707) to allow RPs to request audience-scoped tokens.
- Document the per-service claim policy in the GGID configuration.

### Summary

GGID's OAuth service has solid foundational support for OIDC (authorization code flow, PKCE, PAR, JAR, backchannel logout, DPoP) but lacks the privacy-preserving layer that RFC 9701 and OIDC Core Sections 5.5/8.1 define. The most impactful improvements are PPID support (prevents user correlation), selective disclosure (prevents over-sharing), and ACR/AMR population (enables step-up auth). These three items can be completed in approximately 7-12 developer-days and would bring GGID from basic OIDC compliance to privacy-preserving OIDC compliance.

---

*Document version: 1.0 | Last updated: 2025 | Author: GGID Security Research*
