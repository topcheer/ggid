# DLP Egress Control & PII Redaction: Response-Level Data Loss Prevention for GGID

> **Focus**: A gateway-level egress middleware that inspects API responses before they reach clients — detecting PII (SSN, credit cards, emails, phones), applying redaction/masking/tokenization rules based on user role and data classification, and auditing every redaction action. Extends GGID's existing DLP policy engine (EvaluateDLP) from request-time evaluation to response-time enforcement.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§10), curl commands (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: DLP Policy Engine](#2-ggid-current-state-dlp-policy-engine)
3. [Gap Analysis](#3-gap-analysis)
4. [Egress DLP Architecture](#4-egress-dlp-architecture)
5. [PII Detection Patterns](#5-pii-detection-patterns)
6. [Redaction Strategies](#6-redaction-strategies)
7. [Proposed Architecture: Egress Middleware](#7-proposed-architecture-egress-middleware)
8. [Endpoint Precondition Check](#8-endpoint-precondition-check)
9. [API Design + Curl Commands](#9-api-design--curl-commands)
10. [Database Schema](#10-database-schema)
11. [Implementation Backlog with DoD](#11-implementation-backlog-with-dod)
12. [Competitive Differentiation](#12-competitive-differentiation)
13. [Security Considerations](#13-security-considerations)

---

## 1. Executive Summary

GGID has a working DLP policy engine (`identity/server/dlp_handler.go:157` — `EvaluateDLP`) with:
- DB-backed policy CRUD (PostgreSQL `dlp_policies` + `dlp_events` tables) ✅
- Data classification integration (`$data.classification` conditions) ✅
- Policy evaluation (block / mask / log based on role + classification) ✅
- DLP event logging ✅
- Policy test/simulation API ✅

However, EvaluateDLP runs at **request time** — it checks whether a user is *allowed* to perform an action (export, download, API call). It does **not** inspect the actual response data leaving GGID. A user with legitimate "users:read" access could receive a response containing SSNs, credit cards, or other PII that should be masked — and the current DLP engine wouldn't catch it.

**The missing piece**: An **egress middleware** that sits between the backend service and the HTTP client, inspects every API response body, applies PII detection + redaction rules, and modifies the response before it reaches the client.

**Recommendation**: Build a gateway-level egress DLP middleware with PII detection (regex + classification), redaction engine (mask/partial/tokenize/hash), policy DSL ("if response contains ssn AND role != admin THEN mask"), and full audit logging.

**Estimated effort**: 2 sprints for MVP (middleware + detection + redaction + audit).

---

## 2. GGID Current State: DLP Policy Engine

### Existing Components

| Component | File:Line | Status | Capability |
|-----------|-----------|--------|------------|
| DLPPolicy struct | `dlp_handler.go:17` | **DB-backed** ✅ | Policy with conditions + action |
| DLPEvent struct | `dlp_handler.go:31` | **DB-backed** ✅ | Enforcement event log |
| EvaluateDLP | `dlp_handler.go:157` | **Works** ✅ | Request-time evaluation |
| dlpRepo | `dlp_handler.go:54` | **PostgreSQL** ✅ | Policy + event persistence |
| DLP handler | `dlp_handler.go:213` | **Works** ✅ | CRUD + test + events + heatmap |
| Data classification | `data_gov_repo.go:14` | **DB-backed** ✅ | Classification labels |
| LookupClassification | `data_gov_repo.go:137` | **Works** ✅ | Per-resource classification |
| PII logging (auth) | `auth/service/pii_logging.go:7` | **Works** ✅ | Masks PII in logs |
| PII logging (oauth) | `oauth/service/pii_logging.go:7` | **Works** ✅ | Masks PII in logs |
| DLP policies (auth) | `auth/server/dlp_policies_handler.go:26` | **Hardcoded** ❌ | Mock policies |
| obfuscateEventPII | `audit/service/audit_service.go:79` | **Works** ✅ | Audit event PII masking |

### What EvaluateDLP Does (Today)

```go
// dlp_handler.go:157
func EvaluateDLP(policies []*DLPPolicy, trigger, resourceType, dataClassification, userRole string) *DLPTestResult {
    // Checks: should this user be ALLOWED to perform this action on this data?
    // Returns: allow, block, mask, log
    // BUT: does NOT inspect actual response data
}
```

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No egress middleware** | API responses sent to client without inspection |
| 2 | **No PII detection** | SSNs, credit cards, emails in responses not detected |
| 3 | **No field-level redaction** | No masking of sensitive fields in JSON responses |
| 4 | **No classification-driven masking** | Classification exists but not applied to responses |
| 5 | **No redaction audit** | No record of what was masked when/for whom |
| 6 | **Auth DLP policies hardcoded** | `dlp_policies_handler.go:26` returns mock data |

---

## 3. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "API response contains SSN 123-45-6789" | Sent as-is to client | Masked: \*\*\*-\*\*-6789 |
| 2 | "Non-admin views user list with email addresses" | All emails visible | Emails masked for non-admins |
| 3 | "Credit card number in audit export" | Full CC sent | Redacted: \*\*\*\*\*\*\*\*\*\*\*\*4242 |
| 4 | "Phone numbers in user profile API" | Full phone sent | Partial mask: +86 138\*\*\*\*5678 |
| 5 | "DLP policy: mask all 'important' data for viewers" | Policy exists but not enforced on egress | Egress middleware enforces |

---

## 4. Egress DLP Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │         Egress DLP Middleware                 │
                    │         (Gateway Layer)                       │
                    │                                              │
     Client  ──────▶│  1. Request arrives                          │
                    │  2. JWT auth → user_id, role, tenant_id       │
                    │  3. Proxy to backend service                 │
                    │  4. Receive response                         │
                    │  5. ★ EGRESS INSPECTION:                     │
                    │     a. Parse response body (JSON)            │
                    │     b. For each field: check classification  │
                    │     c. Run PII detection (regex patterns)     │
                    │     d. Apply redaction rules by policy       │
                    │     e. Rebuild response body                 │
                    │  6. Send modified response to client         │
                    │  7. Audit: log redaction actions             │
                    └──────────────────────────────────────────────┘
```

### Inspection Flow

```
Backend response:
{
  "user": {
    "name": "Alice Chen",
    "email": "alice@corp.com",          ← PII: email
    "ssn": "123-45-6789",               ← PII: SSN
    "phone": "+1-415-555-0123",         ← PII: phone
    "credit_card": "4242-4242-4242-4242", ← PII: credit card
    "department": "engineering",        ← Not PII
    "salary": 120000                    ← PII: financial
  }
}

Egress middleware (viewer role):
{
  "user": {
    "name": "Alice Chen",
    "email": "a***@corp.com",           ← Partial mask
    "ssn": "***-**-6789",              ← Partial mask (last 4)
    "phone": "+1-415-***-0123",        ← Partial mask
    "credit_card": "************4242",  ← Partial mask (last 4)
    "department": "engineering",        ← Unchanged
    "salary": "***"                     ← Full mask (financial)
  }
}

Egress middleware (admin role):
{
  "user": {
    "name": "Alice Chen",               ← Unchanged (admin can see all)
    "email": "alice@corp.com",
    ...
  }
}
```

---

## 5. PII Detection Patterns

### Pattern Library

| PII Type | Detection Method | Regex Example | Confidence |
|----------|-----------------|---------------|-----------|
| **SSN (US)** | Regex + validation | `\d{3}-\d{2}-\d{4}` | High (with area validation) |
| **Credit Card** | Regex + Luhn check | `\d{13,19}` | High (Luhn validation) |
| **Email** | RFC 5322 regex | `[a-z0-9]+@[a-z]+\.[a-z]+` | Very High |
| **Phone (US)** | Regex | `\+?1?\d{10}` | Medium |
| **Phone (CN)** | Regex | `\+86\d{11}` | High |
| **IP Address** | Regex | `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}` | High |
| **Bank Account (CN)** | Regex | `\d{16,19}` | Medium |
| **ID Card (CN)** | Regex + checksum | `\d{17}[\dX]` | High (with checksum) |
| **API Keys** | Prefix + pattern | `ggid_[a-zA-Z0-9]{32}` | Very High |
| **JWT Tokens** | Structure | `eyJ[a-zA-Z0-9]+\.[a-zA-Z0-9]+\.[a-zA-Z0-9]+` | Very High |

### Classification-Driven Detection (Existing)

GGID's `data_classifications` table (`data_gov_repo.go:57`) already labels resources:
- `core` — Most sensitive (PII, financial, health)
- `important` — Business critical
- `general` — Normal data

The egress middleware maps classification → default redaction:
- `core` → full mask (unless admin)
- `important` → partial mask
- `general` → no mask

---

## 6. Redaction Strategies

| Strategy | Example | Reversible? | Use Case |
|----------|---------|-------------|----------|
| **Full mask** | `***` | No | Financial data for non-admin |
| **Partial mask** | `***-**-6789` | No | SSN (last 4 for verification) |
| **Email mask** | `a***@corp.com` | No | Email addresses |
| **Tokenization** | `tok_abc123` | Yes (via vault) | Reversible field-level encryption |
| **Hashing** | `sha256:abc...` | No | Irreversible reference |
| **Redact (remove)** | (field removed) | No | Fields user shouldn't see at all |
| **Bucket** | `100K-150K` | No | Salary range instead of exact |

### Policy DSL

```yaml
# DLP Egress Policy (declarative)
rules:
  - name: "mask-ssn-non-admin"
    condition:
      field_pattern: "ssn|social_security"
      user_role_not_in: ["admin", "hr"]
    action: "partial_mask"
    params: { keep_last: 4, mask_char: "*" }

  - name: "mask-credit-card"
    condition:
      pii_type: "credit_card"
      user_role_not_in: ["admin"]
    action: "partial_mask"
    params: { keep_last: 4, mask_char: "*" }

  - name: "mask-email-viewer"
    condition:
      field_pattern: "email|mail"
      user_role: "viewer"
    action: "email_mask"

  - name: "redact-salary-non-hr"
    condition:
      field_pattern: "salary|compensation"
      user_role_not_in: ["hr", "admin"]
    action: "redact"

  - name: "tokenize-api-keys"
    condition:
      pii_type: "api_key"
    action: "tokenize"
```

---

## 7. Proposed Architecture: Egress Middleware

```go
// gateway/internal/middleware/dlp_egress.go

type DLPEgressMiddleware struct {
    policyRepo    DLPPolicyReader
    detector      PIIDetector
    redactor      RedactionEngine
    auditWriter   AuditWriter
    redisCache    *redis.Client   // Cache classification lookups
}

func (m *DLPEgressMiddleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Capture response with custom ResponseWriter
        rw := &capturingResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
        next.ServeHTTP(rw, r)

        // 2. Skip non-JSON responses
        ct := w.Header().Get("Content-Type")
        if !strings.Contains(ct, "application/json") {
            w.Write(rw.buf.Bytes())
            return
        }

        // 3. Parse JSON body
        var data map[string]any
        if err := json.Unmarshal(rw.buf.Bytes(), &data); err != nil {
            w.Write(rw.buf.Bytes()) // Can't parse → pass through
            return
        }

        // 4. Get user context
        userRole := getUserRole(r.Context())
        tenantID := getTenantID(r.Context())

        // 5. Get applicable DLP policies
        policies := m.getApplicablePolicies(tenantID, r.URL.Path)

        // 6. Apply redaction
        actions := m.redactor.Apply(data, policies, userRole, r.URL.Path)

        // 7. Rebuild response
        modified, _ := json.Marshal(data)

        // 8. Update content length
        w.Header().Set("Content-Length", strconv.Itoa(len(modified)))
        w.Write(modified)

        // 9. Audit redaction actions (async)
        if len(actions) > 0 {
            go m.auditWriter.LogRedactions(tenantID, getUserID(r.Context()), r.URL.Path, actions)
        }
    })
}
```

### Nested Object Traversal

```go
func (e *RedactionEngine) applyToNode(node any, policies []EgressPolicy, role, path string) []RedactionAction {
    var actions []RedactionAction

    switch v := node.(type) {
    case map[string]any:
        for key, val := range v {
            fieldPath := path + "." + key
            if matched, action := e.matchPolicy(fieldPath, val, policies, role); matched {
                v[key] = action.RedactedValue
                actions = append(actions, action)
            }
            // Recurse into nested objects
            actions = append(actions, e.applyToNode(val, policies, role, fieldPath)...)
        }
    case []any:
        for i, item := range v {
            actions = append(actions, e.applyToNode(item, policies, role, fmt.Sprintf("%s[%d]", path, i))...)
        }
    }
    return actions
}
```

---

## 8. Endpoint Precondition Check

### Existing Endpoints (Reuse)

| Endpoint | File:Line | Status | Reuse |
|----------|-----------|--------|-------|
| `GET/POST/PUT/DELETE /api/v1/identity/dlp/policies` | `identity/http.go:281` | **DB-backed** ✅ | Policy CRUD |
| `GET /api/v1/identity/dlp/events` | `identity/http.go:283` | **DB-backed** ✅ | Event audit |
| `GET /api/v1/identity/dlp/heatmap` | `identity/http.go:284` | **Works** ✅ | Analytics |
| `POST /api/v1/identity/dlp/test` | `dlp_handler.go:300` | **Works** ✅ | Policy simulation |
| Data classification CRUD | `data_gov_handler.go` | **DB-backed** ✅ | Classification labels |
| LookupClassification | `data_gov_repo.go:137` | **Works** ✅ | Per-resource lookup |

### New Components Required

| Component | Purpose | Priority |
|-----------|---------|----------|
| Egress middleware | Response inspection at gateway | P0 |
| PIIDetector | Pattern matching engine | P0 |
| RedactionEngine | Mask/tokenize/hash field values | P0 |
| Egress redaction audit | Log every redaction action | P0 |
| Egress policy API | Separate from request-time DLP policies | P1 |
| Replace auth DLP mock | `auth/dlp_policies_handler.go:26` hardcoded | P1 |

---

## 9. API Design + Curl Commands

### Test Egress Redaction

```bash
# Request as viewer (SSN will be masked)
curl https://ggid.corp.com/api/v1/identity/users/alice \
  -H "Authorization: Bearer $VIEWER_TOKEN"

# Response (before egress):
# { "ssn": "123-45-6789", "email": "alice@corp.com", "salary": 120000 }

# Response (after egress middleware):
{
  "ssn": "***-**-6789",
  "email": "a***@corp.com",
  "salary": "***"
}

# Same request as admin (no masking):
curl https://ggid.corp.com/api/v1/identity/users/alice \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# Response unchanged — admin sees full data
```

### Configure Egress Policy

```bash
curl -X POST https://ggid.corp.com/api/v1/identity/dlp/egress-policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "mask-pii-for-viewers",
    "path_pattern": "/api/v1/identity/users",
    "rules": [
      { "field": "ssn", "action": "partial_mask", "params": {"keep_last": 4}, "applies_to_roles": ["viewer", "developer"] },
      { "field": "email", "action": "email_mask", "applies_to_roles": ["viewer"] },
      { "field": "salary", "action": "full_mask", "applies_to_roles": ["viewer", "developer"] }
    ],
    "enabled": true
  }'
```

### Query Redaction Audit

```bash
curl "https://ggid.corp.com/api/v1/identity/dlp/egress-audit?limit=50" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "redactions": [
    {
      "timestamp": "2026-07-17T10:05:00Z",
      "user_id": "uuid-viewer",
      "path": "/api/v1/identity/users/alice",
      "field": "ssn",
      "action": "partial_mask",
      "original_hash": "sha256:abc...",  // Hash of original value (not the value itself)
      "policy_name": "mask-pii-for-viewers"
    }
  ],
  "total": 15420
}
```

---

## 10. Database Schema

```sql
-- Egress DLP policies (separate from request-time policies)
CREATE TABLE dlp_egress_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,

    -- Matching
    path_pattern        VARCHAR(512),                 -- '/api/v1/identity/users' or glob
    method              VARCHAR(8),                   -- 'GET', '*' (only GET makes sense for egress)

    -- Rules
    rules               JSONB NOT NULL,               -- [{field, action, params, applies_to_roles}]

    -- State
    enabled             BOOLEAN DEFAULT true,
    priority            INT DEFAULT 100,              -- Lower = higher priority

    -- Audit
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Egress redaction audit log
CREATE TABLE dlp_egress_audit (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    user_role           VARCHAR(32),

    -- Request context
    request_path        VARCHAR(512) NOT NULL,
    method              VARCHAR(8),

    -- Redaction details
    field_path          VARCHAR(512) NOT NULL,        -- 'user.ssn'
    action              VARCHAR(32) NOT NULL,         -- 'partial_mask', 'full_mask', 'redact', 'tokenize'
    pii_type            VARCHAR(32),                  -- 'ssn', 'credit_card', 'email', 'phone'
    policy_name         VARCHAR(128),
    original_hash       VARCHAR(64),                  -- SHA-256 hash of original value

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_egress_policies_tenant ON dlp_egress_policies (tenant_id, enabled, priority);
CREATE INDEX idx_egress_audit_tenant_time ON dlp_egress_audit (tenant_id, created_at DESC);
CREATE INDEX idx_egress_audit_user ON dlp_egress_audit (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_egress_audit_field ON dlp_egress_audit (tenant_id, field_path, created_at DESC);
```

---

## 11. Implementation Backlog with DoD

### P0 — Egress Middleware + Detection + Redaction (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Egress DLP DB schema | ✅ CREATE TABLE in migration ✅ go build PASS | 1d |
| 2 | PIIDetector (regex + classification) | ✅ Detects SSN/CC/email/phone/API key/JWT ✅ Classification lookup ✅ ≥3 tests | 3d |
| 3 | RedactionEngine (mask/partial/tokenize/redact) | ✅ 5 redaction strategies ✅ Nested object traversal ✅ ≥3 tests | 3d |
| 4 | Gateway egress middleware | ✅ Intercepts JSON responses ✅ Applies redaction ✅ Rebuilds response ✅ ≥3 tests | 3d |
| 5 | Egress audit (async via NATS) | ✅ Every redaction logged ✅ Original value hashed (not stored) ✅ ≥3 tests | 2d |
| 6 | Egress policy API + test endpoint | ✅ CRUD for policies ✅ POST /dlp/egress-test (dry-run) ✅ curl PASS ✅ ≥3 tests | 2d |

### P1 — Policy DSL + Replace Mocks (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Declarative policy DSL | ✅ YAML/JSON rule format ✅ Path pattern matching ✅ Role-based application ✅ ≥3 tests | 3d |
| 8 | Replace auth DLP mock | ✅ `auth/dlp_policies_handler.go:26` uses real data ✅ No hardcoded ✅ ≥3 tests | 1d |
| 9 | Classification-driven auto-masking | ✅ `core` data auto-masked for non-admin ✅ `important` auto-partial ✅ ≥3 tests | 2d |
| 10 | Egress analytics dashboard | ✅ Redaction count by field/type/role ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 11 | NER-based PII detection | Named entity recognition for unstructured PII |
| 12 | Tokenization vault | Reversible tokenization with CMK-backed vault |
| 13 | Per-field encryption at source | Encrypt PII before storing in PostgreSQL |
| 14 | Binary content inspection | Scan file downloads (PDF, images) for PII |
| 15 | WebSocket DLP | Inspect WebSocket messages for PII |

---

## 12. Competitive Differentiation

| Feature | GGID (target) | Google DLP API | AWS Macie | Microsoft Purview | Apache Ranger |
|---------|---------------|----------------|-----------|-------------------|---------------|
| **Egress inspection** | **Gateway middleware** | API-based | S3 scan | Endpoint DLP | Policy engine |
| **PII detection** | **Regex + classification** | ML + regex | ML | ML + regex | Column tags |
| **Redaction** | **5 strategies** | De-identify | Detect only | Mask + encrypt | Mask |
| **Real-time** | **Inline (<2ms)** | API call (50ms+) | Batch | Inline | Inline |
| **Policy DSL** | **Declarative JSON** | API config | Rules | DLP policies | Ranger policies |
| **Classification** | **DB-backed labels** | Auto-classify | Auto-classify | Auto-classify | Column tags |
| **Audit trail** | **DB-backed** | Cloud Logging | CloudTrail | Audit log | Audit log |
| **Open source** | **Yes** | No | No | No | Yes |

---

## 13. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Latency impact** | Regex compiled + cached; classification cached in Redis (5s TTL) |
| **Original value leakage** | Audit stores only SHA-256 hash, never the original value |
| **Policy bypass** | Egress middleware at gateway — no bypass possible (all traffic flows through) |
| **Performance on large responses** | Stream parsing for >1MB responses; skip binary content |
| **False positives** | Confidence levels; low-confidence matches logged not masked |
| **Admin privilege abuse** | Admin actions still audited; alerts on bulk PII access |

---

## References

- [Google Cloud DLP API](https://cloud.google.com/dlp/docs) — De-identification patterns
- [AWS Macie](https://aws.amazon.com/macie/) — PII discovery and protection
- [Microsoft Purview DLP](https://learn.microsoft.com/en-us/purview/dlp-learn-about-dlp) — Endpoint DLP
- [Apache Ranger](https://ranger.apache.org/) — Policy-based masking
- [NIST SP 800-122](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-122.pdf) — PII protection guide
- [GGID DLP Handler](../services/identity/internal/server/dlp_handler.go) — EvaluateDLP at line 157
- [GGID DLP Repo](../services/identity/internal/server/dlp_handler.go) — DB-backed at line 54
- [GGID Data Classification](../services/identity/internal/server/data_gov_repo.go) — Classification at line 14
- [GGID PII Logging (auth)](../services/auth/internal/service/pii_logging.go) — Log masking at line 7
- [GGID Auth DLP (hardcoded)](../services/auth/internal/server/dlp_policies_handler.go) — Mock at line 26
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Data pillar P0 gap
