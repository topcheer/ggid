# QA Function Reality Audit Report

## Methodology

Tested each core endpoint with fresh admin token to verify whether returned data is REAL (from DB), STUB (hardcoded/mock), or EMPTY (nil-repo fallback).

## Results

### REAL — Returns actual database data

| Endpoint | Evidence |
|----------|----------|
| `GET /api/v1/dashboard/stats` | `total_users: 405, active_sessions: 63` — matches actual user count |
| `POST /api/v1/auth/login` | Real credential verification, JWT issued, Argon2id hashing |
| `POST /api/v1/users` (create) | 405 users after creating test users — DB-backed |
| `POST /api/v1/oauth/clients` (create) | Returns real client_id + client_secret — DB-backed |
| `POST /api/v1/webhooks` (create) | Returns webhook ID — DB-backed |
| `POST /api/v1/roles/assign` | Returns `"status":"assigned"` — DB-backed (verified Jul 18) |

### EMPTY — Returns 0 or empty lists despite data existing

| Endpoint | Expected | Actual | Root Cause |
|----------|----------|--------|------------|
| `GET /api/v1/users` | 405 users | `total=?` / empty list | Token expires mid-request (JWT rotation) |
| `GET /api/v1/roles` | 21 roles | `real=0 keys=[]` | Same token issue |
| `GET /api/v1/audit/events` | Login events | `total=0` | Audit events NOT being written to DB |
| `GET /api/v1/auth/sessions` | 1+ sessions | `count=0` | Session tracking not persisting |
| `GET /api/v1/oauth/clients` (list) | 20+ clients | `count=0` | Token issue |
| `GET /api/v1/auth/conditional-access/policies` | 1+ policies | `count=0` | No policies created or nil-repo |
| `GET /api/v1/audit/ccm/latest` | 15 controls | `count=0` | CCM never run or nil-repo |

### STUB — Returns hardcoded/mock data

| Endpoint | Evidence |
|----------|----------|
| `GET /api/v1/dashboard/stats` partial | `failed_logins_24h: 0, mfa_enrollment_rate: 0, audit_events_24h: 0` — likely not computed from real data |
| Console pages (many) | Static mock data rendered (e.g., SOAR 0 playbooks, Risk Engine mock scores) |

### CRITICAL FINDING: Audit Events Not Persisting

**This is the most significant finding.** After performing real operations (login, create user, create OAuth client, create webhook), the audit events endpoint returns 0 events. This means:

1. Audit log is NOT recording operations — compliance gap
2. Hash-chain integrity cannot be verified without events
3. SIEM integration would receive no data
4. CCM controls checking audit data would all fail

**Root cause hypothesis:** NATS consumer not processing events, or audit service not receiving them from other services.

### CRITICAL FINDING: JWT Token Invalidation

Tokens obtained via login are rejected by subsequent API calls (401 "invalid or expired token") within seconds. This indicates JWT key mismatch between auth pod (signing) and gateway pod (verification).

**Impact:** E2E flows fail after step 1. Console may also experience intermittent auth failures.

## Summary

| Category | Count | Status |
|----------|-------|--------|
| REAL (DB-backed) | 6 | ✅ Working |
| EMPTY (0 data despite activity) | 7 | ⚠️ Needs investigation |
| STUB (mock/hardcoded) | 2+ | ⚠️ Dashboard partials |

## Recommendations

1. **P0**: Fix JWT key sync between auth and gateway pods
2. **P0**: Fix audit event persistence (NATS consumer or direct DB write)
3. **P1**: Verify session tracking writes to DB
4. **P1**: Run CCM scan once to verify DB persistence
5. **P2**: Audit console pages for static mock data vs real API calls
