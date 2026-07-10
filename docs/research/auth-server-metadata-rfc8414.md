# OAuth 2.0 Authorization Server Metadata (RFC 8414)

## 1. Overview

RFC 8414 defines a standardized metadata format that OAuth 2.0 clients use to
discover authorization server (AS) endpoints and capabilities — without hardcoding
URLs or manually configuring each client.

**The problem:** OAuth 2.0 originally specified endpoint paths (authorize, token,
introspect, revocation) but provided no mechanism for clients to discover them
automatically. Every client had to be configured manually with the AS's exact
endpoint URLs, JWKS location, supported grant types, and authentication methods.
This is fragile — any endpoint change breaks all clients.

**The solution:** RFC 8414 introduces `GET /.well-known/oauth-authorization-server`,
returning a JSON document with the AS's issuer URL, endpoint locations, supported
grant types, response types, signing algorithms, and capability flags.

**Relationship to OIDC Discovery:** OpenID Connect defined
`/.well-known/openid-configuration` for RP discovery. RFC 8414 serves non-OIDC
OAuth 2.0 deployments that need endpoint discovery but not OIDC-specific fields
(`userinfo_endpoint`, `id_token_signing_alg_values`). OIDC Discovery is for RPs;
RFC 8414 is for pure OAuth 2.0 clients (API gateways, M2M).

---

## 2. Metadata Document Format

### Endpoint

```
GET /.well-known/oauth-authorization-server
```

- No authentication required (public discovery)
- Returns `application/json`; `Cache-Control: max-age` recommended

### Required Metadata Fields

| Field | Description | Example |
|-------|-------------|---------|
| `issuer` | AS issuer URL (verified against fetch URL) | `https://auth.example.com` |
| `authorization_endpoint` | Authorization endpoint URL | `https://auth.example.com/oauth/authorize` |
| `token_endpoint` | Token endpoint URL | `https://auth.example.com/oauth/token` |
| `response_types_supported` | Supported response types | `["code", "token"]` |
| `grant_types_supported` | Supported grant types | `["authorization_code", "client_credentials", "refresh_token"]` |
| `subject_types_supported` | Subject identifier types | `["public", "pairwise"]` |

### Optional Metadata Fields

| Field | Description |
|-------|-------------|
| `introspection_endpoint` | RFC 7662 token introspection URL |
| `revocation_endpoint` | RFC 7009 token revocation URL |
| `jwks_uri` | JWKS endpoint for signature verification |
| `scopes_supported` | Scopes the AS supports |
| `token_endpoint_auth_methods_supported` | Client auth methods at token endpoint |
| `code_challenge_methods_supported` | PKCE challenge methods (`S256`, `plain`) |
| `registration_endpoint` | RFC 7591 dynamic client registration URL |
| `service_documentation` | URL to human-readable docs |
| `op_policy_uri` | Privacy policy URL |
| `op_tos_uri` | Terms of service URL |

### RFC 8707 Extension (Resource Indicators)

```json
"resource_indicators_supported": true
```

Set to `true` when the AS supports RFC 8707 resource indicators — allowing
clients to specify a target resource (audience) in authorization and token
requests.

### RFC 9126 Extension (Pushed Authorization Requests)

```json
"pushed_authorization_request_endpoint": "https://auth.example.com/oauth/par",
"require_pushed_authorization_requests": false
```

When PAR is supported, the metadata advertises the PAR endpoint URL and whether
it is mandatory for all authorization requests.

### Full JSON Example (GGID)

```json
{
  "issuer": "https://auth.ggid.io",
  "authorization_endpoint": "https://auth.ggid.io/oauth/authorize",
  "token_endpoint": "https://auth.ggid.io/oauth/token",
  "introspection_endpoint": "https://auth.ggid.io/oauth/introspect",
  "revocation_endpoint": "https://auth.ggid.io/oauth/revoke",
  "jwks_uri": "https://auth.ggid.io/oauth/jwks",
  "registration_endpoint": "https://auth.ggid.io/oauth/register",
  "pushed_authorization_request_endpoint": "https://auth.ggid.io/oauth/par",
  "require_pushed_authorization_requests": false,
  "response_types_supported": ["code", "token", "id_token"],
  "grant_types_supported": [
    "authorization_code",
    "client_credentials",
    "refresh_token",
    "urn:ietf:params:oauth:grant-type:device_code",
    "urn:ietf:params:oauth:grant-type:jwt-bearer"
  ],
  "subject_types_supported": ["public"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "client_secret_post",
    "none"
  ],
  "code_challenge_methods_supported": ["S256", "plain"],
  "resource_indicators_supported": true,
  "service_documentation": "https://docs.ggid.io/oauth"
}
```

---

## 3. OIDC Discovery vs RFC 8414

### OIDC Discovery (`/.well-known/openid-configuration`)

- Defined by OpenID Connect Discovery 1.0; scope is OIDC-specific
- Includes `userinfo_endpoint`, `id_token_signing_alg_values_supported`, `claims_supported`
- Required for OIDC RPs; only OIDC providers serve this

### RFC 8414 (`/.well-known/oauth-authorization-server`)

- Defined by IETF RFC 8414; scope is generic OAuth 2.0 (no OIDC fields)
- Required for pure OAuth 2.0 clients (API gateways, M2M); any AS can serve this

### Overlap and Strategy

Most metadata fields overlap between the two. OIDC Discovery is effectively a
superset — it adds `userinfo_endpoint`, `id_token_signing_alg_values_supported`,
and `claims_supported` on top of the RFC 8414 endpoint and capability fields.
RFC 8414 adds nothing beyond what OIDC Discovery serves but exists for AS
deployments that are not OIDC providers.

**Recommended strategy for GGID:** Serve BOTH endpoints. Share a common
metadata builder and add OIDC-specific fields only in the `openid-configuration`
response.

### Comparison Table

| Field | OIDC Discovery | RFC 8414 | GGID Currently Serves |
|-------|:-:|:-:|:-:|
| `issuer` | Yes | Yes (required) | Yes |
| `authorization_endpoint` | Yes | Yes (required) | Yes |
| `token_endpoint` | Yes | Yes (required) | Yes |
| `userinfo_endpoint` | Yes | No | Yes |
| `jwks_uri` | Yes | Yes | Yes |
| `introspection_endpoint` | No | Yes | Yes (in OIDC) |
| `revocation_endpoint` | No | Yes | Yes (in OIDC) |
| `response_types_supported` | Yes | Yes (required) | Yes |
| `grant_types_supported` | Yes | Yes (required) | Yes |
| `subject_types_supported` | Yes | Yes (required) | Yes |
| `scopes_supported` | Yes | Yes | Yes |
| `claims_supported` | Yes | No | Yes |
| `id_token_signing_alg_values_supported` | Yes | No | Yes |
| `code_challenge_methods_supported` | Yes | Yes | Yes |
| `registration_endpoint` | No | Yes | No |
| `pushed_authorization_request_endpoint` | No | Yes (RFC 9126) | No |
| `resource_indicators_supported` | No | Yes (RFC 8707) | No |

---

## 4. Metadata Signing

### Signed Metadata (draft-ietf-oauth-metadata-signed)

An IETF draft proposes signing the AS metadata document as a JWT. The purpose
is to prevent metadata tampering and cryptographically prove the AS identity —
clients can verify the signature against the AS's published JWKS.

**How it would work:** The metadata endpoint returns a JWT (not plain JSON)
whose payload contains the standard metadata fields. The JWT header references
a key ID; the client fetches the JWKS and verifies the signature.

This is still at the draft stage and not widely deployed.

### Current Practice

- Most AS serve **unsigned JSON** metadata
- Security relies on HTTPS (TLS proves server identity)
- Clients validate: `metadata.issuer == URL they fetched from` (RFC 8414 Section 5)
- This prevents redirect and spoofing attacks

**GGID recommendation:** Serve unsigned JSON. Monitor the signing standard.

---

## 5. Issuer Discovery

### WebFinger (RFC 7033)

When a client knows a user's identifier (e.g., email `user@example.com`) but
not the AS issuer, WebFinger provides issuer discovery:

```
GET https://example.com/.well-known/webfinger?resource=acct:user@example.com
```

Response:
```json
{
  "subject": "acct:user@example.com",
  "links": [
    {
      "rel": "http://openid.net/specs/connect/1.0/issuer",
      "href": "https://auth.example.com"
    }
  ]
}
```

The client then fetches `https://auth.example.com/.well-known/oauth-authorization-server`
(or `openid-configuration`) to get full metadata.

### Issuer-Based Discovery

When the client already knows the issuer URL (from config, client registration,
or WebFinger):

1. Append `/.well-known/oauth-authorization-server` to the issuer URL
2. Fetch the metadata document over HTTPS
3. **Validate:** `metadata.issuer == expected_issuer` (prevent redirect attacks)
4. Use the discovered endpoint URLs for all subsequent requests

### GGID Multi-Tenant Discovery Flow

GGID supports multi-tenancy, so the issuer can be tenant-specific:

```
Tenant ID → Issuer URL → Metadata → Endpoints

acme-corp → https://auth.ggid.io/acme-corp
         → https://auth.ggid.io/acme-corp/.well-known/oauth-authorization-server
         → Per-tenant scopes, features, PAR settings
```

Currently GGID uses a single issuer for all tenants — not tenant-specific.

---

## 6. GGID Current Discovery

### Code Analysis

Examining `server.go` and `oauth_service.go`:

| Endpoint | Served? | Route |
|----------|---------|-------|
| `/.well-known/openid-configuration` | **Yes** | Line 145 — calls `oauthSvc.GetDiscoveryConfig()` |
| `/.well-known/oauth-authorization-server` | **No** | Not implemented |
| `/.well-known/webfinger` | No | Not implemented |

### Current OIDC Discovery Fields (GetDiscoveryConfig)

GGID's `OIDCDiscoveryConfig` struct (domain/models.go:156) already includes many
RFC 8414 fields:

| Field | Current Value | RFC 8414 Required? |
|-------|---------------|:-:|
| `issuer` | `s.issuer` (from config) | Yes |
| `authorization_endpoint` | `{issuer}/oauth/authorize` | Yes |
| `token_endpoint` | `{issuer}/oauth/token` | Yes |
| `userinfo_endpoint` | `{issuer}/oauth/userinfo` | No (OIDC only) |
| `jwks_uri` | `{issuer}/oauth/jwks` | No |
| `revocation_endpoint` | `{issuer}/oauth/revoke` | No |
| `introspection_endpoint` | `{issuer}/oauth/introspect` | No |
| `response_types_supported` | `["code", "token", "id_token"]` | Yes |
| `grant_types_supported` | `["authorization_code", "refresh_token", "client_credentials"]` | Yes |
| `subject_types_supported` | `["public"]` | Yes |
| `scopes_supported` | `["openid", "profile", "email", "offline_access"]` | No |
| `claims_supported` | `["sub", "email", "name", ...]` | No (OIDC only) |
| `token_endpoint_auth_methods_supported` | `["client_secret_basic", "client_secret_post", "none", "tls_client_auth", ...]` | No |
| `code_challenge_methods_supported` | `["S256", "plain"]` | No |

### Missing from GGID Discovery

| Field | Impact |
|-------|--------|
| `registration_endpoint` | DCR exists at `/oauth/register` but not advertised |
| `pushed_authorization_request_endpoint` | PAR not implemented |
| `resource_indicators_supported` | RFC 8707 not advertised |
| RFC 8414 endpoint | `/.well-known/oauth-authorization-server` not served |
| Per-tenant issuer | Single issuer for all tenants |

### Tenant Specificity

**No.** GGID's discovery uses a single `s.issuer` value configured at startup.
All tenants see the same metadata. The `X-Tenant-ID` header is used at the
authorize and token endpoints but not reflected in discovery metadata.

---

## 7. Implementation Design

### New Endpoint

```
GET /.well-known/oauth-authorization-server  →  RFC 8414 metadata
GET /.well-known/openid-configuration         →  OIDC metadata (existing)
```

### Go Code

Add a route in `buildHandler` and a new service method:

```go
// In buildHandler (server.go):
mux.HandleFunc("/.well-known/oauth-authorization-server",
    func(w http.ResponseWriter, r *http.Request) {
        metadata := oauthSvc.GetASMetadata()
        writeJSON(w, http.StatusOK, metadata)
    })

// In oauth_service.go:
// ASMetadata is the RFC 8414 response (no OIDC-specific fields).
type ASMetadata struct {
    Issuer                            string   `json:"issuer"`
    AuthorizationEndpoint             string   `json:"authorization_endpoint"`
    TokenEndpoint                     string   `json:"token_endpoint"`
    IntrospectionEndpoint             string   `json:"introspection_endpoint,omitempty"`
    RevocationEndpoint                string   `json:"revocation_endpoint,omitempty"`
    JwksURI                           string   `json:"jwks_uri,omitempty"`
    RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
    PushedAuthorizationRequestEndpoint string  `json:"pushed_authorization_request_endpoint,omitempty"`
    RequirePushedAuthorizationRequests bool    `json:"require_pushed_authorization_requests,omitempty"`
    ResponseTypesSupported            []string `json:"response_types_supported"`
    GrantTypesSupported               []string `json:"grant_types_supported"`
    SubjectTypesSupported             []string `json:"subject_types_supported"`
    ScopesSupported                   []string `json:"scopes_supported,omitempty"`
    TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
    CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
    ResourceIndicatorsSupported       bool     `json:"resource_indicators_supported,omitempty"`
    ServiceDocumentation              string   `json:"service_documentation,omitempty"`
}

func (s *OAuthService) GetASMetadata() *ASMetadata {
    base := s.issuer
    return &ASMetadata{
        Issuer:                            s.issuer,
        AuthorizationEndpoint:             base + "/oauth/authorize",
        TokenEndpoint:                     base + "/oauth/token",
        IntrospectionEndpoint:             base + "/oauth/introspect",
        RevocationEndpoint:                base + "/oauth/revoke",
        JwksURI:                           base + "/oauth/jwks",
        RegistrationEndpoint:              base + "/oauth/register",
        ResponseTypesSupported:            []string{"code", "token", "id_token"},
        GrantTypesSupported:               []string{"authorization_code", "refresh_token",
            "client_credentials",
            "urn:ietf:params:oauth:grant-type:device_code",
            "urn:ietf:params:oauth:grant-type:jwt-bearer"},
        SubjectTypesSupported:             []string{"public"},
        ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
        TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
        CodeChallengeMethodsSupported:     []string{"S256", "plain"},
        ServiceDocumentation:              "https://docs.ggid.io/oauth",
    }
}
```

### Per-Tenant Metadata

For multi-tenancy, the issuer becomes `https://auth.ggid.io/{tenant_id}` and
metadata is generated per-tenant:

```go
func (s *Server) handleASMetadata(w http.ResponseWriter, r *http.Request) {
    tenantID := r.URL.Path[len("/.well-known/oauth-authorization-server"):]
    // Or: parse tenant from path if using /{tenant}/.well-known/...
    metadata := s.oauthSvc.GetASMetadataForTenant(tenantID)
    json.NewEncoder(w).Encode(metadata)
}
```

Per-tenant differences may include: enabled grant types, supported scopes,
PAR requirement, resource indicator support, allowed auth methods.

---

## 8. Roadmap

| Phase | Task | Effort | Priority |
|-------|------|--------|----------|
| 1 | Verify OIDC discovery completeness (add `registration_endpoint`) | 0.5 day | High |
| 2 | Add `/.well-known/oauth-authorization-server` endpoint | 1 day | High |
| 3 | Per-tenant metadata (tenant-specific issuer, scopes, features) | 2 days | Medium |
| 4 | WebFinger (`/.well-known/webfinger`) for email-based issuer discovery | 1 day | Low |
| 5 | Signed metadata (when draft-ietf-oauth-metadata-signed matures) | TBD | Future |

**Phase 1-2** (~1.5 days) closes the RFC 8414 gap — metadata fields already
exist in `GetDiscoveryConfig`; Phase 2 exposes them at a second well-known URL.

**Phase 3** enables multi-tenant isolation at the discovery layer (per-tenant
scopes, PAR enforcement). **Phase 4-5** are nice-to-haves deferred until demand.
