# GGID Functional Reality Report

**Date**: 2026-07-19  
**Scope**: All 7 services — REAL/STUB/MOCK classification  
**Method**: Source code audit + data flow tracing  

---

## Legend

| Tag | Meaning |
|-----|---------|
| **REAL** | Handler queries real PostgreSQL/Redis data, processes it, returns actual results |
| **STUB** | Handler exists, returns hardcoded/mock data without DB queries |
| **NIL-FALLBACK** | Handler tries DB, but falls back to empty list/static data when pool=nil |
| **WIRED** | Logic implemented but not connected to background processing (no goroutine/subscriber) |

---

## 1. Auth Service

### Core Authentication — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `POST /api/v1/auth/login` | **REAL** | Calls `authSvc.Login()` → `local_provider.go` → PG credential lookup + Argon2id verify |
| `POST /api/v1/auth/register` | **REAL** | Calls `authSvc.Register()` → PG insert + Argon2id hash |
| `POST /api/v1/auth/refresh` | **REAL** | Calls `tokenService.RefreshToken()` → Redis + PG token store |
| `POST /api/v1/auth/logout` | **REAL** | Calls `tokenService.RevokeToken()` → Redis DEL |
| `POST /api/v1/auth/password/change` | **REAL** | Calls `passwordService.ChangePassword()` → PG update + Argon2id |
| `POST /api/v1/auth/password/forgot` | **REAL** | Email service → Redis token store |
| `POST /api/v1/auth/password/reset` | **REAL** | Redis token verify + PG password update |
| `GET /api/v1/auth/sessions` | **REAL** | Redis session scan |
| `POST /api/v1/auth/mfa/enroll` | **REAL** | TOTP secret generation + PG store |
| `POST /api/v1/auth/mfa/verify` | **REAL** | TOTP code verification + PG update |
| `POST /api/v1/auth/key-rotation/rotate` | **REAL** | PG key_rotation_log insert |
| `POST /api/v1/auth/login-attempts/:username` (reset) | **REAL** | Redis counter clear |

### Advanced Auth — MIXED

| Endpoint | Status | Issue |
|----------|--------|-------|
| `POST /api/v1/auth/conditional-access` (KB-080) | **REAL** | PG repo for CAP policies, evaluated during login |
| `GET /api/v1/auth/cae/*` (KB-081) | **REAL** | PG repo for CAE evaluations |
| `GET /api/v1/auth/login-patterns/:user_id` | **STUB** | Returns hardcoded pattern data (line 40) |
| `GET /api/v1/auth/login-geo/enrich` | **STUB** | Returns hardcoded geo data (line 16) |
| `GET /api/v1/auth/impossible-travel` | **STUB** | Returns hardcoded travel data (line 50) |
| `GET /api/v1/auth/device-attest/:user_id` | **STUB** | Returns hardcoded attestation (line 26) |
| `GET /api/v1/auth/dlp/policies` | **NIL-FALLBACK** | Tries PG, returns empty list if nil pool |
| `GET /api/v1/auth/session-limits` | **NIL-FALLBACK** | Tries PG, returns defaults if nil pool |
| `GET /api/v1/auth/device-fingerprints/:user_id` | **NIL-FALLBACK** | Tries PG, returns empty if nil pool |
| `POST /api/v1/auth/batch3c/*` (4 endpoints) | **STUB** | Returns hardcoded status objects |

---

## 2. Identity Service

### Core User Management — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `GET /api/v1/users` (list) | **REAL** | PG `SELECT ... FROM users WHERE tenant_id` with pagination |
| `POST /api/v1/users` (create) | **REAL** | PG INSERT + Argon2id hash |
| `GET /api/v1/users/:id` | **REAL** | PG SELECT by UUID |
| `PUT /api/v1/users/:id` | **REAL** | PG UPDATE |
| `DELETE /api/v1/users/:id` | **REAL** | PG soft delete (deleted_at) |
| `POST /api/v1/users/import` (CSV) | **REAL** | PG batch insert |
| `POST /api/v1/users/:id/lock` | **REAL** | PG status update |
| SCIM `/scim/v2/Users` | **REAL** | PG-backed with SCIM filter parsing |
| SCIM `/scim/v2/Groups` | **STUB** | In-memory store with mock data (groups.go:188) |

### Identity Advanced — STUB

| Endpoint | Status | Issue |
|----------|--------|-------|
| `GET /api/v1/identity/mdm/connectors` | **STUB** | Returns empty list `[]MDMConnector{}` |
| `GET /api/v1/identity/mdm/devices` | **STUB** | Returns empty list `[]MDMDevice{}` |
| `GET /api/v1/identity/rebac/*` (3 endpoints) | **NIL-FALLBACK** | Tries PG, returns empty |
| `GET /api/v1/identity/jml/rules` | **NIL-FALLBACK** | Tries PG, returns empty |
| `GET /api/v1/identity/attestations` | **NIL-FALLBACK** | Tries PG, returns stored or empty |
| `GET /api/v1/identity/dlp/*` (3 endpoints) | **NIL-FALLBACK** | Tries PG, returns empty |
| `GET /api/v1/identity/secrets/*` | **STUB** | In-memory map, not real secret store |
| `GET /api/v1/identity/settings/delegations` | **STUB** | Returns hardcoded empty array |
| `GET /api/v1/identity/settings/saml-metadata` | **STUB** | Returns hardcoded metadata |
| `POST /api/v1/identity/settings/rotate-key` | **STUB** | Returns hardcoded `{"status":"rotated","new_kid":"key-2"}` |

---

## 3. OAuth Service

### Core OAuth — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `POST /oauth/token` | **REAL** | Full grant_type handling, client verify (Argon2id), JWT signing |
| `GET /oauth/authorize` | **REAL** | Authorization code flow with PKCE |
| `POST /api/v1/oauth/clients` | **REAL** | PG client registration + Argon2id secret hash |
| `GET /api/v1/oauth/clients` | **REAL** | PG list |
| `POST /api/v1/oauth/clients/:id/rotate-secret` | **REAL** | Argon2id verify old + hash new |
| `POST /oauth/revoke` | **REAL** | Redis token blocklist |
| `POST /oauth/introspect` | **REAL** | JWT parse + client auth |
| `GET /.well-known/jwks.json` | **REAL** | RSA public key export |
| `GET /.well-known/openid-configuration` | **REAL** | Dynamic discovery doc |
| `POST /api/v1/agents/register` | **REAL** | Redis-backed agent store |
| `POST /api/v1/agents/token` | **REAL** | RFC 8693 token exchange |
| `POST /api/v1/agents/:id/scopes` | **REAL** | Scope validation + Redis store (KB-335) |

### OAuth Advanced — STUB

| Endpoint | Status | Issue |
|----------|--------|-------|
| `GET /api/v1/oauth/validate-client-secret` | **REAL** | Entropy calculation (functional) |
| `POST /api/v1/oauth/clients/:id/lifecycle` | **STUB** | In-memory state, no real lifecycle processing |
| `POST /api/v1/oauth/client-onboarding` | **STUB** | Generates random secret, does NOT persist to DB |

---

## 4. Policy Service

### Core RBAC — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `GET /api/v1/roles` | **REAL** | PG SELECT roles |
| `POST /api/v1/roles` | **REAL** | PG INSERT + audit event |
| `POST /api/v1/roles/assign` | **REAL** | PG INSERT + `role.assigned` audit event (KB-330) |
| `DELETE /api/v1/roles/:id` | **REAL** | PG DELETE + audit |
| `GET /api/v1/policies` | **REAL** | PG SELECT policies |
| `POST /api/v1/policies` | **REAL** | PG INSERT + audit |
| `DELETE /api/v1/policies/:id` | **REAL** | PG DELETE |
| `GET /api/v1/permissions` | **REAL** | PG SELECT permissions |

### Policy Advanced — STUB

| Endpoint | Status | Issue |
|----------|--------|-------|
| `GET /api/v1/policies/standing-access` | **STUB** | Hardcoded access list (line 18) |
| `GET /api/v1/policies/impact-preview` | **STUB** | Hardcoded impact data (line 35) |
| `GET /api/v1/policies/effectiveness` | **STUB** | Hardcoded metrics (line 19) |
| `GET /api/v1/policies/emergency-access/audit` | **STUB** | Hardcoded break-glass events |
| `POST /api/v1/policies/jit/elevate` | **STUB** | Returns fake token, no real elevation |
| `GET /api/v1/policies/role-mining` | **STUB** | Hardcoded mining results |
| `GET /api/v1/policies/blast-radius` | **STUB** | Hardcoded blast radius data |
| `POST /api/v1/policies/abac/import` | **STUB** | In-memory only |
| `GET /api/v1/policies/delegation/validate` | **STUB** | Returns hardcoded validation result |
| `GET /api/v1/policies/sod-conflicts` | **STUB** | Hardcoded SoD conflicts |

---

## 5. Org Service

### Core Org — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `GET /api/v1/orgs` | **REAL** | PG SELECT |
| `POST /api/v1/orgs` | **REAL** | PG INSERT |
| `GET /api/v1/orgs/:id/tree` | **REAL** | PG LTREE query |
| `POST /api/v1/orgs/:id/departments` | **REAL** | PG INSERT |

---

## 6. Audit Service

### Core Audit — REAL

| Endpoint | Status | Evidence |
|----------|--------|----------|
| `GET /api/v1/audit` | **REAL** | PG SELECT from audit_events with filters |
| `GET /api/v1/audit/stats` | **REAL** | PG aggregate queries |
| `GET /api/v1/audit/export` | **REAL** | PG query + CSV generation |
| `GET /api/v1/audit/stream` (SSE) | **REAL** | PG poll + SSE push |

### ITDR Detection — REAL (NATS-wired, processing events)

| Component | Status | Evidence |
|-----------|--------|----------|
| NATS consumer wiring | **REAL** | `nats_consumer.go:174`: `c.engine.Evaluate(ctx, &event)` called on every persisted event |
| Detection Engine (`engine.go`) | **REAL** | 15 rules, all with real Evaluate() logic, wired in main.go:104-126 |
| BruteForceRule | **REAL** | Tracks failed login count via StateStore |
| ImpossibleTravelRule | **REAL** | Geo-distance + time calculation |
| CredentialStuffingRule | **REAL** | Pattern matching across IPs |
| OffHoursAdminRule | **REAL** | Time-window + role check |
| TokenReplayRule | **REAL** | jti tracking |
| 10 additional rules (KB-192) | **REAL** | All have Evaluate() implementations |
| ITDR → CAE callback | **REAL** | `nats_consumer.go:188`: critical detections publish `ggid.session.revoke` |
| StateStore | **REAL** | Redis-backed (multi-replica safe) or in-memory fallback |
| ITDR persistence | **REAL** | Detections stored in PG via `ITDRRepository` |

**Correction**: ITDR is fully wired. The NATS consumer (`nats_consumer.go`) subscribes to audit events, persists them, then calls `engine.Evaluate()` on each event. Critical detections trigger session revocation via NATS callback. This is a real-time threat detection pipeline.

### CCM Engine — STUB (hardcoded metrics, real persistence)

| Component | Status | Evidence |
|-----------|--------|----------|
| `CCMEngine.RunAll()` | **STUB** | 15 controls with hardcoded metric values. Code comment (line 119): "creates a CCMResult with simulated metric values. In production, these would query real data sources." |
| CCM persistence | **REAL** | Results persisted to PG `ccm_results` table when pool configured |
| CCM history retrieval | **REAL** | PG SELECT from ccm_results |
| CCM summary/reporting | **REAL** | Aggregates stored results |

**Gap**: CCM has 15 well-defined controls with proper names, categories, and thresholds. However, `evalControl()` uses hardcoded metric values instead of querying real data sources. The POST /ccm/run endpoint works (creates results + persists), but the metrics are not real. Results will show "pass" or "fail" based on hardcoded numbers, not actual security posture.

### CAE (Continuous Access Evaluation) — PARTIAL (manual trigger, no background sweep)

| Component | Status | Evidence |
|-----------|--------|----------|
| CAE handler `POST /cae/run` | **STUB** | Records a synthetic evaluation (line 82: "manual-sweep"), does NOT scan active sessions table |
| CAE status `GET /cae/status` | **REAL** | PG COUNT from cae_evaluations |
| CAE log `GET /cae/log` | **REAL** | PG SELECT from cae_evaluations |
| `EvaluateSessionForCAE()` | **REAL** | Evaluates single session against CAP policies (programmatic API) |
| CAP policy evaluation | **REAL** | `capRepo.Evaluate()` checks IP/risk/auth-method conditions |
| Background goroutine | **MISSING** | No ticker/goroutine for periodic session sweep |
| Session table scan | **MISSING** | `caeRun` does not query `sessions` table |

**Gap**: CAE has the evaluation logic (`EvaluateSessionForCAE` is real), but the POST /cae/run endpoint doesn't actually scan the sessions table — it just records a synthetic "manual-sweep" evaluation. There is no background goroutine. The infrastructure exists but the active session scanning is not implemented.

---

## 7. Gateway Service

### Core Gateway — REAL

| Feature | Status | Evidence |
|----------|--------|----------|
| Reverse proxy | **REAL** | httputil.ReverseProxy with route table |
| JWT validation | **REAL** | RS256 + JWKS cache |
| Rate limiting | **REAL** | Fixed-window + token bucket + 5-dimensional |
| CORS | **REAL** | Per-tenant origin validation |
| Session middleware | **REAL** | JWT parse + tenant context injection |
| List cache (KB-325) | **REAL** | Redis-backed GET list endpoint cache |
| Token endpoint limit (KB-326) | **REAL** | 10/min on /oauth/token |
| Metrics | **REAL** | Prometheus histograms + counters |
| OpenAPI spec | **REAL** | Dynamic generation with 47 schema-rich paths |

---

## Summary

### By Service

| Service | REAL | STUB | NIL-FALLBACK | Total Endpoints |
|---------|------|------|-------------|-----------------|
| Auth | 15 | 8 | 4 | 27 |
| Identity | 9 | 8 | 6 | 23 |
| OAuth | 12 | 2 | 0 | 14 |
| Policy | 8 | 10 | 0 | 18 |
| Org | 4 | 0 | 0 | 4 |
| Audit | 4+ITDR | 1 (CCM) | 0 | 5 |
| Gateway | 9 | 0 | 0 | 9 |
| **Total** | **63** | **29** | **10** | **100** |

### Severity Assessment

| Category | Count | Impact |
|----------|-------|--------|
| **Critical (core CRUD)** | 0 | All core CRUD endpoints are REAL |
| **High (security features)** | 2 | CCM metrics hardcoded, CAE no background sweep |
| **Medium (analytics/advisory)** | 29 | STUB endpoints return realistic-looking demo data |
| **Low (NIL-fALLBACK)** | 10 | Graceful empty responses when DB not configured |

### Top 5 Priorities for v1.1

1. **CCM real queries**: Replace hardcoded metrics in `evalControl()` with actual DB queries
2. **CAE background sweep**: Add goroutine for periodic active session evaluation
4. **SCIM Groups**: Replace in-memory mock with PG-backed store
5. **Policy advanced**: Wire standing-access, impact-preview, role-mining to real data queries
