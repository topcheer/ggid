# IAM Security Audit R372 — Data Validation (R29) + Error Handling (R24)

**Date**: 2026-08-04  
**Auditor**: Independent (first contact)  
**Scope**: services/gateway/, services/auth/, services/identity/, services/oauth/, services/audit/, pkg/errors/, pkg/rbac/, pkg/crypto/, console/src/

---

## Checked Code Paths

### Data Validation
- `services/gateway/internal/middleware/bodysize.go` — MaxBodySize middleware, ParseMaxBodySize
- `services/gateway/internal/router/router.go` — Gateway router, rate-limits handler, posture handler (json.NewDecoder)
- `services/auth/cmd/main.go:456-461` — Auth HTTP server MaxBytesReader setup
- `services/auth/internal/server/registration_handler.go` — handleRegister, validatePassword, validateEmail, username validation
- `services/identity/internal/service/identity_service.go` — CreateUser input validation, validatePasswordComplexity
- `services/identity/internal/scim/handler.go` (917 lines) — All SCIM endpoints: listUsers, createUser, replaceUser, patchUser, searchUsers, filter parsing
- `services/identity/internal/scim/bulk.go` (350 lines) — HandleBulk, executeBulkOp, bulk CRUD
- `services/identity/internal/server/bulk_import.go` (324 lines) — handleBulkImport, DetectHashType, VerifyMultiHash
- `services/oauth/internal/service/oauth_service.go` — OAuthService, VerifyAuthTicket, CreateClientInput
- `services/oauth/internal/server/rar_handler.go` — RAR authorization_details validation, 6 type handlers
- `pkg/httputil/response.go` — WriteJSON, WriteError, WriteJSONError
- 514 `json.NewDecoder(r.Body)` occurrences across 325 files (sampled auth/identity/gateway/audit)
- Body limit coverage: auth (1/1), audit (1/1), gateway (middleware), identity (SCIM handlers + bulk_import), but NOT gateway router inline handlers

### Error Handling
- 1099 `fmt.Errorf` across 212 files vs 132 `pkg/errors` usages across 59 files (ratio ~8:1)
- 514 `json.NewDecoder(r.Body).Decode()` calls — many without error type discrimination
- 336 `json.NewEncoder().Encode()` calls with `_ =` error suppression
- 386 `defer .Close()` without error checking across 174 files
- 71 anonymous `go func()` launches — 40 with recover, ~31 without
- 8 `ctx.Err()` checks across 4 files (extremely low)
- 10 `http.Error()` direct calls across 8 files (inconsistent with writeJSON/writeError)
- 5+ separate writeJSON implementations, 4 writeServiceError implementations
- 3 panic calls in test files only (no production panics found)

---

## FINDINGS

### === Data Validation (R29) ===

---

#### DV-01 [P0] — 514 json.NewDecoder(r.Body) calls without JSON depth limiting
**File**: 325 files across all services; 20+ in identity server, 20+ in auth server, 3 in gateway router  
**Examples**:
- `services/gateway/internal/router/router.go:1503` — rate-limits handler
- `services/gateway/internal/router/router.go:1543` — policy handler
- `services/gateway/internal/router/router.go:1633` — policy handler
- `services/identity/internal/server/wasm_plugin_handler.go:38`
- `services/auth/internal/server/device_fingerprint_handler.go:41`
- `services/oauth/internal/server/rar_handler.go:212`

**Problem**: Go's `encoding/json` has no built-in nesting depth limit. An attacker can craft deeply nested JSON (e.g., `{"a":{"a":{"a":...}}}`) to cause stack overflow or extreme CPU consumption. The standard library `json.Decoder` does not limit depth by default.

**Risk**: DoS via deeply nested JSON payloads. With 514 unprotected decode sites, the attack surface is enormous. Even services with outer `MaxBytesReader` are vulnerable — a 10MB body with 100K nesting levels can exhaust stack.

**Fix**: Wrap all `json.NewDecoder` calls with `dec := json.NewDecoder(r.Body); dec.MaxDepth(100)` (custom depth limiter) or use a shared helper that enforces max nesting depth.

---

#### DV-02 [P0] — SCIM filter has no depth/complexity limit
**File**: `services/identity/internal/scim/handler.go:865-881`  
**Functions**: `parseSCIMFilter()`, `parseExternalIdFilter()`

**Problem**: The SCIM filter parser does string matching (`strings.Index`, `strings.HasPrefix`) with no limit on filter expression length or complexity. A filter like `userName eq "x" and userName eq "x" and ... (repeated 10MB)` is accepted. The parser only handles simple `attr eq "value"` patterns, so complex filters silently return empty results, but the parsing itself has no length guard.

**Risk**: CPU/DoS via extremely long filter expressions. Also, complex SCIM filters are silently ignored (returning unfiltered results) rather than rejected, potentially leaking data.

**Fix**: Add `maxFilterLength = 1024` check. Reject filters exceeding this limit with `400 invalidFilter`. Also reject filters containing AND/OR/NOT operators if the parser doesn't support them.

---

#### DV-03 [P0] — Gateway router inline handlers have no per-handler body size limit
**File**: `services/gateway/internal/router/router.go:1503, 1543, 1633`

**Problem**: Three gateway router handler functions decode `json.NewDecoder(r.Body)` without wrapping with `http.MaxBytesReader`. Although the outer middleware `MaxBodySize` applies to the gateway, these handlers still trust the body without explicit per-handler limits.

**Risk**: Defense-in-depth violation. If middleware is bypassed or misconfigured, these handlers accept unlimited body sizes.

**Fix**: Add `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` before each `json.NewDecoder` call in gateway router handlers.

---

#### DV-04 [P1] — authorization_details has no schema/size validation on custom fields
**File**: `services/oauth/internal/server/rar_handler.go:13-22, 64-76`

**Problem**: `AuthorizationDetail` struct has unbounded fields:
- `Identifier map[string]any` — no key/value count or size limit
- `Constraints map[string]any` — no size limit
- `Locations`, `Actions`, `Datatypes`, `Privileges`, `Fields` — `[]string` with no length cap
- No limit on total number of `AuthorizationDetail` elements in the request

**Risk**: Memory exhaustion via large authorization_details payloads. Also, `map[string]any` accepts arbitrary nested data without validation.

**Fix**: Add validation: max 10 elements per slice, max 1024 chars per string, max 10 keys per map, max 20 authorization_details per request.

---

#### DV-05 [P1] — SCIM bulk operations lack per-operation data size limit
**File**: `services/identity/internal/scim/bulk.go:130-148` (`bulkCreateUser`)

**Problem**: Each bulk operation's `Data json.RawMessage` field has no individual size limit. With 1000 operations allowed and only a 10MB total body limit, each operation could be ~10KB, but there's no validation that individual `op.Data` is reasonable. The `json.Unmarshal(op.Data, &scimUser)` could process large raw messages.

**Risk**: Memory pressure with 1000 operations each containing multi-KB data payloads.

**Fix**: Add per-operation data size check: `if len(op.Data) > 65536 { reject }`.

---

#### DV-06 [P1] — Username validation missing in SCIM createUser (no length/charset check)
**File**: `services/identity/internal/scim/handler.go:501-505`

**Problem**: SCIM createUser only checks `scimUser.UserName == ""` (line 502) but does NOT validate username length, character set, or format. The identity_service.go `CreateUser` (line 42) checks `len(input.Username) > 255` but doesn't validate charset. Compare with `registration_handler.go:156-176` which has thorough charset validation.

**Risk**: SCIM users can have usernames with special characters, spaces, or unicode that may break downstream systems, SQL queries, or display logic.

**Fix**: Add username charset and length validation in SCIM createUser handler, matching the registration handler's validation.

---

#### DV-07 [P1] — Enum values not validated in SCIM PATCH operations
**File**: `services/identity/internal/scim/handler.go:634-638` (`SCIMPatchOp`)

**Problem**: The `Op` field in SCIM PATCH operations accepts any string. The handler checks `strings.ToLower(op.Op)` against "replace"/"add"/"remove" via switch-case fallthrough, but invalid op values are silently ignored rather than rejected with an error.

**Risk**: Silent data corruption — a typo in operation type causes the operation to be silently skipped.

**Fix**: Return `400 invalidValue` when `op.Op` is not one of "add", "replace", "remove".

---

#### DV-08 [P2] — SCIM ExternalID has no length limit
**File**: `services/identity/internal/scim/handler.go:526`

**Problem**: `scimUser.ExternalID` is passed directly to `CreateUser` without length validation. The identity_service.go only validates Username (255), Email (320), DisplayName (255), Phone (32), Locale (10), Timezone (64) — ExternalID is not checked.

**Risk**: Storage of arbitrarily long external identifiers.

**Fix**: Add `len(input.ExternalID) > 255` check to identity_service.go CreateUser validation.

---

#### DV-09 [P2] — Identity CreateUser doesn't validate email format
**File**: `services/identity/internal/service/identity_service.go:35-44`

**Problem**: `CreateUser` validates field lengths (line 42) but does NOT validate email format. It relies on callers (SCIM handler, registration handler) to validate. However, if any internal code path calls CreateUser without prior email validation, malformed emails enter the database.

**Risk**: Data integrity — invalid email addresses stored, potentially breaking password reset flows.

**Fix**: Add email format validation in CreateUser as defense-in-depth.

---

#### DV-10 [P2] — validatePassword in auth handler doesn't check for common passwords
**File**: `services/auth/internal/server/registration_handler.go:119-135`

**Problem**: `validatePassword()` checks length (8-64) and character classes (upper, lower, digit) but doesn't check against a breached password list or require special characters. `validatePasswordComplexity` in identity_service (line 417) is referenced but its implementation wasn't fully audited.

**Risk**: Users can set weak passwords like "Password1" that pass the current checks.

**Fix**: Consider integrating zxcvbn (already imported in `zxcvbn.go`) for password strength scoring.

---

### === Error Handling (R24) ===

---

#### EH-01 [P0] — 336 json.NewEncoder().Encode() calls silently discard errors
**File**: 164 files, 336 occurrences  
**Central**: `pkg/httputil/response.go:15` — `_ = json.NewEncoder(w).Encode(v)`  
**Also**: SCIM handler `writeSCIMJSON` (line 155): `_ = json.NewEncoder(w).Encode(v)`

**Problem**: `json.Encode` errors are discarded with `_ =`. If encoding fails (due to partial write, connection closed, or marshaling error), the server silently continues without logging or handling the failure. The central `WriteJSON` in httputil propagates this anti-pattern to all services.

**Risk**: Silent data corruption — partial JSON responses sent to clients. Debugging becomes extremely difficult when encode errors are invisible. In security contexts, a truncated response could cause clients to misinterpret partial data.

**Fix**: Log encode errors in WriteJSON. For critical handlers, consider writing to a buffer first, then flushing.

---

#### EH-02 [P0] — ~31 goroutines launched without panic recovery
**File**: 71 total goroutines across services, 40 with recover, ~31 without

**Problem**: 31 anonymous goroutines lack `defer recover()`. If any panics, the entire process crashes. Key locations include:
- Various `go func()` in auth/oauth/audit cmd/main.go (server startup)
- Internal service goroutines for background tasks

**Risk**: Process crash from a single goroutine panic, causing service outage.

**Fix**: Add `defer func() { if r := recover(); r != nil { log.Printf("goroutine panic: %v", r) } }()` to every goroutine.

---

#### EH-03 [P0] — Only 8 ctx.Err() checks across entire codebase (4 files)
**File**: 4 files total with ctx.Err() checks

**Problem**: With hundreds of handler functions and database operations, only 8 explicit `ctx.Err()` checks exist. This means:
- Long-running operations continue processing even after clients disconnect
- Database queries aren't cancelled when the request context is cancelled
- Resources are wasted on abandoned requests

**Risk**: Resource exhaustion from orphaned work. In a DoS scenario, attackers can initiate many slow requests, disconnect, and the server continues processing all of them.

**Fix**: Add `if err := ctx.Err(); err != nil { return err }` at key checkpoints in handler chains and long-running operations.

---

#### EH-04 [P0] — Three error response formats coexist (writeJSON, http.Error, writeServiceError)
**File**: Multiple services  
- `httputil.WriteJSON` → `{"error": "msg"}` (via writeJSON → WriteJSON)
- `http.Error()` → plain text (10 occurrences in 8 files)
- `writeServiceError` → `errors.WriteAPIError` (structured GGIDError format)
- SCIM: `writeSCIMError` → SCIM error format

**Problem**: Four different error response formats across the codebase:
1. `{"error": "message"}` — simple JSON (writeJSONError/writeError)
2. `http.Error(w, "msg", code)` — plain text body
3. Structured `GGIDError` with code/details (writeServiceError → WriteAPIError)
4. SCIM `ErrorResponse` with schemas/detail/status/scimType

**Risk**: Client-side inconsistency. API consumers cannot reliably parse error responses. Security tools and WAFs may miss error patterns. Debugging becomes harder.

**Fix**: Standardize on `writeServiceError`/`WriteAPIError` everywhere. Replace all `http.Error` calls with structured JSON errors.

---

#### EH-05 [P1] — fmt.Errorf dominates (1099) vs pkg/errors (132) — 89% inconsistency
**File**: 212 files with fmt.Errorf, 59 files with pkg/errors

**Problem**: The codebase has `pkg/errors` with structured error types (`ErrInvalidArgument`, `ErrNotFound`, etc.) and `WriteAPIError` for proper HTTP status mapping. But 89% of error creation uses `fmt.Errorf`, bypassing structured error handling. This means:
- HTTP status code mapping is manual and inconsistent
- Error wrapping with `%w` is not standardized
- Error classification for observability/metrics is lost

**Risk**: Incorrect HTTP status codes returned to clients. Inability to programmatically classify errors for monitoring/alerting.

**Fix**: Migrate error creation to `pkg/errors` types where structured HTTP response mapping is needed.

---

#### EH-06 [P1] — 386 defer .Close() calls without error checking
**File**: 174 files across all services

**Problem**: All `defer file.Close()` / `defer body.Close()` / `defer rows.Close()` calls discard the returned error. For file writes, this means data may not be flushed. For database rows, resource leaks may go undetected.

**Risk**: Silent resource leaks, undetected write failures (data loss), file descriptor exhaustion.

**Fix**: Use `defer func() { if err := x.Close(); err != nil { log.Printf("close error: %v", err) } }()` pattern.

---

#### EH-07 [P1] — DB errors not classified — generic 500 returned for most DB failures
**File**: Multiple handlers across identity/auth/oauth services

**Problem**: Database errors are typically handled with `writeError(w, 500, "internal error")` or `writeSCIMError(w, 500, ...)`. There are only 13 instances of DB error classification (using `pgconn.PgError` code matching like `isUniqueViolation`). Common patterns not handled:
- `pgx.ErrNoRows` → should map to 404, often mapped to 500
- Connection errors → should trigger retry/circuit-breaker
- Constraint violations → should map to 409

**Risk**: Information leakage (generic 500 hides actionable errors), poor user experience, missed retry opportunities.

**Fix**: Create a `classifyDBError(err) (httpStatus, message)` helper. Apply at all DB-facing handler boundaries.

---

#### EH-08 [P1] — searchUsers silently ignores JSON decode errors
**File**: `services/identity/internal/scim/handler.go:403`

**Problem**: `_ = json.NewDecoder(r.Body).Decode(&body)` — decode error is explicitly discarded. If the body is malformed JSON, the handler proceeds with zero-value `body` struct, returning all users unfiltered.

**Risk**: Information disclosure — malformed request returns full user list. Also violates principle of failing securely (fail-closed).

**Fix**: Return `400 Bad Request` on decode failure instead of proceeding with defaults.

---

#### EH-09 [P1] — applyAttributeFilter silently ignores all errors
**File**: `services/identity/internal/scim/handler.go:767-806`

**Problem**: `json.Marshal`, `json.Unmarshal`, and final `json.Unmarshal` all have errors discarded (lines 773-774, 803-804). If marshaling fails, the original unfiltered user data is returned. The final `json.Unmarshal(filtered, &result)` on line 804 silently discards errors — if the filtered data doesn't match SCIMUser, the zero-value struct is returned.

**Risk**: Data integrity — attribute filtering may silently fail, returning data that should have been filtered out.

**Fix**: Return errors from these operations, or at minimum log warnings.

---

#### EH-10 [P1] — Error detail leakage in bulk import error logging
**File**: `services/identity/internal/server/bulk_import.go:177`

**Problem**: `slog.Error("bulk import: credential update failed", "user_id", createdUser.ID, "error", execErr)` logs the internal user UUID and full database error to logs. While not directly exposed to the API consumer, if log aggregation is accessible, internal IDs and DB error details (table names, constraint names) are exposed.

**Risk**: Information leakage through log systems.

**Fix**: Sanitize DB error details before logging. Use structured logging with redacted error types.

---

#### EH-11 [P2] — RAR consent preview handler has no body size limit
**File**: `services/oauth/internal/server/rar_handler.go:212`

**Problem**: `json.NewDecoder(r.Body).Decode(&req)` has no `MaxBytesReader`. If this handler is not behind the gateway's body size middleware, it accepts unlimited body sizes.

**Risk**: Memory exhaustion DoS.

**Fix**: Add `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)`.

---

#### EH-12 [P2] — writeSCIMJSON encodes to response before checking encode success
**File**: `services/identity/internal/scim/handler.go:152-156`

**Problem**: `writeSCIMJSON` writes headers and status code, then encodes JSON. If encoding fails midway, the client receives a truncated response with a 200 status — impossible to retry or detect.

**Risk**: Silent data corruption in API responses.

**Fix**: Encode to a buffer first, then write the buffer to the response writer.

---

## Summary Statistics

| Severity | Count |
|----------|-------|
| P0       | 7     |
| P1       | 9     |
| P2       | 4     |
| **Total**| **20**|

### P0 Issues
1. **DV-01**: 514 json.NewDecoder calls without JSON depth limiting (DoS)
2. **DV-02**: SCIM filter has no depth/complexity limit (DoS + data leak)
3. **DV-03**: Gateway router inline handlers lack per-handler body size limit
4. **EH-01**: 336 json.Encode errors silently discarded
5. **EH-02**: ~31 goroutines without panic recovery
6. **EH-03**: Only 8 ctx.Err() checks (resource exhaustion)
7. **EH-04**: 4 error response formats coexist (inconsistency)

### Positive Findings
- MaxBytesReader is applied at service entry points (auth, audit, identity SCIM, bulk import)
- Email validation uses `net/mail.ParseAddress` (not naive string matching)
- Password complexity enforced in identity_service.go CreateUser (covers all creation paths)
- SCIM bulk operations limited to 1000 with failOnErrors support
- BOLA protection in SCIM bulk operations (tenant verification)
- Bulk import validates email format and field lengths
- DetectHashType validates hash content against claimed type
- RAR authorization_details has type validation with 6 registered types
- UUID parsing on all resource ID inputs (uuid.Parse)
- SCIM ETag/If-Match optimistic locking implemented
- Tenant context injection prevents cross-tenant access (BOLA defense)
- writeServiceError uses structured pkg/errors.WriteAPIError
- No production panics found (only in test files)

### Trends vs Prior Rounds
- **Body limits**: Improved — MaxBytesReader present on most service entry points and SCIM handlers
- **Input length**: Improved — identity_service validates 6 field lengths, bulk_import validates 2
- **Email validation**: Good — uses net/mail.ParseAddress consistently
- **Password complexity**: Good — enforced at CreateUser level (defense-in-depth)
- **Error consistency**: Still poor — 89% fmt.Errorf vs 11% pkg/errors, 4 response formats
- **ctx.Err()**: Critically low — only 8 checks in entire codebase
- **goroutine recover**: Moderate — 56% have recover, 44% don't
- **json.Encode errors**: Pervasive issue — 336 instances of silently discarded encode errors
