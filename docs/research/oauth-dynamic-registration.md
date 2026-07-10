# OAuth 2.0 Dynamic Client Registration (RFC 7591 / 7592)

> Research document for GGID OAuth service enhancement.
> Status: RFC 7591 partially implemented; RFC 7592 not yet implemented.

---

## 1. Overview

**RFC 7591** defines the OAuth 2.0 Dynamic Client Registration Protocol, allowing
clients to register themselves with an Authorization Server (AS) programmatically.
**RFC 7592** extends this with a management protocol for reading, updating, and
deleting registered clients after initial creation.

**Problem solved**: Without dynamic registration, each OAuth client requires an
administrator to manually create a `client_id` and `client_secret` — a bottleneck
for platforms with many integrations.

**Use cases**:
- **Multi-tenant SaaS**: each tenant self-registers its internal apps
- **Developer portals**: developers get API credentials without ops involvement
- **Federation**: partner organizations register SPs automatically
- **OID4VCI / wallet ecosystems**: credential wallets register dynamically

**Contrast with static registration**:

| Aspect | Static | Dynamic (RFC 7591) |
|--------|--------|---------------------|
| Onboarding | Manual admin form | `POST /oauth/register` |
| Time to live credentials | Hours/days | Seconds |
| Self-service | No | Yes |
| Audit trail | Config change log | Registration event |
| Scaling | Linear with ops headcount | Unlimited |

---

## 2. RFC 7591 — Registration Protocol

### Register Endpoint

```
POST /oauth/register
Content-Type: application/json

{
  "client_name": "My SPA App",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid profile email",
  "logo_uri": "https://app.example.com/logo.png",
  "policy_uri": "https://app.example.com/privacy",
  "tos_uri": "https://app.example.com/terms",
  "contacts": ["admin@example.com"],
  "software_id": "my-app-v1",
  "software_version": "2.0.0"
}
```

Response (201 Created):

```json
{
  "client_id": "gcid_aBcD1234...",
  "client_secret": "gcs_xYzW5678...",
  "client_id_issued_at": 1700000000,
  "client_secret_expires_at": 0,
  "client_name": "My SPA App",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic"
}
```

The AS validates the metadata, creates the client, and returns credentials.
If the client is confidential, a `client_secret` is returned **once**.
A `registration_access_token` (RFC 7592) may also be issued for management.

### Client Metadata Fields

| Field | Required | Description |
|-------|----------|-------------|
| `redirect_uris` | Yes | Allowed callback URLs (exact match) |
| `grant_types` | No | `authorization_code`, `client_credentials`, `refresh_token`, `urn:ietf:params:oauth:grant-type:device_code` |
| `response_types` | No | `code`, `token` (deprecated), `id_token`, `code id_token` |
| `token_endpoint_auth_method` | No | `client_secret_basic`, `client_secret_post`, `private_key_jwt`, `none` |
| `scope` | No | Space-delimited requested scopes |
| `client_name` | No | Human-readable name |
| `client_uri` | No | Client homepage URL |
| `logo_uri` | No | Logo image URL |
| `jwks_uri` / `jwks` | No | Client public keys (for `private_key_jwt`) |
| `contacts` | No | Admin email addresses |
| `software_id` | No | Stable UUID for distributed software |
| `software_version` | No | Version of the software |
| `policy_uri` | No | Privacy policy URL |
| `tos_uri` | No | Terms of service URL |

### Security Considerations

- **Open vs protected registration**: Open allows anonymous registration (abuse risk);
  protected requires an initial access token (recommended for production)
- **Rate limiting**: per-tenant and per-IP limits on the register endpoint
- **Redirect URI validation**: exact match only — no wildcards; HTTPS required in production
- **Secret entropy**: `client_secret` must be high-entropy (32+ bytes random), hashed at rest
- **Scope restriction**: AS may clamp requested scopes to tenant-allowed set

---

## 3. RFC 7592 — Management Protocol

### Operations

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/oauth/register/{client_id}` | Read client metadata |
| `PUT` | `/oauth/register/{client_id}` | Update metadata (full replace) |
| `DELETE` | `/oauth/register/{client_id}` | Delete client + revoke all tokens |
| `POST` | `/oauth/register/{client_id}/rotate` | Rotate client_secret (extension) |

### Authentication

All management operations require a **registration access token** — a bearer token
issued at registration time (RFC 7592 Section 2). Key properties:

- Valid **only** for managing that specific client (scoped)
- **Cannot** be used for OAuth flows (separate from access tokens)
- Returned in the initial `POST /oauth/register` response
- On `PUT`, the AS **must** issue a new registration access token (old one invalidated)

```
GET /oauth/register/gcid_aBcD1234...
Authorization: Bearer <registration_access_token>
```

### Use Cases

- **Self-service**: update redirect URIs after deployment without admin help
- **Secret rotation**: periodic client_secret rotation for compliance
- **Decommissioning**: client deletes itself, revoking all outstanding tokens
- **Metadata sync**: update logo, policy, or contact info dynamically

---

## 4. GGID Integration

### Current State

GGID **already implements** RFC 7591 registration (partial):

- `POST /oauth/register` — registered in `server.go` (line 577)
- `POST /api/v1/oauth/register` — alias route (line 526)
- `DynamicRegistrationRequest` struct with standard metadata fields (line 878)
- `DynamicRegistrationResponse` with `client_id`, `client_secret`, timestamps (line 896)
- `DynamicClientRegister()` method (line 912) — creates client, hashes secret with Argon2id
- Tenant-scoped via `tenant.FromContext(ctx)` — clients tied to `tenant_id`

**Existing key generation** (already in production):

```go
// oauth_service.go line 795
func generateClientID() string {
    id, _ := crypto.GenerateRandomToken(16)
    return "gcid_" + id   // ~43 chars base64url
}

func generateClientSecret() string {
    secret, _ := crypto.GenerateRandomToken(32)
    return "gcs_" + secret  // ~47 chars base64url
}
```

**Existing `OAuthClient` domain model** (`domain/models.go` line 28):

```go
type OAuthClient struct {
    ID                      uuid.UUID
    TenantID                uuid.UUID
    ClientID                string          // public identifier
    ClientSecretHash        string          // Argon2id hash
    Name                    string
    Type                    ClientType      // confidential | public
    GrantTypes              []string
    ResponseTypes           []string
    RedirectURIs            []string
    Scopes                  []string
    TokenEndpointAuthMethod string
    Metadata                map[string]any
    RequirePKCE             bool
    Enabled                 bool
    CreatedAt               time.Time
    UpdatedAt               time.Time
}
```

### What's Missing

| Feature | RFC | Status |
|---------|-----|--------|
| Registration (`POST`) | 7591 | **Done** |
| Registration access token | 7592 | Not implemented |
| Read (`GET /{id}`) | 7592 | Not implemented |
| Update (`PUT /{id}`) | 7592 | Not implemented |
| Delete (`DELETE /{id}`) | 7592 | Not implemented |
| Secret rotation | 7592 ext | Not implemented |
| Initial access token | 7591 §3 | Not implemented |
| `contacts` field | 7591 §2 | Not in request struct |

### Proposed RFC 7592 Handler

```go
// New fields to add to DynamicRegistrationResponse
RegistrationAccessToken string `json:"registration_access_token,omitempty"`

// New management endpoints in server.go
mux.HandleFunc("/oauth/register/", func(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/oauth/register/"), "/")
    clientID := parts[0]

    // Validate registration access token
    rat := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    if !oauthSvc.ValidateRegistrationToken(r.Context(), clientID, rat) {
        writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
        return
    }

    switch r.Method {
    case http.MethodGet:
        // Return full client metadata
    case http.MethodPut:
        // Update metadata, issue new registration access token
    case http.MethodDelete:
        // Delete client, revoke all tokens
    default:
        writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
    }
})
```

### Database Schema

```sql
-- New columns on existing oauth_clients table (or new table)
ALTER TABLE oauth_clients ADD COLUMN IF NOT EXISTS
    registration_access_token_hash TEXT DEFAULT '';

-- For RFC 7592 token rotation tracking
ALTER TABLE oauth_clients ADD COLUMN IF NOT EXISTS
    registration_access_token_issued_at TIMESTAMPTZ DEFAULT NOW();
```

### Multi-Tenant Considerations

- All clients are scoped to `tenant_id` (RLS already enforced)
- Tenant admin can view/revoke all clients via `ListClients()` / `DeleteClient()`
- Per-tenant client limits (e.g., max 100 clients/tenant) to prevent abuse
- Registration requires tenant context — either via `X-Tenant-ID` header or
  an initial access token bound to a tenant

---

## 5. Security Analysis

| Threat | Mitigation |
|--------|------------|
| **Open registration abuse** (spam clients) | Require initial access token; per-tenant rate limit |
| **Redirect URI attacks** | Exact-match validation; HTTPS-only in production; reject localhost except for dev tenants |
| **Secret compromise** | Argon2id hash at rest (already done); rotation endpoint; short-lived secrets |
| **Token replay** | Registration access token scoped to single client; rotation on PUT |
| **Resource exhaustion** | Per-tenant client count limit; per-IP registration throttle |
| **software_id tracking** | Track distributed app instances; aggregate analytics per software_id |

**Recommendation**: Implement protected registration (initial access token) as default.
Open registration should be opt-in per tenant, behind an admin flag.

---

## 6. Comparison with Other Implementations

| Feature | Auth0 | Keycloak | Ory Hydra | GGID |
|---------|-------|----------|-----------|------|
| RFC 7591 registration | Yes (M2M apps) | Yes (client-reg service) | Yes (plugin) | **Yes** |
| RFC 7592 management | Partial | Yes | Yes | **No** |
| Registration access token | Proprietary token | Yes | Yes | **No** |
| Initial access token | Tenant API key | Yes (configurable) | Yes | **No** |
| Secret rotation | Via management API | Yes | Via PUT | **No** |
| software_id tracking | Yes | Yes | No | **No** |
| Multi-tenant scoping | Per-tenant apps | Per-realm | Per-project | **Yes** (tenant_id) |

**GGID advantage**: already has RFC 7591 registration with Argon2id hashing and
tenant isolation. Adding RFC 7592 management is the clear next step.

---

## 7. Roadmap

| Phase | Scope | Effort | Priority |
|-------|-------|--------|----------|
| **Phase 1** | Add `contacts` field; registration access token in response; initial access token validation | 2-3 days | P1 |
| **Phase 2** | RFC 7592 management: `GET/PUT/DELETE /oauth/register/{client_id}` | 3-4 days | P1 |
| **Phase 3** | Secret rotation endpoint (`POST /oauth/register/{id}/rotate`) | 1-2 days | P2 |
| **Phase 4** | software_id multi-tenant tracking + analytics dashboard | 2-3 days | P3 |

**Total effort**: ~1.5-2 weeks for Phase 1-2 (core RFC compliance).

**Priority**: P1 — critical for developer self-service and multi-tenant onboarding.
Without RFC 7592, clients have no way to self-manage after registration, forcing
admin intervention for every metadata change.

### Phase 1 Implementation Checklist

1. [ ] Add `RegistrationAccessToken` field to `DynamicRegistrationResponse`
2. [ ] Generate registration access token in `DynamicClientRegister()`
3. [ ] Store token hash in `oauth_clients.registration_access_token_hash`
4. [ ] Add `Contacts` field to `DynamicRegistrationRequest`
5. [ ] Add initial access token validation middleware (configurable on/off)
6. [ ] Add per-tenant client count limit check
7. [ ] Tests: registration with/without initial token, tenant isolation, rate limit
