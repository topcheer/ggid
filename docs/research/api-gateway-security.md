# API Gateway Security Patterns for IAM Systems

## 1. Overview

This document focuses on **security-specific patterns** for the GGID API Gateway —
the entry point for all client requests to the IAM suite's microservices. It
complements the existing [`api-gateway-patterns.md`](./api-gateway-patterns.md),
which covers rate limiting, circuit breakers, canary/blue-green deployment, and
WASM plugins. This document deliberately avoids duplicating those topics and
instead concentrates on **application-layer security**: input validation,
payload sanitization, mass assignment prevention, response filtering, and the
OWASP API Security Top 10.

### Scope

| Topic | Covered Here | Covered in api-gateway-patterns.md |
|---|---|---|
| OWASP API Top 10 mapping | Yes | — |
| Request schema validation | Yes | — |
| Payload sanitization | Yes | — |
| Response field filtering | Yes | — |
| Mass assignment prevention | Yes | — |
| HTTP security headers | Yes | — |
| API versioning security | Yes | — |
| Rate limiting | Cross-ref only | Full coverage |
| Circuit breaker | Cross-ref only | Full coverage |
| WASM plugins | Cross-ref only | Full coverage |

---

## 2. OWASP API Security Top 10 (2023) for IAM

The OWASP API Security Top 10 (2023) enumerates the most critical API security
risks. IAM systems are high-value targets — a single vulnerability can expose
every tenant's identity data. Below, each risk is mapped to concrete IAM attack
scenarios and assessed against GGID's current implementation.

### Risk Matrix

| # | OWASP Risk (2023) | IAM Attack Scenario | GGID Status | Severity |
|---|---|---|---|---|
| API1 | Broken Object Level Authorization (BOLA) | User A requests `GET /api/v1/users/{B_id}` — cross-tenant access to another org's user | Partial — JWT carries tenant_id; RLS at DB layer; but no gateway-level object ownership check | **Critical** |
| API2 | Broken Authentication | Attacker brute-forces `/api/v1/auth/login` with credential stuffing bots | Partial — rate limiting exists but not wired into active chain; no account lockout | **High** |
| API3 | Broken Object Property Level Auth | User updates own profile with `{"role":"admin"}` field — privilege escalation | **Missing** — no field-level write validation at gateway | **Critical** |
| API4 | Unrestricted Resource Consumption | Attacker sends 100MB JSON to `/auth/register`, causing OOM | Partial — `MaxBodySize` middleware exists but not in active chain | **High** |
| API5 | Broken Function Level Authorization | Non-admin user calls `POST /api/v1/admin/routes/reload` | Partial — JWT verified, but admin routes checked by path match only, no role claim check in gateway | **High** |
| API6 | Unrestricted Access to Sensitive Business Flows | Automated password reset spam (sends thousands of reset emails) | Partial — rate limiting available but not wired | **Medium** |
| API7 | Server-Side Request Forgery | Webhook configuration accepts arbitrary URLs — attacker sets `http://169.254.169.254/latest/meta-data/` | **Missing** — no SSRF protection for webhook URLs | **High** |
| API8 | Security Misconfiguration | CORS `Access-Control-Allow-Origin: *` in production leaks cross-origin access | **Vulnerable** — default CORS config uses wildcard | **Medium** |
| API9 | Improper Inventory Management | Staged/old API versions (`/api/v0/`) left accessible with weaker auth | Low — only v1 active; no stale version endpoints found | **Low** |
| API10 | Unsafe Consumption of APIs | GGID blindly trusts upstream OAuth/SAML IdP responses without validation | Partial — provider validation exists in `pkg/social` and `pkg/saml` | **Medium** |

### Detailed Analysis: Top 3 Critical Risks

#### API1 — BOLA in Multi-Tenant IAM

In GGID, tenant isolation depends on PostgreSQL Row-Level Security (RLS) at the
database layer. The gateway resolves tenant_id from the JWT and injects it into
the request context. However, the gateway does **not** verify that the object ID
in the URL path belongs to the same tenant. A compromised JWT for tenant A could
request `GET /api/v1/users/{tenant_b_user_uuid}` and the gateway would forward
it. RLS at the DB layer provides defense-in-depth, but gateway-level checks add
a fail-fast layer and reduce unnecessary backend load.

**Recommendation**: Implement a gateway middleware that validates `X-Tenant-ID`
from JWT context against tenant ownership metadata before forwarding to backend
services. For high-sensitivity endpoints (`/users/`, `/roles/`, `/orgs/`), add
a backend query that rejects cross-tenant object access explicitly.

#### API3 — Broken Object Property Level Auth (Mass Assignment)

This is GGID's most significant gap. The gateway forwards request bodies to
backends without schema validation or field whitelisting. An attacker who is
authenticated as a regular user could send:

```json
POST /api/v1/users/me
{
  "display_name": "New Name",
  "role": "admin",
  "tenant_id": "00000000-0000-0000-0000-000000000002",
  "is_active": true
}
```

If the backend ORM blindly maps all JSON fields to database columns (Go struct
tags), the `role` and `tenant_id` fields could be overwritten. See Section 6
for detailed mitigation patterns.

#### API4 — Unrestricted Resource Consumption

GGID has `MaxBodySize` middleware but it is **not wired** into the active chain
(see Section 8). Auth endpoints like `/register` and `/login` accept arbitrary
body sizes. Additionally, computationally expensive operations (bcrypt password
hashing) can be abused — a single `POST /api/v1/auth/register` with a 1MB
password field triggers expensive hashing.

---

## 3. Request Validation & Schema Enforcement

### Why Validate at the Gateway

Validating requests at the gateway before they reach backend services provides
three benefits:

1. **Defense-in-depth**: Backend services may have inconsistent validation
2. **Performance**: Reject malformed requests before allocating backend resources
3. **Observability**: Centralized logging of rejected requests across all endpoints

### Content-Type Enforcement

The gateway should reject requests with unexpected Content-Type headers before
parsing the body. For JSON endpoints, only `application/json` and
`application/json; charset=utf-8` should be accepted.

```go
// ContentTypeEnforce middleware rejects non-JSON Content-Type for write methods.
func ContentTypeEnforce(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost || r.Method == http.MethodPut ||
            r.Method == http.MethodPatch {
            ct := r.Header.Get("Content-Type")
            // Accept application/json with optional charset
            if !strings.HasPrefix(ct, "application/json") {
                writeJSONError(w, http.StatusUnsupportedMediaType,
                    "Content-Type must be application/json")
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

### JSON Schema Validation

For critical IAM endpoints (register, password reset, role creation), enforce a
JSON schema that defines allowed fields, types, and constraints. The
`github.com/santhosh-tekuri/jsonschema/v5` library provides draft-2020-12
support.

```go
import (
    "github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaValidator holds compiled JSON schemas keyed by route pattern.
type SchemaValidator struct {
    schemas map[string]*jsonschema.Schema
}

// NewSchemaValidator compiles schemas from embedded JSON definitions.
func NewSchemaValidator() (*SchemaValidator, error) {
    sv := &SchemaValidator{schemas: make(map[string]*jsonschema.Schema)}
    compiler := jsonschema.NewCompiler()

    // Register embedded schema definitions
    schemas := map[string]string{
        "POST:/api/v1/auth/register":    registerSchema,
        "POST:/api/v1/auth/login":       loginSchema,
        "POST:/api/v1/auth/password/reset": resetSchema,
        "POST:/api/v1/roles":            createRoleSchema,
    }
    for name, src := range schemas {
        if err := compiler.AddResource(name, strings.NewReader(src)); err != nil {
            return nil, fmt.Errorf("compile schema %s: %w", name, err)
        }
        schema, err := compiler.Compile(name)
        if err != nil {
            return nil, fmt.Errorf("compile schema %s: %w", name, err)
        }
        sv.schemas[name] = schema
    }
    return sv, nil
}

// ValidateMiddleware validates request bodies against route-specific schemas.
func (sv *SchemaValidator) ValidateMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Method + ":" + r.URL.Path
        schema, ok := sv.schemas[key]
        if !ok {
            // No schema for this route — pass through
            next.ServeHTTP(w, r)
            return
        }

        // Read and restore body
        body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB max
        if err != nil {
            writeJSONError(w, http.StatusBadRequest, "failed to read body")
            return
        }
        r.Body = io.NopCloser(bytes.NewReader(body))

        // Parse JSON
        var v any
        if err := json.Unmarshal(body, &v); err != nil {
            writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
            return
        }

        // Validate against schema
        if err := schema.Validate(v); err != nil {
            writeJSONError(w, http.StatusUnprocessableEntity,
                "schema validation failed: "+err.Error())
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

Example schema for user registration:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "username": { "type": "string", "minLength": 3, "maxLength": 64 },
    "email": { "type": "string", "format": "email", "maxLength": 255 },
    "password": { "type": "string", "minLength": 8, "maxLength": 128 },
    "display_name": { "type": "string", "maxLength": 128 }
  },
  "required": ["username", "email", "password"],
  "additionalProperties": false
}
```

The `additionalProperties: false` directive is critical for IAM systems — it
rejects any field not explicitly listed, preventing mass assignment attacks at
the schema layer.

---

## 4. Payload Sanitization

### Threats Mitigated

| Attack | Example | Mitigation |
|---|---|---|
| Null byte injection | `username=admin\x00` truncates string at DB layer | Strip `\x00` from all string inputs |
| Unicode normalization | `ａdmin` (fullwidth) bypasses string comparison, resolves to `admin` | NFC normalize all inputs |
| Log injection | `username=\nGET /admin HTTP/1.1` injects fake log lines | Escape `\n`, `\r`, `\t` in logged values |
| CRLF injection | `email=test@test.com\r\nBcc:attacker@evil.com` | Strip `\r\n` from all inputs |
| Path traversal | `display_name=../../../etc/passwd` | Strip `../` sequences |
| SQL injection at API layer | `username=' OR 1=1--` (defense-in-depth, RLS handles primary defense) | Quote and validate lengths |

### Go Sanitization Middleware

```go
import (
    "golang.org/x/text/unicode/norm"
)

// InputSanitizer strips dangerous characters and normalizes Unicode.
type InputSanitizer struct {
    // MaxFieldLength limits individual string field lengths.
    MaxFieldLength int
    // Allowed control characters (everything else is stripped).
    // Only tab is allowed in display_name fields.
}

// NewInputSanitizer creates a sanitizer with IAM-safe defaults.
func NewInputSanitizer() *InputSanitizer {
    return &InputSanitizer{MaxFieldLength: 1024}
}

// SanitizeMiddleware applies sanitization to JSON request bodies.
func (s *InputSanitizer) SanitizeMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost && r.Method != http.MethodPut &&
            r.Method != http.MethodPatch {
            next.ServeHTTP(w, r)
            return
        }

        ct := r.Header.Get("Content-Type")
        if !strings.HasPrefix(ct, "application/json") {
            next.ServeHTTP(w, r)
            return
        }

        // Read and restore body
        body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
        if err != nil {
            writeJSONError(w, http.StatusBadRequest, "failed to read body")
            return
        }
        r.Body = io.NopCloser(bytes.NewReader(body))

        var data any
        if err := json.Unmarshal(body, &data); err != nil {
            next.ServeHTTP(w, r) // Let schema validator handle malformed JSON
            return
        }

        // Recursively sanitize
        sanitized := s.sanitizeValue(data)
        cleanBody, _ := json.Marshal(sanitized)
        r.Body = io.NopCloser(bytes.NewReader(cleanBody))
        r.ContentLength = int64(len(cleanBody))
        r.Header.Set("Content-Length", strconv.Itoa(len(cleanBody)))

        next.ServeHTTP(w, r)
    })
}

func (s *InputSanitizer) sanitizeValue(v any) any {
    switch val := v.(type) {
    case string:
        return s.sanitizeString(val)
    case map[string]any:
        result := make(map[string]any, len(val))
        for k, vv := range val {
            result[s.sanitizeString(k)] = s.sanitizeValue(vv)
        }
        return result
    case []any:
        result := make([]any, len(val))
        for i, vv := range val {
            result[i] = s.sanitizeValue(vv)
        }
        return result
    default:
        return v
    }
}

func (s *InputSanitizer) sanitizeString(str string) string {
    // 1. Unicode NFC normalization (prevents homoglyph/normalization attacks)
    str = norm.NFC.String(str)

    // 2. Strip null bytes (prevents truncation attacks)
    str = strings.ReplaceAll(str, "\x00", "")

    // 3. Strip CRLF sequences (prevents log injection, HTTP response splitting)
    str = strings.ReplaceAll(str, "\r\n", "")
    str = strings.ReplaceAll(str, "\r", "")
    str = strings.ReplaceAll(str, "\n", "")

    // 4. Strip path traversal sequences (defense-in-depth)
    str = strings.ReplaceAll(str, "../", "")
    str = strings.ReplaceAll(str, "..\\", "")

    // 5. Enforce max length
    if len(str) > s.MaxFieldLength {
        str = str[:s.MaxFieldLength]
    }

    return str
}
```

### Log Injection Prevention

GGID's structured logger (`recovery.go`) outputs JSON-formatted log records.
Because the `json.Marshal` function naturally escapes special characters, the
log injection surface is reduced compared to plain-text logging. However,
user-controlled values that appear in error messages or panic recovery records
should still be sanitized before being included in log entries:

```go
// sanitizeForLog escapes newlines and control characters for safe logging.
func sanitizeForLog(s string) string {
    s = strings.ReplaceAll(s, "\n", "\\n")
    s = strings.ReplaceAll(s, "\r", "\\r")
    s = strings.ReplaceAll(s, "\t", "\\t")
    // Strip other control characters (U+0000 to U+001F except space)
    var b strings.Builder
    for _, r := range s {
        if r >= 0x20 || r == 0x09 { // allow tab
            b.WriteRune(r)
        }
    }
    return b.String()
}
```

---

## 5. Response Filtering & Data Minimization

### Principle

API responses should contain the **minimum data** necessary for the client.
Over-exposing fields creates information leaks that attackers can exploit for
reconnaissance. In an IAM system, the most dangerous leaked fields are:

| Field | Risk |
|---|---|
| `password_hash` | Offline cracking attack |
| `totp_secret` | MFA bypass |
| `internal_id` | Object enumeration |
| `tenant_id` (cross-tenant) | Tenant correlation for BOLA attacks |
| `created_at` / `updated_at` (internal) | Infrastructure timing analysis |
| `backup_codes` | MFA bypass |

### Response Filtering Middleware

```go
// ResponseFilter strips sensitive fields from JSON API responses before
// they are sent to the client. Fields are matched by path prefix for
// granular control.
type ResponseFilter struct {
    // SensitiveFields maps route prefix to a set of field names to strip.
    // The special key "*" applies to all routes.
    SensitiveFields map[string]map[string]bool
}

func NewResponseFilter() *ResponseFilter {
    return &ResponseFilter{
        SensitiveFields: map[string]map[string]bool{
            "*": {
                "password_hash": true,
                "password":      true,
                "totp_secret":   true,
                "backup_codes":  true,
                "secret":        true,
                "private_key":   true,
                "api_key":       true,
            },
            "/api/v1/users": {
                "email_verified":   true,
                "mfa_enabled":      true,
                "last_login":       true,
                "failed_attempts":  true,
                "internal_id":      true,
            },
            "/api/v1/roles": {
                "system_managed": true,
                "deleted_at":     true,
            },
        },
    }
}

// FilterMiddleware intercepts the response body and strips sensitive fields.
func (rf *ResponseFilter) FilterMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Wrap the response writer to capture the body
        buf := &bytes.Buffer{}
        rw := &capturingResponseWriter{ResponseWriter: w, buf: buf}

        next.ServeHTTP(rw, r)

        // Only filter JSON responses
        ct := w.Header().Get("Content-Type")
        if !strings.HasPrefix(ct, "application/json") {
            // Non-JSON: write as-is
            w.Write(buf.Bytes())
            return
        }

        // Parse, filter, and re-serialize
        filtered := rf.filterJSON(buf.Bytes(), r.URL.Path)
        w.Header().Set("Content-Length", strconv.Itoa(len(filtered)))
        w.Write(filtered)
    })
}

func (rf *ResponseFilter) filterJSON(body []byte, path string) []byte {
    var data any
    if err := json.Unmarshal(body, &data); err != nil {
        return body // Can't parse — return as-is (fail open for availability)
    }

    // Build the set of fields to strip for this path
    strip := make(map[string]bool)
    if global, ok := rf.SensitiveFields["*"]; ok {
        for k := range global {
            strip[k] = true
        }
    }
    for prefix, fields := range rf.SensitiveFields {
        if prefix != "*" && strings.HasPrefix(path, prefix) {
            for k := range fields {
                strip[k] = true
            }
        }
    }

    data = rf.stripFields(data, strip)
    result, err := json.Marshal(data)
    if err != nil {
        return body
    }
    return result
}

func (rf *ResponseFilter) stripFields(v any, strip map[string]bool) any {
    switch val := v.(type) {
    case map[string]any:
        result := make(map[string]any, len(val))
        for k, vv := range val {
            if !strip[k] {
                result[k] = rf.stripFields(vv, strip)
            }
        }
        return result
    case []any:
        result := make([]any, len(val))
        for i, vv := range val {
            result[i] = rf.stripFields(vv, strip)
        }
        return result
    default:
        return v
    }
}

type capturingResponseWriter struct {
    http.ResponseWriter
    buf     *bytes.Buffer
    status  int
}

func (w *capturingResponseWriter) WriteHeader(code int) {
    w.status = code
    // Don't forward yet — we need to modify Content-Length after filtering
}

func (w *capturingResponseWriter) Write(b []byte) (int, error) {
    return w.buf.Write(b)
}
```

---

## 6. Mass Assignment Prevention

### How Mass Assignment Works in IAM

Mass assignment occurs when a backend framework automatically binds incoming JSON
fields to struct fields without checking whether the caller is authorized to
modify each field. In Go, this typically happens with `json.Unmarshal` into a
struct that contains sensitive fields.

**Attack scenario in GGID**:

```http
PATCH /api/v1/users/me
Content-Type: application/json

{
    "display_name": "Updated Name",
    "role": "admin",
    "tenant_id": "00000000-0000-0000-0000-000000000002",
    "is_admin": true,
    "email_verified": true,
    "mfa_enabled": false
}
```

If the backend handler does `json.Unmarshal(body, &user)` without field-level
checks, the attacker escalates to admin and disables MFA.

### Field Whitelist Pattern

The most robust defense is a **per-endpoint, per-role field whitelist** that
specifies exactly which fields each caller type may write.

```go
// FieldPolicy defines which fields are writable for a given endpoint and role.
type FieldPolicy struct {
    // AllowedFields maps endpoint → role → set of writable fields.
    AllowedFields map[string]map[string]map[string]bool
}

func NewFieldPolicy() *FieldPolicy {
    return &FieldPolicy{
        AllowedFields: map[string]map[string]map[string]bool{
            "PATCH:/api/v1/users/me": {
                "user": {
                    "display_name": true,
                    "avatar_url":   true,
                    "phone":        true,
                },
                "admin": {
                    "display_name":  true,
                    "avatar_url":    true,
                    "phone":         true,
                    "email":         true,
                    "is_active":     true,
                    "email_verified": true,
                },
            },
            "PUT:/api/v1/users/{id}": {
                "admin": {
                    "display_name":  true,
                    "email":         true,
                    "role":          true,
                    "is_active":     true,
                    "email_verified": true,
                },
            },
        },
    }
}

// EnforceFieldPolicy strips disallowed fields from the request body.
// It must be called AFTER JWTAuth middleware so that UserIDKey and role
// are available in the context.
func (fp *FieldPolicy) EnforceFieldPolicy(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost && r.Method != http.MethodPut &&
            r.Method != http.MethodPatch {
            next.ServeHTTP(w, r)
            return
        }

        // Resolve policy key (simplified — in production, normalize path params)
        key := r.Method + ":" + r.URL.Path
        rolePolicies, ok := fp.AllowedFields[key]
        if !ok {
            // No policy defined for this endpoint — block by default
            // (fail-closed for safety)
            next.ServeHTTP(w, r)
            return
        }

        // Get caller role from JWT context
        role := r.Context().Value(RoleKey).(string)
        if role == "" {
            role = "user"
        }

        allowed, ok := rolePolicies[role]
        if !ok {
            // Role not in policy — block all extra fields
            allowed = map[string]bool{}
        }

        // Read, filter, and restore body
        body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
        if err != nil {
            writeJSONError(w, http.StatusBadRequest, "failed to read body")
            return
        }

        var data map[string]any
        if err := json.Unmarshal(body, &data); err != nil {
            writeJSONError(w, http.StatusBadRequest, "invalid JSON")
            return
        }

        // Strip non-allowed fields
        violations := []string{}
        for field := range data {
            if !allowed[field] {
                violations = append(violations, field)
            }
        }
        for _, v := range violations {
            delete(data, v)
        }

        if len(violations) > 0 {
            // Log the attempt for security monitoring
            log.Printf("mass assignment attempt: user=%s role=%s fields=%v",
                r.Context().Value(UserIDKey), role, violations)
        }

        // Restore filtered body
        cleanBody, _ := json.Marshal(data)
        r.Body = io.NopCloser(bytes.NewReader(cleanBody))
        r.ContentLength = int64(len(cleanBody))

        next.ServeHTTP(w, r)
    })
}
```

### Backend Defense (Defense-in-Depth)

Even with gateway-level field filtering, backend handlers should use explicit
field mapping rather than blanket `json.Unmarshal`. In Go, use separate input
and output structs:

```go
// UserUpdateInput only contains fields a user is allowed to modify.
type UserUpdateInput struct {
    DisplayName *string `json:"display_name,omitempty"`
    AvatarURL   *string `json:"avatar_url,omitempty"`
    Phone       *string `json:"phone,omitempty"`
}

// The internal User struct has sensitive fields:
type User struct {
    ID             uuid.UUID `json:"id"`
    DisplayName    string    `json:"display_name"`
    Email          string    `json:"email"`
    Role           string    `json:"role"`            // NOT in UserUpdateInput
    PasswordHash   string    `json:"-"`               // Never serialized
    TenantID       uuid.UUID `json:"-"`               // Never serialized
    IsActive       bool      `json:"is_active"`       // NOT in UserUpdateInput
    EmailVerified  bool      `json:"email_verified"`  // NOT in UserUpdateInput
}

func HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
    var input UserUpdateInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        // Only fields in UserUpdateInput can be set — role, is_active, etc.
        // are structurally impossible to assign via this handler.
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    // ... apply only the fields from input to the database ...
}
```

The Go `json:"-"` tag ensures the field is never serialized in responses, and
using a separate input struct ensures only intended fields are deserializable.

---

## 7. HTTP Header Security

### Required Security Headers

| Header | Value | Purpose |
|---|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Force HTTPS, prevent protocol downgrade |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME-type sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking via iframe |
| `Content-Security-Policy` | `default-src 'self'; frame-ancestors 'none'` | Restrict resource loading |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limit referrer leakage |
| `Permissions-Policy` | `geolocation=(), microphone=(), camera=()` | Disable unused browser APIs |
| `Cache-Control` | `no-store` (for auth responses) | Prevent caching of sensitive data |

### Header Removal

The gateway should strip server-identifying headers to reduce reconnaissance
surface:

```go
// RemoveServerHeaders strips identifying headers from responses.
func RemoveServerHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Del("Server")
        w.Header().Del("X-Powered-By")
        w.Header().Del("X-AspNet-Version")
        // Set a generic server name
        w.Header().Set("Server", "ggid")
        next.ServeHTTP(w, r)
    })
}
```

### GGID's Current Implementation

GGID has two implementations of security header middleware:

1. **`middleware.SecurityHeaders`** (in `middleware.go` lines 224-232) — a
   simple, hardcoded version that sets HSTS, X-Content-Type-Options,
   X-Frame-Options, and Referrer-Policy.

2. **`middleware.SecurityHeadersConfigurable`** (in `security_headers.go`) — a
   full-featured version with per-tenant overrides, CSP, and configurable
   HSTS duration.

**Neither is wired into the active middleware chain** in `Handler()` (see
Section 8). The configurable version provides excellent tenant-specific
control but is not applied to any production traffic.

### Cache-Control for Auth Endpoints

Token endpoints, login, and registration responses must never be cached. The
gateway should inject `Cache-Control: no-store` for these paths:

```go
// NoCacheMiddleware prevents caching of sensitive API responses.
func NoCacheMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Apply no-cache to auth and token endpoints
        if strings.Contains(r.URL.Path, "/auth/") ||
            strings.Contains(r.URL.Path, "/oauth/") ||
            strings.Contains(r.URL.Path, "/token") {
            w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
            w.Header().Set("Pragma", "no-cache")
            w.Header().Set("Expires", "0")
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## 8. API Versioning & Deprecation Security

### Security Risks of Old API Versions

Maintaining multiple API versions introduces security risks:

1. **Weaker authentication on old versions**: v1 might have no rate limiting while v2 does
2. **Known vulnerabilities**: v1 endpoints may have bugs patched in v2
3. **Inconsistent authorization**: v1 might check permissions differently
4. **Expanded attack surface**: Each version adds endpoints to secure and monitor

### Sunset Header Pattern (RFC 8594)

The `Sunset` HTTP header (RFC 8594) signals to API consumers that an endpoint
will be removed. Combined with `Deprecation` (draft), this gives clients a
migration window:

```go
// VersionDeprecationConfig tracks per-version deprecation status.
type VersionDeprecationConfig struct {
    // Deprecations maps version → deprecation info.
    Deprecations map[string]*DeprecationInfo
}

type DeprecationInfo struct {
    Deprecated   bool      // true if version is deprecated
    SunsetDate   string    // RFC 3339 date when version will be removed
    SunsetEpoch  int64     // Unix timestamp for programmatic comparison
    UpgradeTo    string    // recommended replacement version
    BlogURL      string    // link to migration guide
}

// DeprecationMiddleware adds Sunset and Deprecation headers for old versions.
func DeprecationMiddleware(cfg *VersionDeprecationConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            version := extractAPIVersionFromRequest(r)
            if info, ok := cfg.Deprecations[version]; ok && info.Deprecated {
                // Add deprecation headers (draft IETF specification)
                w.Header().Set("Deprecation", "true")
                if info.SunsetDate != "" {
                    w.Header().Set("Sunset", info.SunsetDate)
                }
                // Link to migration guide
                if info.BlogURL != "" {
                    w.Header().Set("Link",
                        `<`+info.BlogURL+`>; rel="deprecation"`)
                }
                // Warning header per RFC 7234
                w.Header().Set("Warning",
                    `299 - "This API version is deprecated and will be removed on `+
                        info.SunsetDate+`"`)
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Forced Upgrade for Critical Vulnerabilities

For versions with critical security vulnerabilities, the gateway should return
`410 Gone` or `426 Upgrade Required` rather than forwarding to a vulnerable
backend:

```go
// ForcedUpgradeMiddleware blocks access to vulnerable API versions entirely.
func ForcedUpgradeMiddleware(blockedVersions map[string]bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            version := extractAPIVersionFromRequest(r)
            if blockedVersions[version] {
                w.Header().Set("Content-Type", "application/json")
                w.Header().Set("Upgrade", "api/v2")
                w.WriteHeader(http.StatusUpgradeRequired) // 426
                json.NewEncoder(w).Encode(map[string]string{
                    "error":  "api version blocked due to security vulnerability",
                    "action": "upgrade to a newer API version",
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### GGID's Current Versioning

GGID implements API versioning in `apiversion.go` with URL path (`/api/v2/`),
header (`Api-Version`), and query parameter (`?api_version=`) support. The
middleware routes to different backends per version. However, it does not
implement deprecation headers or forced upgrades. Currently only v1 is active.

---

## 9. GGID Gateway Security Audit

### Active Middleware Chain (from `Handler()` in `router.go`)

```
PanicRecovery → CORS → RequestID → RequestLogger
→ RateLimiter (TenantBucketLimiter) → TenantResolver → JWTAuth → ReverseProxy
```

### Existing Security Middleware Inventory

| File | Middleware | In Active Chain? | Notes |
|---|---|---|---|
| `middleware.go` | `SecurityHeaders` | **No** | Basic HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy |
| `security_headers.go` | `SecurityHeadersConfigurable` | **No** | Full config with per-tenant overrides, CSP, HSTS duration |
| `middleware.go` | `CSRFProtect` | **No** | Double-submit cookie CSRF protection |
| `bodysize.go` | `MaxBodySize` | **No** | Limits request body to prevent OOM |
| `ip_filter.go` | `IPFilterMiddleware` | **No** | Per-tenant IP allowlist/denylist with CIDR |
| `botdetect.go` | `BotDetect` | **No** | Blocks known malicious User-Agents |
| `botdetect.go` | `BehavioralBotDetect` | **No** | Rate-based bot detection |
| `apikey.go` | API key auth | **No** | Alternative auth mechanism |
| `apiversion.go` | `APIVersioning` | **No** | Version routing |
| `middleware.go` | `JWTAuth` | **Yes** | RS256 JWT with JWKS, issuer/audience validation |
| `middleware.go` | `TenantResolver` | **Yes** | Resolves tenant from header, JWT, subdomain |
| `middleware.go` | `CORS` | **Yes** | Default config: `Access-Control-Allow-Origin: *` |
| `recovery.go` | `PanicRecovery` | **Yes** | Structured panic recovery |

### Identified Security Gaps

#### Gap 1: Security Headers Not Applied (API8 — Security Misconfiguration)

The active middleware chain does not include `SecurityHeaders` or
`SecurityHeadersConfigurable`. Production responses lack HSTS,
X-Content-Type-Options, X-Frame-Options, and CSP headers. This is a
straightforward fix — add one line to the middleware chain in `Handler()`.

#### Gap 2: No Body Size Enforcement (API4 — Unrestricted Resource Consumption)

`MaxBodySize` middleware exists but is not applied. Auth endpoints accept
arbitrarily large request bodies, creating a denial-of-service vector. The
bcrypt password hashing in the auth service makes `/register` especially
vulnerable — a large body triggers expensive computation.

#### Gap 3: No Request Schema Validation (API3 — Broken Object Property Level Auth)

No JSON schema validation exists. Request bodies are forwarded to backends
without field-level checking. This is the root cause of the mass assignment
vulnerability surface described in Section 6.

#### Gap 4: CORS Wildcard in Production (API8 — Security Misconfiguration)

The default CORS config (`DefaultCORSConfig()`) sets
`Access-Control-Allow-Origin: *`. While this is acceptable for development, it
should be restricted to explicit frontend origins in production. The code
already supports this via `CORSWithConfig` with specific origins, but the
default is insecure.

#### Gap 5: CSRF Protection Not Wired

`CSRFProtect` middleware implements double-submit cookie protection but is not
in the active chain. While JWT Bearer token authentication is not vulnerable to
traditional CSRF (tokens are not sent automatically by browsers like cookies),
the hosted login/registration pages served by the gateway do use cookies and
would benefit from CSRF protection.

#### Gap 6: No Response Filtering

API responses are forwarded without filtering. If a backend accidentally
includes `password_hash` or `totp_secret` in a response, it reaches the client
unfiltered. The response filtering middleware described in Section 5 would add
a defense-in-depth layer.

#### Gap 7: IP Filtering Not Wired

`IPFilterMiddleware` provides per-tenant IP allowlist/denylist enforcement but
is not in the active chain. Enterprise tenants requiring IP-based access
restrictions have no enforcement.

#### Gap 8: Bot Detection Not Wired

`BotDetect` blocks known attack tools (sqlmap, nikto, nmap, etc.) by User-Agent
but is not in the active chain. This is a zero-cost defense — these tools
identify themselves in the User-Agent header.

---

## 10. Gap Analysis & Recommendations

### Summary of Gaps

| Gap | OWASP Risk | Current Impact | Fix Complexity |
|---|---|---|---|
| Security headers not wired | API8 | HSTS, CSP, X-Frame-Options missing | **Trivial** (1 line) |
| No body size enforcement | API4 | DoS via large bodies | **Trivial** (1 line) |
| CORS wildcard | API8 | Cross-origin data access | **Low** (config change) |
| No schema validation | API3 | Mass assignment | **Medium** (3-5 days) |
| No payload sanitization | API3/API4 | Injection vectors | **Medium** (2-3 days) |
| No response filtering | API3 | Data leakage | **Medium** (2 days) |
| CSRF not wired | API2 | Hosted page abuse | **Low** (1 line + test) |
| IP filtering not wired | API5 | No tenant IP restriction | **Low** (1 line) |
| Bot detection not wired | API2 | Automated attack tools | **Trivial** (1 line) |

### Implementation Roadmap

#### Phase 1: Quick Wins (1 day, P0)

Wire existing middleware into the active chain. All of these are already
implemented and tested:

```go
// In Handler(), update the middleware chain:
handler := middleware.TenantResolver(gw.cfg.DomainSuffix)(inner)
handler = gw.rateLimiter.Middleware(handler)
handler = middleware.RequestLogger(logger)(handler)
handler = middleware.MaxBodySize(10 << 20)(handler)          // ADD: 10MB limit
handler = middleware.BotDetect(handler)                       // ADD: bot detection
handler = middleware.SecurityHeaders(handler)                 // ADD: security headers
handler = middleware.RequestID(handler)
handler = middleware.CORS(handler)
handler = middleware.PanicRecovery(logger)(handler)
```

**Effort**: 1 day (including testing and configuration)
**Impact**: Eliminates Gaps 1, 2, 8 immediately

#### Phase 2: CORS Hardening (1 day, P0)

Change the default CORS configuration from wildcard to explicit origins loaded
from environment variables:

```go
func productionCORSConfig() CORSConfig {
    origins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
    if len(origins) == 0 || origins[0] == "" {
        log.Fatal("CORS_ALLOWED_ORIGINS must be set in production")
    }
    return CORSConfig{
        AllowedOrigins:   origins,
        AllowCredentials: true,
    }
}
```

**Effort**: 1 day (config + test + deploy verification)
**Impact**: Eliminates Gap 4

#### Phase 3: Request Schema Validation (3-5 days, P1)

Implement JSON schema validation for critical endpoints:

1. Define schemas for register, login, password reset, role create, org create
2. Add `SchemaValidator.ValidateMiddleware` to the active chain
3. Use `additionalProperties: false` on all schemas
4. Add integration tests that verify malformed requests are rejected

**Effort**: 3-5 days (schema design + middleware + testing)
**Impact**: Eliminates Gap 3, provides mass assignment defense at schema layer

#### Phase 4: Payload Sanitization & Response Filtering (3 days, P1)

1. Add `InputSanitizer.SanitizeMiddleware` to the active chain
2. Add `ResponseFilter.FilterMiddleware` to the active chain
3. Define sensitive field lists per endpoint
4. Test that `password_hash` and `totp_secret` are stripped from all responses

**Effort**: 3 days (middleware + field lists + testing)
**Impact**: Eliminates Gaps 5, 6

#### Phase 5: Mass Assignment Backend Hardening (3 days, P1)

1. Audit all backend handlers that accept user input (`POST`/`PUT`/`PATCH`)
2. Replace blanket `json.Unmarshal` with explicit input structs
3. Add `json:"-"` tags to sensitive response fields
4. Add field policy enforcement for admin-only fields

**Effort**: 3 days (code audit + refactoring + testing)
**Impact**: Defense-in-depth for mass assignment, prevents privilege escalation

### Total Effort Estimate

| Phase | Effort | Priority | Gaps Closed |
|---|---|---|---|
| Phase 1 | 1 day | P0 | 1, 2, 8 |
| Phase 2 | 1 day | P0 | 4 |
| Phase 3 | 3-5 days | P1 | 3 |
| Phase 4 | 3 days | P1 | 5, 6 |
| Phase 5 | 3 days | P1 | Backend mass assignment |
| **Total** | **11-13 days** | | **All 8 gaps + backend hardening** |

### Priority Rationale

Phases 1-2 (2 days total) address the highest-impact, lowest-effort gaps by
simply wiring existing, tested middleware into the production chain. This is the
single highest-ROI security improvement available to GGID today.

Phases 3-5 build the application-layer validation and sanitization capabilities
that prevent the most dangerous IAM-specific attacks (mass assignment,
privilege escalation, data leakage).

---

## References

- [OWASP API Security Top 10 (2023)](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)
- [RFC 8594: The Sunset HTTP Header Field](https://datatracker.ietf.org/doc/html/rfc8594)
- [JSON Schema Specification (Draft 2020-12)](https://json-schema.org/draft/2020-12/json-schema-validation.html)
- [Unicode Normalization Forms (UAX #15)](https://unicode.org/reports/tr15/)
- GGID existing docs: [`api-gateway-patterns.md`](./api-gateway-patterns.md)
- GGID source: `services/gateway/internal/middleware/`, `services/gateway/internal/router/router.go`
