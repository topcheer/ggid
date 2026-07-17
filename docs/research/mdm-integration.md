# Mobile Device Management (MDM) Integration: Device Compliance, Certificate Provisioning, and Hardware Attestation for GGID

> **Focus**: Integrating GGID with enterprise MDM platforms (Microsoft Intune, Jamf Pro, Google Android Management, Apple DEP) to pull device compliance posture, provision device identity certificates via SCEP/EST, define compliance policies, receive real-time compliance webhooks, and verify hardware attestation — advancing the Devices pillar from "Initial" to "Advanced" in CISA ZTMM 2.0.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§11), curl commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Device Infrastructure](#2-ggid-current-state-device-infrastructure)
3. [Gap Analysis](#3-gap-analysis)
4. [MDM Protocol Landscape](#4-mdm-protocol-landscape)
5. [Proposed Architecture](#5-proposed-architecture)
6. [Device Compliance Posture Model](#6-device-compliance-posture-model)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [API Design + Curl Commands](#8-api-design--curl-commands)
9. [Database Schema](#9-database-schema)
10. [Certificate Provisioning (SCEP/EST)](#10-certificate-provisioning-scepest)
11. [Compliance Policy Engine](#11-compliance-policy-engine)
12. [Real-Time Compliance Webhooks](#12-real-time-compliance-webhooks)
13. [Hardware Attestation](#13-hardware-attestation)
14. [Implementation Backlog with DoD](#14-implementation-backlog-with-dod)
15. [Competitive Differentiation](#15-competitive-differentiation)

---

## 1. Executive Summary

The ZTMM 2.0 assessment identified MDM integration as a **P0 gap** — GGID's Devices pillar is at "Initial" maturity. While GGID has a solid device posture engine (DB-backed, Redis-cached, compliance scoring), it has no integration with enterprise MDM platforms. Device posture is currently updated via manual API calls, not via real-time MDM compliance data.

GGID has foundational device infrastructure:
- **Device posture** (`identity/server/device_posture.go:74`) — DB-backed with Redis cache, compliance scoring, trust levels ✅
- **Device attestation** (`auth/server/device_attest_handler.go:10`) — TPM + Secure Boot + Code Integrity ✅ (basic)
- **Device bindings** (`auth/service/device_binding.go`) — Device-to-user binding ✅
- **Device trust handler** (`auth/server/device_trust_handler.go:21`) — Stub ❌
- **ZTNA posture check** (`gateway/protected_app_router.go:225`) — Posture-gated routing ✅
- **ZT posture aggregation** (`identity/server/zt_posture_handler.go:9`) — Hardcoded ❌

What's completely missing:
1. **No MDM connectors** — No Intune, Jamf, Android Management API integration
2. **No device certificate provisioning** — No SCEP/EST server, no internal CA
3. **No compliance policy engine** — Posture checks are ad-hoc, not rule-driven
4. **No real-time compliance webhooks** — No notification when device falls out of compliance
5. **No hardware attestation validation** — TPM quote accepted but not verified
6. **No device enrollment flow** — No MDM push → GGID registration pipeline

**Recommendation**: Build an **MDM Integration Layer** with: pluggable MDM connectors (Intune/Jamf/Android), device certificate provisioning (SCEP server + internal CA), compliance policy engine (rule-based posture evaluation), real-time webhook receivers (MDM → GGID → CAE revocation), and hardware attestation validation.

**Estimated effort**: 4 sprints for MVP (Intune connector + SCEP + compliance policies + webhooks).

---

## 2. GGID Current State: Device Infrastructure

### Existing Components

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| Device posture DB | `identity/server/device_posture.go:74` | **DB-backed** ✅ | No MDM source |
| Posture compliance eval | `device_posture.go:106` (`evaluatePosture`) | **Works** ✅ | Ad-hoc rules, not policy-driven |
| Redis posture cache | `device_posture.go:59` | **Redis** ✅ | Good pattern |
| Device posture Upsert | `device_posture.go:96` | **Works** ✅ | Manual API calls only |
| Device posture Get | `device_posture.go:127` | **Works** ✅ | Redis cache first |
| Device attestation | `auth/server/device_attest_handler.go:10` | **Basic** ⚠️ | TPM quote accepted, not verified |
| Device binding | `auth/service/device_binding.go` | **Works** ✅ | Device-to-user mapping |
| Device binding status | `auth/server/device_binding_status_handler.go` | **Works** ✅ | Status check |
| Device trust handler | `auth/server/device_trust_handler.go:21` | **Stub** ❌ | Returns 0 |
| ZTNA posture check | `gateway/protected_app_router.go:225` | **Works** ✅ | Posture-gated routing |
| ZT posture aggregation | `identity/server/zt_posture_handler.go:9` | **Hardcoded** ❌ | Returns fake data |
| Device fingerprint | `auth/server/device_fingerprint_analytics_handler.go` | **Hardcoded** ❌ | Fake clusters |
| Device bound SSO | `oauth/service/device_bound_sso.go` | **Works** ✅ | Token-to-device binding |

### Posture Data Model (Existing)

```go
// device_posture.go:14-31
type DevicePosture struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    DeviceID        string
    UserID          uuid.UUID
    TrustLevel      string         // 'unknown', 'low', 'medium', 'high', 'full'
    ComplianceScore int            // 0-100
    Compliant       bool
    Checks          map[string]any // {disk_encrypted, os_version, jailbreak, screen_lock, ...}
    LastCheckAt     *time.Time
    LastSeen        *time.Time
    ...
}
```

The checks map supports arbitrary signals but there's no structured compliance policy engine — it's just a JSON blob.

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No MDM connectors** | Can't pull device status from Intune/Jamf/Android |
| 2 | **No SCEP/EST server** | Can't issue device identity certificates |
| 3 | **No compliance policy engine** | Posture rules are ad-hoc, not configurable |
| 4 | **No compliance webhooks** | Device falls out of compliance → GGID doesn't know |
| 5 | **No device enrollment flow** | No MDM → GGID registration pipeline |
| 6 | **TPM attestation not verified** | Quote accepted but not cryptographically verified |
| 7 | **No Apple Secure Enclave attestation** | iOS attestation not supported |
| 8 | **No Android Play Integrity** | Android attestation not supported |
| 9 | **No device inventory sync** | Devices not auto-discovered from MDM |
| 10 | **No compliance dashboard** | No visibility into fleet compliance status |

---

## 4. MDM Protocol Landscape

### Platform Comparison

| MDM Platform | API | Auth | Key Capabilities | Market Share |
|-------------|-----|------|-------------------|-------------|
| **Microsoft Intune** | Microsoft Graph API | OAuth 2.0 (Azure AD) | Compliance policies, conditional access, app protection | ~35% |
| **Jamf Pro** | Jamf Pro API (REST) | OAuth 2.0 / Bearer token | Apple DEP, macOS/iOS management, pre-stage enrollment | ~15% (Apple) |
| **Google Android Management** | Android Management API | OAuth 2.0 (GCP) | Work profiles, kiosk mode, app distribution | ~10% |
| **VMware Workspace ONE** | REST API | OAuth 2.0 | Cross-platform, conditional access | ~8% |
| **IBM MaaS360** | REST API | API key | Cross-platform, AI threat detection | ~5% |
| **Kandji** | REST API | Bearer token | Apple-only, declarative device management | ~3% |

### Intune Graph API Compliance

```http
GET https://graph.microsoft.com/v1.0/deviceManagement/managedDevices/{id}
Authorization: Bearer {azure_ad_token}

Response:
{
  "id": "...",
  "deviceName": "Alice's MacBook",
  "operatingSystem": "macOS",
  "osVersion": "14.5",
  "complianceState": "compliant",
  "encryptionState": "encrypted",
  "jailBroken": false,
  "managedDeviceOwnerType": "personal",
  "lastSyncDateTime": "2026-07-17T09:00:00Z"
}
```

### Jamf Pro API

```http
GET https://corp.jamfcloud.com/api/v1/computers-inventory/{id}
Authorization: Bearer {jamf_token}

Response:
{
  "hardware": { "osVersion": "14.5", "platform": "Mac" },
  "general": { "lastContactTime": "2026-07-17T09:00:00Z" },
  "security": { "secureBootLevel": "full", "sipStatus": "enabled" },
  "fileVault2": { "fileVaultEnabled": true }
}
```

### Android Management API

```http
GET https://androidmanagement.googleapis.com/v1/{name=enterprises/*/devices/*}
Authorization: Bearer {oauth_token}

Response:
{
  "name": "enterprises/corp/devices/abc",
  "managementMode": "DEVICE_OWNER",
  "state": "ACTIVE",
  "appliedPolicyVersion": "5",
  "applicationReports": [...],
  "hardwareInfo": { "brand": "Google", "model": "Pixel 8" }
}
```

---

## 5. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │       MDM Integration Layer                   │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  MDM Connectors (pluggable)           │    │
                    │  │                                      │    │
                    │  │  ├── Intune Connector (Graph API)     │    │
                    │  │  ├── Jamf Connector (Jamf REST API)   │    │
                    │  │  ├── Android Mgmt Connector (GCP)     │    │
                    │  │  └── Generic Connector (webhook)      │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Compliance Policy Engine             │    │
                    │  │                                      │    │
                    │  │  Rules:                              │    │
                    │  │  ├── min_os_version >= 14.0          │    │
                    │  │  ├── disk_encryption == true         │    │
                    │  │  ├── screen_lock_enabled == true     │    │
                    │  │  ├── jailbreak_detected == false     │    │
                    │  │  ├── required_apps ⊇ [VPN, AV]       │    │
                    │  │  └── forbidden_apps ∩ [] == empty    │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Device Posture Store                 │    │
                    │  │  (existing device_posture table +     │    │
                    │  │   Redis cache)                        │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Integration Hooks                   │    │
                    │  │  ├── → CAE: posture change → revoke   │    │
                    │  │  ├── → PDP: posture in authz decision │    │
                    │  │  ├── → Risk Engine: device signal     │    │
                    │  │  └── → Audit: log compliance changes  │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────┐  ┌─────────────────┐   │
                    │  │  SCEP/EST Server │  │  Webhook        │   │
                    │  │  (device cert    │  │  Receiver       │   │
                    │  │   provisioning)  │  │  (MDM → GGID)   │   │
                    │  └──────────────────┘  └─────────────────┘   │
                    └──────────────────────────────────────────────┘
```

---

## 6. Device Compliance Posture Model

### Unified Compliance Signals (from all MDM sources)

| Signal | Intune | Jamf | Android | Manual API |
|--------|--------|------|---------|------------|
| OS version | ✅ | ✅ | ✅ | ✅ |
| Disk encryption | ✅ | ✅ (FileVault) | ✅ | ✅ |
| Screen lock | ✅ | ✅ | ✅ | ✅ |
| Jailbreak/root | ✅ | ✅ | ✅ | ✅ |
| Firewall enabled | ✅ | ✅ | N/A | ✅ |
| AV software installed | ✅ | ✅ | N/A | ✅ |
| Secure Boot | Via Graph | ✅ | N/A | ✅ |
| SIP (macOS) | Via Graph | ✅ | N/A | ✅ |
| Required apps | ✅ | ✅ | ✅ | ❌ |
| Forbidden apps | ✅ | ✅ | ✅ | ❌ |
| Last sync time | ✅ | ✅ | ✅ | ✅ |
| Battery level | ✅ | ❌ | ✅ | ❌ |
| Compromised status | ✅ | ✅ | ✅ | ✅ |

### Compliance Score Calculation

```go
func calculateComplianceScore(checks map[string]any, policies []CompliancePolicy) (int, bool) {
    score := 100
    allCompliant := true
    
    for _, policy := range policies {
        result := policy.Evaluate(checks)
        if !result.Compliant {
            score -= policy.Penalty
            allCompliant = false
        }
    }
    
    if score < 0 { score = 0 }
    return score, allCompliant
}
```

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Enhance)

| Endpoint | File:Line | Current | Target |
|----------|-----------|---------|--------|
| `GET /api/v1/identity/devices/posture` | `device_posture.go:240` | **Works** ✅ | Add MDM data |
| `POST /api/v1/identity/devices/posture` | `device_posture.go:240` | **Manual update** ✅ | Keep, MDM also feeds |
| `GET /api/v1/identity/zt-posture` | `zt_posture_handler.go:9` | **Hardcoded** ❌ | DB-backed with MDM data |
| `POST /api/v1/auth/devices/attest` | `device_attest_handler.go:10` | **Basic** ⚠️ | Add crypto verification |
| `GET /api/v1/auth/devices/{id}/trust-score` | `device_trust_handler.go:21` | **Stub** ❌ | Real trust from posture |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/mdm/connectors` | GET/POST | List/configure MDM connectors | P0 |
| `/api/v1/mdm/connectors/{id}/sync` | POST | Trigger device sync from MDM | P0 |
| `/api/v1/mdm/devices` | GET | List devices from MDM | P0 |
| `/api/v1/mdm/devices/{id}/compliance` | GET | Get compliance for a device | P0 |
| `/api/v1/mdm/compliance-policies` | GET/POST | List/create compliance policies | P0 |
| `/api/v1/mdm/webhooks/intune` | POST | Intune compliance webhook | P0 |
| `/api/v1/mdm/webhooks/jamf` | POST | Jamf compliance webhook | P0 |
| `/api/v1/mdm/scep` | POST | SCEP certificate enrollment | P1 |
| `/api/v1/mdm/attest/verify` | POST | Hardware attestation verification | P1 |
| `/api/v1/mdm/dashboard` | GET | Fleet compliance dashboard | P1 |

---

## 8. API Design + Curl Commands

### Configure MDM Connector

```bash
curl -X POST https://ggid.corp.com/api/v1/mdm/connectors \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "type": "intune",
    "name": "Corporate Intune",
    "config": {
      "tenant_id": "azure-tenant-id",
      "client_id": "azure-app-id",
      "client_secret": "azure-secret",
      "poll_interval_minutes": 15
    }
  }'

# Response:
{ "connector_id": "mdm_7f3a...", "status": "active", "last_sync": null }
```

### Trigger Device Sync

```bash
curl -X POST https://ggid.corp.com/api/v1/mdm/connectors/mdm_7f3a/sync \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{ "synced_devices": 1247, "updated": 89, "new": 3, "non_compliant": 12 }
```

### List Devices with Compliance

```bash
curl "https://ggid.corp.com/api/v1/mdm/devices?compliant=false&limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response:
{
  "devices": [
    {
      "device_id": "dev_abc123",
      "user_id": "uuid-alice",
      "platform": "macOS",
      "os_version": "14.5",
      "mdm_source": "jamf",
      "compliant": false,
      "compliance_score": 45,
      "trust_level": "low",
      "violations": ["disk_encryption_disabled", "os_version_below_minimum"],
      "last_sync": "2026-07-17T09:00:00Z"
    }
  ],
  "summary": { "total": 1247, "compliant": 1235, "non_compliant": 12, "compliance_rate": 99.0 }
}
```

### Create Compliance Policy

```bash
curl -X POST https://ggid.corp.com/api/v1/mdm/compliance-policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Corporate Mac Standard",
    "platform": "macOS",
    "rules": [
      { "check": "os_version", "operator": ">=", "value": "14.0", "penalty": 30 },
      { "check": "disk_encrypted", "operator": "==", "value": true, "penalty": 25 },
      { "check": "screen_lock_enabled", "operator": "==", "value": true, "penalty": 15 },
      { "check": "jailbreak_detected", "operator": "==", "value": false, "penalty": 100 },
      { "check": "firewall_enabled", "operator": "==", "value": true, "penalty": 10 }
    ],
    "action_on_violation": "mark_non_compliant"
  }'
```

### SCEP Certificate Enrollment

```bash
# Step 1: Device requests certificate via SCEP
curl -X POST https://ggid.corp.com/api/v1/mdm/scep \
  -d '{
    "device_id": "dev_abc123",
    "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...",
    "challenge": "scep-challenge-password"
  }'

# Response:
{
  "certificate": "-----BEGIN CERTIFICATE-----\nMIID...",
  "serial_number": "ser_7f3a...",
  "expires_at": "2027-07-17T00:00:00Z",
  "ca_cert": "-----BEGIN CERTIFICATE-----\nMIID..."
}
```

### Compliance Webhook Receiver

```bash
# Intune sends compliance change notification
curl -X POST https://ggid.corp.com/api/v1/mdm/webhooks/intune \
  -H "X-MS-Notification-Type": "ComplianceChange" \
  -d '{
    "value": [{
      "deviceId": "intune-dev-123",
      "complianceState": "noncompliant",
      "reason": "Encryption required but not enabled",
      "timestamp": "2026-07-17T10:00:00Z"
    }]
  }'

# GGID processes:
# 1. Update device_posture table (compliant=false)
# 2. Publish event to NATS
# 3. CAE revokes active sessions for this device
# 4. Next request through gateway → PDP → deny (posture non-compliant)
```

---

## 9. Database Schema

```sql
-- MDM connector configurations
CREATE TABLE mdm_connectors (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    type                VARCHAR(32) NOT NULL,         -- 'intune', 'jamf', 'android', 'generic'
    name                VARCHAR(128) NOT NULL,
    config              JSONB NOT NULL,               -- {tenant_id, client_id, client_secret_enc, ...}
    status              VARCHAR(16) DEFAULT 'active', -- 'active', 'error', 'disabled'
    last_sync_at        TIMESTAMPTZ,
    last_sync_count     INT DEFAULT 0,
    last_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- MDM-managed devices (synced from MDM)
CREATE TABLE mdm_devices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    connector_id        UUID REFERENCES mdm_connectors(id) ON DELETE CASCADE,

    -- Device identity
    device_id           VARCHAR(256) NOT NULL,        -- GGID device ID
    mdm_device_id       VARCHAR(256),                 -- MDM platform's device ID
    platform            VARCHAR(32),                  -- 'macOS', 'iOS', 'Windows', 'Android'
    os_version          VARCHAR(32),
    hardware_model      VARCHAR(128),
    serial_number       VARCHAR(128),

    -- Ownership
    user_id             UUID,
    ownership_type      VARCHAR(16),                  -- 'corporate', 'personal'

    -- Compliance from MDM
    mdm_compliance_state VARCHAR(32),                 -- 'compliant', 'noncompliant', 'unknown'
    last_mdm_sync       TIMESTAMPTZ,

    -- Certificate
    cert_serial         VARCHAR(128),
    cert_expires_at     TIMESTAMPTZ,

    -- State
    enrolled            BOOLEAN DEFAULT true,
    enrolled_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen           TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, device_id)
);

-- Compliance policies (configurable rules per platform)
CREATE TABLE mdm_compliance_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    platform            VARCHAR(32),                  -- 'macOS', 'iOS', 'Windows', 'Android', null=all
    rules               JSONB NOT NULL,               -- [{check, operator, value, penalty}]
    action_on_violation VARCHAR(32) DEFAULT 'mark_non_compliant',
    -- 'mark_non_compliant', 'revoke_session', 'block_access', 'notify_admin'
    enabled             BOOLEAN DEFAULT true,
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Compliance evaluation log (audit trail)
CREATE TABLE mdm_compliance_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    device_id           VARCHAR(256) NOT NULL,
    user_id             UUID,
    compliant           BOOLEAN NOT NULL,
    compliance_score    INT NOT NULL,
    trust_level         VARCHAR(16),
    violations          JSONB DEFAULT '[]',           -- [{rule, expected, actual}]
    source              VARCHAR(32),                  -- 'mdm_sync', 'webhook', 'manual', 'evaluator'
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Device certificates (issued by GGID SCEP)
CREATE TABLE device_certificates (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    device_id           VARCHAR(256) NOT NULL,
    user_id             UUID,
    serial_number       VARCHAR(128) NOT NULL UNIQUE,
    cert_pem            TEXT NOT NULL,
    fingerprint         VARCHAR(64) NOT NULL,
    issued_by           VARCHAR(256) NOT NULL,        -- CA name
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    revoke_reason       TEXT
);

-- Indexes
CREATE INDEX idx_mdm_connectors_tenant ON mdm_connectors (tenant_id, status);
CREATE INDEX idx_mdm_devices_tenant_device ON mdm_devices (tenant_id, device_id);
CREATE INDEX idx_mdm_devices_tenant_user ON mdm_devices (tenant_id, user_id);
CREATE INDEX idx_mdm_devices_compliance ON mdm_devices (tenant_id, mdm_compliance_state);
CREATE INDEX idx_mdm_policies_tenant ON mdm_compliance_policies (tenant_id, platform, enabled);
CREATE INDEX idx_mdm_log_tenant_time ON mdm_compliance_log (tenant_id, created_at DESC);
CREATE INDEX idx_mdm_log_device ON mdm_compliance_log (tenant_id, device_id, created_at DESC);
CREATE INDEX idx_device_certs_device ON device_certificates (tenant_id, device_id);
CREATE INDEX idx_device_certs_expiry ON device_certificates (expires_at) WHERE revoked_at IS NULL;
```

---

## 10. Certificate Provisioning (SCEP/EST)

### SCEP Flow

```
1. MDM enrolls device → sends CSR to GGID SCEP endpoint
2. GGID validates challenge password + device identity
3. GGID signs CSR with internal CA (ed25519 or ECDSA P-256)
4. Returns signed certificate to device
5. Device uses certificate for mTLS authentication to gateway
6. Gateway validates device cert against GGID CA
```

### Internal CA Design

```go
type InternalCA struct {
    caCert    *x509.Certificate
    caKey     crypto.PrivateKey  // ECDSA P-256
    keyStore  KeyStore           // DB-backed key storage
}

func (ca *InternalCA) SignCSR(csr *x509.CertificateRequest, deviceID string, tenantID uuid.UUID) (*x509.Certificate, error) {
    cert := &x509.Certificate{
        SerialNumber: generateSerial(),
        Subject:      csr.Subject,
        DNSNames:     csr.DNSNames,
        NotBefore:    time.Now(),
        NotAfter:     time.Now().Add(365 * 24 * time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }
    
    derBytes, err := x509.CreateCertificate(rand.Reader, cert, ca.caCert, csr.PublicKey, ca.caKey)
    // ... store in device_certificates table
}
```

---

## 11. Compliance Policy Engine

### Policy Rule Structure

```go
type ComplianceRule struct {
    Check    string      `json:"check"`     // "os_version", "disk_encrypted", etc.
    Operator string      `json:"operator"`  // ">=", "==", "!=", "contains", "in"
    Value    interface{} `json:"value"`     // "14.0", true, ["app1", "app2"]
    Penalty  int         `json:"penalty"`   // Score deduction if violated
}

func (r *ComplianceRule) Evaluate(checks map[string]any) PolicyResult {
    actual, exists := checks[r.Check]
    if !exists {
        return PolicyResult{Compliant: false, Message: fmt.Sprintf("%s not reported", r.Check)}
    }
    
    compliant := r.compare(actual, r.Value)
    return PolicyResult{
        Compliant: compliant,
        Message:   r.formatMessage(actual, compliant),
    }
}
```

### Evaluation Flow

```
For each device:
  1. Fetch applicable policies (match platform)
  2. For each rule in policy:
     a. Get check value from device posture checks map
     b. Compare with rule's expected value
     c. If non-compliant: deduct penalty from score
  3. If any rule non-compliant → device non-compliant
  4. If action = "revoke_session" → publish CAE event
  5. Log to mdm_compliance_log
```

---

## 12. Real-Time Compliance Webhooks

### Webhook Flow

```
MDM Platform                    GGID
─────────────                  ────
Device falls out of              
compliance                        
     │                           
     ▼                           
POST /api/v1/mdm/webhooks/intune 
     │                           
     ▼                           
GGID processes webhook:           
  1. Validate webhook signature   
  2. Update mdm_devices table     
  3. Evaluate compliance policies 
  4. Update device_posture table  
  5. If action=revoke_session:    
     a. Publish to NATS           
     b. CAE revokes sessions      
     c. Next gateway request →    
        PDP → deny (non-compliant)
  6. Log to mdm_compliance_log    
  7. Alert admin if critical      
```

### Webhook Signature Verification

```go
func verifyIntuneWebhook(payload []byte, sig string, cert *x509.Certificate) bool {
    // Intune webhooks signed with Azure AD certificate
    // Verify: base64decode(sig) == RSA-SHA256(payload, cert.PublicKey)
    h := sha256.Sum256(payload)
    err := rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, h[:], sigBytes)
    return err == nil
}
```

---

## 13. Hardware Attestation

### Platform-Specific Attestation

| Platform | Attestation Method | What It Proves | Current GGID |
|----------|-------------------|----------------|-------------|
| **macOS** | Apple Secure Enclave | Device identity, Secure Boot, SIP | ❌ Not implemented |
| **iOS** | App Attest (DeviceCheck) | App integrity, device genuineness | ❌ Not implemented |
| **Android** | Play Integrity API | Device integrity, app authenticity | ❌ Not implemented |
| **Windows** | TPM 2.0 Attestation | Secure Boot, BitLocker, Code Integrity | ⚠️ Basic (accepts, doesn't verify) |
| **Linux** | TPM 2.0 (tpm2-tools) | Measured boot, IMA | ❌ Not implemented |

### TPM Attestation Verification (Upgrade)

```go
func VerifyTPMQuote(quote *TPMQuote, nonce string, aikCert *x509.Certificate) error {
    // 1. Verify AIK certificate is from trusted CA
    if !isTrustedAIK(aikCert) {
        return fmt.Errorf("AIK certificate not from trusted CA")
    }
    
    // 2. Verify quote signature
    // Quote contains: PCR values + nonce, signed by AIK
    if !aikCert.CheckSignature(quote.Signature) {
        return fmt.Errorf("quote signature verification failed")
    }
    
    // 3. Verify nonce matches (prevents replay)
    if quote.Nonce != nonce {
        return fmt.Errorf("nonce mismatch - possible replay attack")
    }
    
    // 4. Verify PCR values (measured boot state)
    if !verifyPCRValues(quote.PCRs, expectedPCRValues) {
        return fmt.Errorf("PCR values indicate tampered boot state")
    }
    
    return nil
}
```

---

## 14. Implementation Backlog with DoD

### P0 — Intune Connector + Compliance Engine (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | MDM DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 2d |
| 2 | Intune Graph API connector | ✅ Fetch device compliance from Graph API ✅ OAuth token refresh ✅ DB-backed ✅ ≥3 tests | 5d |
| 3 | Jamf Pro API connector | ✅ Fetch device compliance from Jamf ✅ Bearer token auth ✅ ≥3 tests | 3d |
| 4 | Compliance policy engine | ✅ Rule-based evaluation ✅ Configurable per platform ✅ Penalty scoring ✅ ≥3 tests | 4d |
| 5 | Compliance policy API | ✅ CRUD for policies ✅ Evaluate on demand ✅ DB-backed ✅ curl test PASS ✅ ≥3 tests | 3d |
| 6 | Device sync API | ✅ POST /mdm/connectors/{id}/sync ✅ Updates device_posture ✅ ≥3 tests | 2d |

### P1 — Webhooks + SCEP + CAE Integration (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Compliance webhook receivers | ✅ Intune + Jamf webhooks ✅ Signature verification ✅ Updates posture ✅ ≥3 tests | 4d |
| 8 | CAE integration (posture → session revocation) | ✅ Non-compliant device → session revoked ✅ NATS event ✅ ≥3 tests | 3d |
| 9 | SCEP certificate enrollment | ✅ Sign CSR with internal CA ✅ Store in device_certificates ✅ ≥3 tests | 3d |
| 10 | Replace hardcoded ZT posture aggregation | ✅ Uses real device_posture + MDM data ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Hardware Attestation + Dashboard (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | TPM attestation verification | ✅ Cryptographic verification (not just accept) ✅ PCR check ✅ ≥3 tests | 4d |
| 12 | Apple App Attest | ✅ iOS attestation validation ✅ DeviceCheck API ✅ ≥3 tests | 3d |
| 13 | Android Play Integrity | ✅ Play Integrity API verification ✅ ≥3 tests | 3d |
| 14 | Fleet compliance dashboard | ✅ Compliance rate chart ✅ Per-platform breakdown ✅ Violation list ✅ DB-backed | 3d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 15 | Android Management connector | Google Android Management API |
| 16 | VMware Workspace ONE connector | AirWatch REST API |
| 17 | Automated remediation | Push policy to MDM to fix non-compliance |
| 18 | Device group policies | Different compliance rules per device group |
| 19 | Kandji connector | Apple-only declarative MDM |
| 20 | Certificate auto-renewal | Renew expiring device certs automatically |

---

## 15. Competitive Differentiation

| Feature | GGID (target) | Microsoft + Intune | Okta + Jamf | Google + Android Mgmt | Cloudflare |
|---------|---------------|--------------------|----|---------|-----------|
| **MDM connectors** | **Intune + Jamf + Android** | Native Intune | Jamf integration | Native | None |
| **Compliance policies** | **Configurable rules** | Intune policies | Okta + Jamf | Android policies | None |
| **Device certs** | **SCEP + internal CA** | Intune + PKI | Jamf + SCEP | Google CA | None |
| **Webhook integration** | **Real-time compliance** | Native | Via API | Native | None |
| **Hardware attestation** | **TPM + Apple + Android** | TPM + Attestation | Apple Attestation | Play Integrity | None |
| **CAE integration** | **Posture → session revoke** | Conditional Access | Okta FastPass | Continuous verification | None |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with pluggable MDM integration — supporting Intune, Jamf, and Android Management from one platform, with real-time compliance webhooks driving CAE session revocation.

---

## References

- [CISA ZTMM 2.0: Devices Pillar](https://www.cisa.gov/zero-trust-maturity-model) — Device maturity requirements
- [Microsoft Intune Graph API](https://learn.microsoft.com/en-us/graph/api/resources/intune-graph-overview) — Device compliance
- [Jamf Pro API](https://developer.jamf.com/) — Apple device management
- [Google Android Management API](https://developers.google.com/android/management) — Android device management
- [SCEP Protocol (RFC 8894)](https://datatracker.ietf.org/doc/html/rfc8894) — Certificate enrollment
- [Apple App Attest](https://developer.apple.com/documentation/devicecheck/app_attest) — iOS attestation
- [Android Play Integrity](https://developer.android.com/google/play/integrity) — Android attestation
- [TPM 2.0 Specification](https://trustedcomputinggroup.org/resource/tpm-library-specification/) — Hardware trust
- [GGID Device Posture](../services/identity/internal/server/device_posture.go) — DB-backed posture at line 74
- [GGID Device Attestation](../services/auth/internal/server/device_attest_handler.go) — Basic TPM at line 10
- [GGID ZTNA Posture Check](../services/gateway/internal/router/protected_app_router.go) — Posture-gated routing at line 225
- [GGID ZT Posture Handler](../services/identity/internal/server/zt_posture_handler.go) — Hardcoded posture at line 9
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Devices pillar P0 gap
- [GGID Continuous Authorization & PDP](./continuous-authorization-pdp.md) — Posture in PDP decisions
- [GGID Risk Adaptive Auth Engine](./risk-adaptive-auth-engine.md) — Device trust as risk signal
