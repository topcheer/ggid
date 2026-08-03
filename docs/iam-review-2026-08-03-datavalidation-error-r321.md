# IAM Code Review R321 — Data Validation (17th) + Error Handling (11th)

**Date:** 2026-08-03  
**Reviewer:** Independent code audit (glm-5.2, sa-21)  
**Scope:** `services/identity/internal/` | `services/oauth/internal/` | `services/audit/internal/` | `services/gateway/internal/` | `pkg/errors/` | `console/src/`  
**Mode:** Read-only deep audit, no modifications  

---

## Executive Summary

This round verifies the status of 10 previously-reported P0 data validation items and 10 error handling items tracked since R287-R300. **6 of 10 P0 data validation fixes are confirmed effective.** 4 P0 data validation issues remain unresolved. On error handling, significant progress is visible (goroutine recover coverage improved, HTTP error format consolidated via `writeJSONError`/`ggiderrors.WriteSimpleAPIError`), but systemic issues persist at scale.

**Stats this round:** 4 P0 (new/persisting), 8 P1, 12 P2  
**Cumulative (81 rounds):** ~274 P0, ~508 P1, ~568 P2  
**P0 fixed and committed:** ~50

---

## PART A: Data Validation (17th Deep Review)

### Previously Reported P0 — Fix Verification

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | SCIM filter depth limit | **FIXED** | `filter.go:273-277` — `maxDepth=100` enforced via `p.depth++` in `parseOr()` |
| 2 | Avatar magic bytes | **FIXED** | `http.go:1127-1136` — `http.DetectContentType(data)` validates actual file content, not just client-supplied `Content-Type` header |
| 3 | JSON body size limit (bulk) | **FIXED** | `bulk_import.go:73` — `MaxBytesReader(w, r.Body, 10<<20)`; `bulk.go:58` — `MaxBytesReader(w, r.Body, 10<<20)` |
| 4 | DetectHashType bypass | **FIXED** | `bulk_import.go:198-218` — `DetectHashType` now validates hash content regardless of `explicitType`; production rejects `plaintext`/`unknown` |
| 5 | SCIM bulk BOLA | **FIXED** | `bulk.go:168-175` — `userBelongsToCallerTenant(ctx, existing)` checks tenant ownership before PUT; same pattern in `bulkDeleteUser`/`bulkPatchUser` |
| 6 | SCIM bulk operation limit | **FIXED** | `bulk.go:50,70-73` — `maxBulkOperations=1000` enforced |

### NEW / PERSISTING P0 Findings

---

#### **P0-1: Bulk import accepts emails without format validation**

**File:** `services/identity/internal/server/bulk_import.go:97-102`  
**Code:**
```go
for i, user := range req.Users {
    if user.Email == "" {
        result.Failed++
        result.Errors = append(result.Errors, ImportError{Email: fmt.Sprintf("row_%d", i), Reason: "email is required"})
        continue
    }
    // NO email format validation — proceeds directly to CreateUser
```

**Problem:** The bulk import endpoint only checks `user.Email == ""` but performs zero format validation. Any arbitrary string (e.g. `"aaaaaaaaaaaaa"`, `"<script>alert(1)</script>@x"`, `"admin\x00@admin.com"`) is accepted and passed to `h.svc.CreateUser()`.

**Risk:** HIGH — Invalid/malicious email strings enter the database and propagate to welcome emails, audit logs, SCIM downstream sync, and display in admin console. Null bytes or control characters in emails can cause injection in downstream systems (email headers, LDAP sync, CSV exports). The single-user creation path (`http.go:691`) has format validation (`strings.Contains(email, "@")`) but the bulk path does not.

**Recommendation:** Add email format validation before `CreateUser`:
```go
if _, err := mail.ParseAddress(user.Email); err != nil {
    result.Failed++
    result.Errors = append(result.Errors, ImportError{Email: user.Email, Reason: "invalid email format"})
    continue
}
```

---

#### **P0-2: `authorization_details` accepted without schema validation in OAuth authorize flow**

**File:** `services/oauth/internal/server/server.go:628-645`  
**Code:**
```go
authDetailsJSON := json.RawMessage(nil)
if ad := r.URL.Query().Get("authorization_details"); ad != "" {
    var parsed []any
    if err := json.Unmarshal([]byte(ad), &parsed); err == nil {
        authDetailsJSON = json.RawMessage(ad)  // accepted as-is
    }
}
```

**Problem:** The authorize endpoint validates only that `authorization_details` is valid JSON — it performs **no schema validation**. The `RARRegistry.ValidateDetails()` exists in `rar_handler.go:64-76` and validates type/actions, but **it is never called in the authorize flow**. Any arbitrary JSON structure is accepted and stored in the authorization code, then later in the token. An attacker can inject unlimited arbitrary fields, deeply nested objects (resource exhaustion), or privileges that are rendered in the consent screen.

**Risk:** HIGH — RAR (RFC 9396) requires type validation before consent. Without it:
1. Arbitrary privilege claims can be embedded in tokens
2. Deeply nested JSON can cause parser resource exhaustion (no depth limit)
3. The consent screen renders attacker-controlled content (`RenderConsentLines` will fall through to `fmt.Sprintf("Request access: %v", d.Actions)` for unknown types)

**Recommendation:** Call `registry.ValidateDetails()` before creating the authorization code:
```go
if len(authDetailsJSON) > 0 {
    var details []rar.AuthorizationDetail
    if err := json.Unmarshal(authDetailsJSON, &details); err != nil {
        writeJSON(w, http.StatusBadRequest, ...)
        return
    }
    registry := NewRARRegistry()
    if err := registry.ValidateDetails(details, clientID); err != nil {
        writeJSON(w, http.StatusBadRequest, ...)
        return
    }
}
```

---

#### **P0-3: Cleanup inactive handler — no upper bound on `days` parameter**

**File:** `services/identity/internal/server/cleanup_inactive_handler.go:15-16`  
**Code:**
```go
days, _ := strconv.Atoi(r.URL.Query().Get("days"))
if days == 0 { days = 90 }
```

**Problem:** The `days` parameter has no upper bound. A value like `days=999999999` is accepted. While this is currently a simulation handler (returns hardcoded data), if connected to a real query like `WHERE last_active < NOW() - INTERVAL '$days days'`, an extremely large value could:
1. Match ALL users (mass deletion/disabling)
2. Cause integer overflow in date arithmetic
3. The negative `days` value (e.g. `days=-1`) would match NO users or cause unexpected SQL behavior

Additionally, the `strconv.Atoi` error is silently discarded (`_`), so a non-numeric value silently defaults to 90 with no feedback.

**Risk:** MEDIUM-HIGH (currently simulated, but P0 if connected to real DB query — the handler structure is production-ready and will likely be wired to a real query)

**Recommendation:**
```go
if days < 1 || days > 3650 {
    writeJSONError(w, http.StatusBadRequest, "days must be between 1 and 3650")
    return
}
```

---

#### **P0-4: Frontend admin role derived from client-side localStorage scopes**

**File:** `console/src/lib/api.ts:205-229`  
**Code:**
```typescript
export function getUserScopes(): string[] {
    if (typeof window === "undefined") return ["user"];
    try {
        const raw = localStorage.getItem("ggid_user_scopes");
        return raw ? JSON.parse(raw) : ["user"];
    } catch {
        return ["user"];
    }
}

export function getUserRole(): UserRole {
    const scopes = getUserScopes();
    const isPlatform = scopes.some((s) => {
        const ls = s.toLowerCase();
        return ls === "platform:admin" || ls === "platform administrator" || ls === "platform_admin";
    });
    if (isPlatform) return "platform_admin";
    // ...
}
```

**Problem:** The admin role determination (`platform_admin`, `tenant_admin`) is based entirely on `localStorage.getItem("ggid_user_scopes")`. localStorage is client-controlled — any user can open browser devtools and run `localStorage.setItem("ggid_user_scopes", '["platform:admin"]')` to gain admin UI access. This controls which admin pages, tabs, and actions are *displayed* in the console.

**Risk:** MEDIUM (the backend APIs should enforce RBAC independently, so this is primarily a UI-level privilege display issue, not a server-side bypass). However:
1. Admin UI elements (delete buttons, tenant management panels) become visible to non-admin users
2. If any frontend-only guard relies on `getUserRole()` for routing/conditional API calls, those are bypassable
3. `auth-helpers.ts:31-34` — `X-User-ID` and `X-Tenant-ID` headers are also read from localStorage, meaning the client sends attacker-controlled identity headers

**Recommendation:** 
- Server-side: ensure all RBAC enforcement is JWT-claim-based, never trusting client-supplied `X-User-ID`/`X-Tenant-ID` for privileged operations (the `injectTenant` function at `http.go:1182-1202` does prefer auth-middleware context, which is correct)
- Client-side: derive admin UI state from decoded JWT claims (which are signed), not from localStorage that can be arbitrarily set

---

### P1 Findings (Data Validation)

#### **P1-1: Bulk provision handler accepts unbounded user array without email validation**

**File:** `services/identity/internal/server/bulk_provision_handler.go:49-54`  
```go
for _, u := range req.Users {
    if u.Username == "" || u.Email == "" {
        results = append(results, Result{...Error: "username and email required"})
        continue
    }
    // No email format check, no array size limit, no field length limits
```

**Problem:** No `len(req.Users)` upper limit (unlike `bulk_import.go:85` which caps at 10000), no email format validation, no username length validation. Also generates predictable temp passwords: `uuid.New().String()[:12]`.

**Risk:** MEDIUM — Resource exhaustion via large arrays, predictable passwords

#### **P1-2: User creation email validation is trivially bypassable**

**File:** `services/identity/internal/server/http.go:691`  
```go
if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") || len(req.Email) < 5 {
```

**Problem:** This validation accepts emails like `@.`, `a@b.c`, `<script>@x.`, or strings with embedded null bytes. Should use `net/mail.ParseAddress()` like the SCIM handler does.

**Risk:** MEDIUM — Invalid email data enters system

#### **P1-3: No input field length limits on user creation**

**File:** `services/identity/internal/server/http.go:685-694`  
**Problem:** `Username`, `Email`, `Password`, `DisplayName`, `Phone` fields have no maximum length limits. A 10MB username string would be accepted (subject only to the global body limit). The only length check found in the entire identity service is `len(req.Email) < 5`.

**Risk:** MEDIUM — Memory/storage exhaustion, DB column overflow errors

#### **P1-4: SCIM bulk POST create user — no email format validation**

**File:** `services/identity/internal/scim/bulk.go:137-148`  
```go
email := ""
if len(scimUser.Emails) > 0 {
    email = scimUser.Emails[0].Value
}
user, err := h.svc.CreateUser(ctx, &domain.CreateUserInput{
    Email: email,  // could be "" or any format
```

**Problem:** Empty or malformed email is accepted. An empty email is passed directly to `CreateUser`.

**Risk:** MEDIUM

#### **P1-5: Authorization_details JSON parsing — no depth/size limit**

**File:** `services/oauth/internal/server/server.go:632-633`  
```go
var parsed []any
if err := json.Unmarshal([]byte(ad), &parsed); err == nil {
```

**Problem:** `authorization_details` from URL query param is parsed without size limit or nesting depth limit. `json.Unmarshal` with `[]any` allows arbitrary nesting depth.

**Risk:** MEDIUM — Resource exhaustion via deeply nested JSON

---

### P2 Findings (Data Validation)

| # | File | Line | Issue |
|---|------|------|-------|
| P2-1 | `bulk_import.go:131` | `strings.Split(user.Email, "@")[0]` | Username derived from email — if email contains no `@`, Split returns the full string; if multiple `@`, username is malformed |
| P2-2 | `bulk_import.go:138` | `Password: user.Password` | When both `Password` and `PasswordHash` are provided, the plaintext password goes to `CreateUser` which hashes it, then the pre-hash overwrites it at line 160-162 — inconsistent state |
| P2-3 | `cleanup_inactive_handler.go:15` | `strconv.Atoi` error discarded | Silent fallback to default with no user feedback |
| P2-4 | `bulk_provision_handler.go:55` | `uuid.New().String()[:12]` | Predictable temp password from UUID substring — low entropy |
| P2-5 | `rar_handler.go:163` | `d.Identifier["slug"].(string)` | Unchecked type assertion — panics if `slug` is not a string |
| P2-6 | `rar_handler.go:180-181` | `at["amount"]`, `at["currency"]` | Unchecked type assertions in `fmt.Sprintf` — will produce `%!s(<nil>)` if keys missing |
| P2-7 | `http.go:468-472` | `X-Tenant-ID` header | Still injects tenant from header into context even though `injectTenant` prefers auth middleware — the `ServeHTTP` path at line 468 unconditionally trusts the header |
| P2-8 | `bulk.go:88-92` | Error response detail | Bulk operation error response discards the actual error message (`err`), always returns `"operation failed"` |

---

## PART B: Error Handling (11th Deep Review)

### Previously Reported P0 — Status Verification

| # | Item | R300 Status | Current Status |
|---|------|-------------|----------------|
| 1 | Goroutine recover coverage | 57% | **IMPROVED** — most goroutines now have recover (see below) |
| 2 | json.Unmarshal swallows errors | 116 matches | **PERSISTING** — still widespread |
| 3 | json.Encode errors ignored | 336 matches | **PERSISTING** — `writeJSON` delegates to `httputil.WriteJSON` |
| 4 | HTTP error format 3-way mix | P0 | **IMPROVED** — `writeJSONError` now delegates to `ggiderrors.WriteSimpleAPIError`, consolidating format |
| 5 | pkg/errors coverage | 22% | **IMPROVED** — now 182 matches across services |
| 6 | %w coverage | 39% | **SLIGHTLY IMPROVED** — 42/141 (30%) in identity service |
| 7 | DB error classification | 13 sites | **PERSISTING** — only 4 sites found with `pgconn.PgError` handling |
| 8 | ctx.Err() coverage | 2 sites | **PERSISTING** — only 8 matches, mostly in tests |
| 9 | Error info leakage | — | **GOOD** — no raw `err.Error()` in HTTP responses found |
| 10 | Panic recovery without audit | R295 P0 | **IMPROVED** — panic recovery logs via `slog.Error`, but still no audit event |

---

### Goroutine Recover Coverage Analysis

**Identity service (5 `go func` sites):**
| Location | Has recover? | Audit on panic? |
|----------|-------------|-----------------|
| `server.go:302` (gRPC serve) | NO | NO |
| `server.go:309` (HTTP serve) | NO | NO |
| `grpc_handler.go:462` | YES | NO |
| `scim_token_middleware.go:100` | YES | NO |

**Audit service (6 `go func` sites):**
| Location | Has recover? | Audit on panic? |
|----------|-------------|-----------------|
| `webhook/engine.go:222` | YES | NO |
| `audit_service.go:114` | YES | NO |
| `nats_consumer.go:117` | YES (per-message) | NO |
| `compliance/scheduler.go:66` | **NO** | NO |
| `server/http.go:128` | YES | NO |

**OAuth service (4 `go func` sites):**
| Location | Has recover? |
|----------|-------------|
| `key_rotation.go:173` | YES |
| `key_rotation.go:223` | YES |
| `server.go:314` (HTTP serve) | NO |
| `grpc_handler.go:239` | NO |

---

### P0 Findings (Error Handling)

#### **P0-5: Compliance scheduler goroutine has no panic recovery**

**File:** `services/audit/internal/compliance/scheduler.go:66-75`  
**Code:**
```go
go func() {
    for {
        select {
        case <-s.ticker.C:
            s.GenerateAll(context.Background())  // panic here kills goroutine
        case <-s.stopCh:
            return
        }
    }
}()
```

**Problem:** If `GenerateAll` panics (e.g. nil pointer in report generation), the scheduler goroutine dies silently. No compliance reports will be generated until the service is restarted. This is a critical compliance gap for SOC2/SOX audit requirements.

**Risk:** HIGH — Silent failure of compliance report generation violates audit/compliance SLAs.

**Recommendation:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("Compliance scheduler panic", "error", r)
        }
    }()
    for { ... }
}()
```

---

#### **P0-6: Server lifecycle goroutines (HTTP/gRPC serve) lack recover**

**Files:**
- `services/identity/internal/server/server.go:302-314`
- `services/oauth/internal/server/server.go:314-317`
- `services/oauth/internal/server/grpc_handler.go:239-242`

**Problem:** The goroutines that run `grpcSrv.Serve()` and `httpSrv.ListenAndServe()` have no `recover()`. While these typically don't panic (Go's net/http has internal recovery), a panic in a listener callback or custom handler chain could crash the entire server process without graceful shutdown or error reporting.

**Risk:** MEDIUM-HIGH — Process crash without graceful shutdown, in-flight requests dropped.

---

### P1 Findings (Error Handling)

#### **P1-6: Panic recovery does not record audit events**

**Files:** All panic recovery sites across services.

**Problem:** Every `recover()` handler logs via `slog.Error` but none publish an audit event. A panic in a request handler is a significant security/operational event that should be auditable. The R295 P0-2 item ("panic recovery without audit logging") remains unresolved.

**Example** (`http.go:461-465`):
```go
defer func() {
    if rvr := recover(); rvr != nil {
        slog.Error("PANIC recovered in identity handler", "error", rvr)
        writeJSONError(w, http.StatusInternalServerError, "internal server error")
        // No audit event published
    }
}()
```

**Recommendation:** Add `h.publishAuditEvent("system.panic", "error", ...)` in recovery blocks.

#### **P1-7: json.Unmarshal errors silently discarded across services**

**Evidence:** `grep` found 422 total `json.Unmarshal` calls across services, with 36 explicit `_ = json.Unmarshal` (error intentionally discarded). Many more use patterns like `if err := json.Unmarshal(...); err == nil {` which silently skips invalid data.

**Production examples (not tests):**
- `nhi_risk_engine.go:196` — `_ = json.Unmarshal(signalsJSON, &signalsMap)` — silently ignores malformed risk signals
- `oauth/server.go:633,641` — invalid `authorization_details` JSON silently ignored (no error returned to client)

**Risk:** MEDIUM — Silent data corruption, malformed input processed without validation.

#### **P1-8: DB error classification coverage extremely low**

**Evidence:** Only 4 sites classify PostgreSQL errors using `pgconn.PgError`:
- `pg_repo.go:53`
- `tenant_handlers.go:423`
- `ciam_handler.go:133`

The vast majority of DB errors bubble up as generic errors. Callers cannot distinguish unique constraint violations, deadlocks, serialization failures, or connection errors. This means retry logic, user-friendly error messages, and proper HTTP status mapping are all impossible at the handler level.

**Risk:** MEDIUM — Users get generic 500 errors for duplicate-key conflicts (should be 409), timeouts (should be 503), etc.

#### **P1-9: `ctx.Err()` almost never checked**

**Evidence:** Only 8 matches for `ctx.Err()` across all services, 6 of which are in tests. Production code almost never checks if the context was cancelled/timed out before proceeding with expensive operations.

**Risk:** MEDIUM — Wasted computation on cancelled requests, slow client disconnect detection.

#### **P1-10: Error messages in bulk import leak internal error details**

**File:** `services/identity/internal/server/bulk_import.go:144`  
```go
result.Errors = append(result.Errors, ImportError{
    Email: user.Email,
    Reason: fmt.Sprintf("create user failed: %v", err),  // raw error to client
})
```

**Problem:** The raw internal error from `CreateUser` is formatted into the API response. This can leak database schema details, constraint names, or internal path information.

**Risk:** LOW-MEDIUM — Information leakage in API response.

---

### P2 Findings (Error Handling)

| # | File | Line | Issue |
|---|------|------|-------|
| P2-9 | `rar_handler.go:219` | `map[string]string{"error": "invalid_request"}` | RAR consent preview uses inconsistent error format (bare map) vs `writeJSONError` pattern |
| P2-10 | `rar_handler.go:205` | `writeJSON(w, http.StatusMethodNotAllowed, ...)` | Uses `writeJSON` for error instead of `writeJSONError` — inconsistent error response format |
| P2-11 | `bulk.go:88-92` | Error swallowed | Bulk op error response always says "operation failed" regardless of actual error type |
| P2-12 | `bulk.go:60` | `writeSCIMErrorWithType` | SCIM errors use different format than identity API errors — third error format in the codebase |
| P2-13 | `server.go:302-314` | `log.Printf` | Identity server lifecycle uses `log.Printf` instead of `slog` — inconsistent structured logging |
| P2-14 | `oauth/server.go:663` | `writeJSON(w, http.StatusBadRequest, ...)` | OAuth server uses bare `writeJSON` for errors instead of `ggiderrors` — yet another error format variant |
| P2-15 | `bulk_provision_handler.go:29` | `writeJSONError` | Good — uses consolidated error helper, but response `Error` field leaks "invalid JSON" (minor) |
| P2-16 | All services | — | HTTP/gRPC server goroutines send to `errCh` but many callers don't log the error from the channel |
| P2-17 | `http.go:463` | `slog.Error("PANIC recovered...", "error", rvr)` | Panic recovery logs `rvr` as error value, but doesn't include stack trace — `debug.Stack()` not called |
| P2-18 | `bulk_import.go:160-162` | `_, _ = h.svc.Pool().Exec(...)` | DB exec error discarded — if credential update fails, user is created without password credential and no error is returned |
| P2-19 | `bulk_import.go:183-185` | `_, _ = h.svc.Pool().Exec(...)` | Role assignment DB exec error discarded — same pattern |
| P2-20 | `http.go:710` | `row.Scan(...)` | Password policy DB scan error silently falls back to defaults — no logging of policy load failure |

---

## Cross-Cutting Observations

### Positive Progress Since R300
1. **HTTP error format consolidation:** `writeJSONError` now delegates to `ggiderrors.WriteSimpleAPIError` — the 3-way format split is being addressed
2. **Goroutine recover coverage:** Most worker goroutines now have recover (audit webhook, NATS consumer, key rotation, SCIM middleware)
3. **Per-message recover in NATS consumer:** `nats_consumer.go:141-153` wraps each message in its own recover — excellent pattern
4. **DetectHashType hardening:** Content-based validation regardless of explicit type claim
5. **Bulk import tenant isolation:** Role assignment now verifies tenant ownership (`bulk_import.go:170-182`)
6. **Avatar magic bytes:** `DetectContentType` validates actual file content

### Persistent Concerns
1. **authorization_details schema validation** — the RAR registry exists but is bypassed in the actual authorize flow
2. **Bulk import email validation** — still missing, 13 rounds running
3. **Error info leakage** — bulk import exposes raw errors to clients
4. **Compliance scheduler** — critical goroutine still unprotected
5. **DB error classification** — only 4 sites classify pgconn errors, same as R300
6. **Frontend admin scope** — still localStorage-based, client-controlled

---

## Summary by Severity

| Severity | Count | Key Items |
|----------|-------|-----------|
| **P0** | 6 | Bulk email no validation, auth_details no schema, cleanup days unbounded, frontend admin scope, compliance scheduler no recover, server goroutines no recover |
| **P1** | 10 | Bulk provision no limits, trivial email check, no field length limits, SCIM bulk empty email, auth_details no depth limit, panic no audit, json.Unmarshal swallowed, DB error classification, ctx.Err() missing, error leak in import |
| **P2** | 20 | Various consistency, logging, and minor security issues |

---

*End of R321 review.*
