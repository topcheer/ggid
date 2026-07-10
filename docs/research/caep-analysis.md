# Continuous Access Evaluation Protocol (CAEP) — Research & GGID Integration Analysis

**Document type:** Technical Research
**Status:** Draft for Architecture Review
**Date:** 2025-01-20
**Author:** Research Team
**Related specs:**
- [OpenID SSE Framework 1.0](https://openid.net/specs/openid-sse-framework-1_0-00.html)
- [OpenID CAEP Specification 1.0](https://openid.net/specs/openid-caep-spec-1_0.html)
- [OpenID CAEP Interoperability Profile 1.0](https://openid.net/specs/openid-caep-interoperability-profile-1_0.html)
- [RFC 8417 — Security Event Token (SET)](https://www.rfc-editor.org/rfc/rfc8417)
- [RFC 8935 — SET Push Delivery](https://www.rfc-editor.org/rfc/rfc8935)
- [RFC 8936 — SET Poll-Based Delivery](https://www.rfc-editor.org/rfc/rfc8936)

---

## Table of Contents

1. [Overview](#1-overview)
2. [SSE Framework](#2-sse-framework)
3. [CAEP Event Types](#3-caep-event-types)
   - 3.1 [session-revoked](#31-session-revoked)
   - 3.2 [credential-change](#32-credential-change)
   - 3.3 [device-compliance-change](#33-device-compliance-change)
   - 3.4 [assurance-level-change](#34-assurance-level-change)
   - 3.5 [identifier-changed](#35-identifier-changed)
   - 3.6 [token-claims-change](#36-token-claims-change)
4. [GGID NATS Integration Design](#4-ggid-nats-integration-design)
5. [Token Revocation Flow](#5-token-revocation-flow)
6. [Session Assurance](#6-session-assurance)
7. [Implementation Roadmap](#7-implementation-roadmap-for-ggid)
8. [Commercial Comparison](#8-comparison-with-commercial-caep-implementations)

---

## 1. Overview

### What is CAEP?

The **Continuous Access Evaluation Protocol (CAEP)** is a specification within the IETF/OpenID **Shared Signals and Events (SSE)** working group. It defines a standardized set of security event types that enable **real-time, cross-domain communication of security state changes** between identity providers (IdPs), relying parties (RPs), and security systems.

### The Problem CAEP Solves

In modern identity infrastructure, a user's security posture is not static. A password may be compromised, a device may fall out of compliance, or an admin may revoke a session. Traditionally, these events propagate slowly — relying parties discover state changes only when they next validate a token (which may be hours away for long-lived JWTs), or through polling.

**Without CAEP:**
- JWT tokens remain valid until expiry (often 15-60 minutes)
- Session revocation requires short token TTLs + aggressive refresh, adding latency
- Credential compromise detected by one service is not communicated to others
- Multi-domain identity (federation) has no standard mechanism for real-time revocation

**With CAEP:**
- Security events propagate in **sub-second** via SSE delivery mechanisms
- Relying parties can **continuously evaluate** access decisions based on the latest security signals
- Cross-organization trust (B2B federation) gains a vendor-neutral event vocabulary

### Relationship to the Shared Signals Framework (SSF)

CAEP is a **profile** of the SSE Framework (SSF). The SSF defines the transport layer — how security events are formatted (SET), delivered (push/poll/streaming), verified, and acknowledged. CAEP defines the **event type vocabulary** that sits on top of that transport.

```
┌─────────────────────────────────────────────────────┐
│                  Application Layer                   │
│  ┌─────────┐  ┌──────────┐  ┌────────────────────┐  │
│  │  CAEP   │  │   RISC   │  │  Custom Profiles   │  │
│  │ Events  │  │  Events  │  │  (e.g., Risk Mgmt) │  │
│  └────┬────┘  └────┬─────┘  └────────┬───────────┘  │
│       │             │                  │              │
│  ─────┴─────────────┴──────────────────┴──────────   │
│              SSE Framework (SSF) 1.0                  │
│  • SET format (RFC 8417)                             │
│  • Delivery: push (RFC 8935), poll (RFC 8936)        │
│  • Subject principals & claims                       │
│  • Transmitter/Receiver configuration metadata       │
│  • Event verification endpoint                       │
└─────────────────────────────────────────────────────┘
```

### Base Specifications

| Specification | Scope | Status |
|---|---|---|
| `openid-sse-framework-1_0` | Transport layer: SET, delivery, verification | Draft - IETF SSE WG |
| `openid-caep-spec-1_0` | CAEP event types & semantics | Draft - IETF SSE WG |
| `openid-caep-interoperability-profile-1_0` | Minimum conformance: session-revoked, credential-change, device-compliance-change | Draft |
| `draft-ietf-secevent-delivery` | HTTP push/poll delivery | Draft - IETF |
| `RFC 8417` | Security Event Token (SET) format | RFC (informational) |

---

## 2. SSE Framework

### 2.1 Security Event Token (SET) Format

A SET is a **JWT (RFC 7519)** that carries security event information. Unlike authentication JWTs, SETs are **not access tokens** — they must never be presented to a resource server for authorization. They are purely **event notifications**.

Per RFC 8417, a SET contains:

| Claim | Required | Description |
|---|---|---|
| `iss` | Yes | Issuer of the SET (the Transmitter) |
| `sub` | Conditional | Subject identifier (if subject is known) |
| `aud` | No | Intended audience (the Receiver) |
| `iat` | Yes | Issued-at timestamp |
| `jti` | Yes | Unique token identifier (for dedup) |
| `events` | Yes | JSON object mapping event URIs to event payloads |
| `toe` | No | Time of event (when the actual event occurred) |

**Example SET payload (before signing):**

```json
{
  "jti": "5d54a1f2-9432-4e89-aa33-2488eff87f12",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "toe": 1737379199,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000"
      },
      "session": {
        "id": "session:a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      },
      "reason": "admin_revocation"
    }
  }
}
```

The SET is signed as a standard JWT (typically using RS256 or ES256) and optionally encrypted (JWE) if the payload contains PII.

### 2.2 Event Delivery Mechanisms

The SSF defines three delivery mechanisms, each with different tradeoffs:

#### HTTP Push (RFC 8935)

```
┌──────────────┐    POST /sse/events     ┌──────────────┐
│  Transmitter │ ───────────────────────> │   Receiver    │
│   (GGID)     │  SET in request body     │  (RP / SIEM)  │
└──────────────┘  Accept: application/secevent+jwt  └──────────────┘
                  <- 200 OK / 202 Accepted ──
```

- Transmitter pushes SETs to receiver's configured endpoint
- Receiver must acknowledge (200/202) within timeout
- Failed pushes trigger retry with exponential backoff
- Best for: real-time, low-latency notifications

#### HTTP Poll (RFC 8936)

```
┌──────────────┐    GET /sse/verify?...   ┌──────────────┐
│   Receiver   │ <─────────────────────── │  Transmitter  │
│              │  200 OK + SET batches    │   (GGID)      │
└──────────────┘                          └──────────────┘
```

- Receiver polls transmitter's events endpoint
- Supports pagination (ack-based cursor)
- Best for: receivers behind firewalls, batch processing, SIEM ingestion

#### Streaming (HTTP POST + Back-Channel)

- Long-lived HTTP connection; server pushes events as they occur
- Hybrid between push and poll
- Best for: high-throughput streaming pipelines (e.g., to Kafka/Splunk)

**GGID Recommendation:** Use **push delivery** for real-time session revocation (Phase 3). For internal NATS-based propagation, use GGID's existing JetStream infrastructure (already implemented in `services/audit/internal/consumer/`).

### 2.3 Subject Types

The SSE Framework defines how events identify their subject (the entity the event applies to):

| Subject Type | Description | GGID Mapping |
|---|---|---|
| `iss_sub` | Issuer-assigned subject identifier | `user:{uuid}` from `domain.Session.UserID` |
| `email` | RFC 822 email address | `user.email` from identity service |
| `phone` | E.164 phone number | `user.phone` |
| `entity` | Non-human entity (service account) | Service account UUID |
| `tenant` | Organizational tenant | `domain.Session.TenantID` |

Subjects use a `subject` object within the event payload:

```json
{
  "subject": {
    "subject_type": "iss_sub",
    "iss": "https://idp.ggid.dev",
    "sub": "user:550e8400-e29b-41d4-a716-446655440000",
    "tenant": "tenant:00000000-0000-0000-0000-000000000001"
  }
}
```

### 2.4 Event Verification

The SSE Framework defines a **verification flow** to confirm the event delivery pipeline is working:

1. **Transmitter** sends a `verification` event containing a random `state` value
2. **Receiver** receives the event and extracts the `state`
3. **Receiver** calls the Transmitter's verification endpoint with the `state`
4. **Transmitter** confirms the state matches and returns success

This proves the delivery channel is bidirectional and functional. It is especially important for initial SSE setup and periodic health checks.

**Verification SET example:**

```json
{
  "jti": "verify-abc-123",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/sse/event-type/verification": {
      "state": "random-nonce-9f8a7b6c"
    }
  }
}
```

---

## 3. CAEP Event Types

CAEP defines a standard vocabulary of security event types. Each event type has a URI identifier and a JSON payload schema. Below is a detailed analysis of each.

The CAEP Interoperability Profile 1.0 defines **three mandatory event types** for compliance: `session-revoked`, `credential-change`, and `device-compliance-change`.

### 3.1 session-revoked

**URI:** `https://schemas.openid.net/secevent/caep/event-type/session-revoked`

#### Trigger Conditions
- Admin manually revokes a user's session (e.g., security incident response)
- User revokes their own session from a device management page
- Idle timeout or inactivity policy triggers automatic revocation
- Session detected as anomalous (impossible travel, new device fingerprint)

#### JSON SET Payload

```json
{
  "jti": "b8e7d5a1-2c34-4f56-8901-234567890abc",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "toe": 1737379199,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000",
        "tenant": "tenant:00000000-0000-0000-0000-000000000001"
      },
      "session": {
        "id": "session:a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "created_at": 1737375600
      },
      "reason": "admin_revocation",
      "reason_admin": "Security incident: suspected credential compromise"
    }
  }
}
```

**Reason values (standardized):**
| Reason | Description |
|---|---|
| `admin_revocation` | Admin-initiated revocation |
| `user_revocation` | User self-service revocation |
| `idle_timeout` | Inactivity policy |
| `security_incident` | Automated security trigger |
| `session_hijack_detected` | Impossible travel / fingerprint mismatch |

#### GGID Integration Point

GGID already has the infrastructure for this event. The existing `SessionRevokeHandler` in `services/gateway/internal/middleware/session.go` (lines 130-177) revokes sessions via Redis. When this handler fires:

1. It deletes `ggid:session:{sessionID}` from Redis and removes from the user session set
2. **NEW:** It should also publish a `session-revoked` CAEP event to NATS
3. The audit service consumer in `services/audit/internal/consumer/nats_consumer.go` persists the event
4. The Gateway subscribes to the CAEP stream and adds the session to a Redis revocation set

The `domain.Session` struct (`services/auth/internal/domain/session.go`) already has the fields needed:
- `ID` (uuid.UUID) -> maps to `session.id`
- `UserID` (uuid.UUID) -> maps to `subject.sub`
- `TenantID` (uuid.UUID) -> maps to `subject.tenant`
- `RevokedAt` (*time.Time) -> confirms revocation occurred

### 3.2 credential-change

**URI:** `https://schemas.openid.net/secevent/caep/event-type/credential-change`

#### Trigger Conditions
- User resets their password
- Admin forces a password reset
- MFA method added (TOTP, WebAuthn)
- MFA method removed
- Credential detected as compromised (breach database match)
- Social login account linked/unlinked

#### JSON SET Payload

```json
{
  "jti": "c9f8e6b2-3d45-4a67-9012-345678901bcd",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379300,
  "toe": 1737379299,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/credential-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000",
        "tenant": "tenant:00000000-0000-0000-0000-000000000001"
      },
      "credential_type": "password",
      "change_type": "updated",
      "change_favorites": false
    }
  }
}
```

**Credential types:**
| Type | Description |
|---|---|
| `password` | Password credential |
| `totp` | Time-based OTP authenticator |
| `webauthn` | FIDO2/WebAuthn credential |
| `federated` | Social/OAuth federated credential |
| `sms` | SMS-based MFA |
| `email_otp` | Email OTP MFA |
| `recovery_code` | Account recovery codes |

**Change types:**
| Type | Description | RP Action |
|---|---|---|
| `created` | New credential added | Informational |
| `updated` | Credential modified | Step-up auth recommended |
| `deleted` | Credential removed | Step-up auth required |
| `compromised` | Credential flagged | Force re-auth |

#### GGID Integration Point

GGID's `domain.Credential` (`services/auth/internal/domain/credential.go`) and `domain.MFA` (`services/auth/internal/domain/mfa.go`) models track credential state. Integration points:

1. **Auth service** (`services/auth/internal/service/`) — when password reset, MFA enrollment, or MFA removal handlers execute
2. **Password hashing** uses Argon2id in `pkg/crypto/crypto.go` (`HashPassword` at line 29) — when a new hash is generated, emit `credential-change` with `change_type=updated`
3. The event triggers RPs to require step-up authentication on the next sensitive operation

### 3.3 device-compliance-change

**URI:** `https://schemas.openid.net/secevent/caep/event-type/device-compliance-change`

#### Trigger Conditions
- MDM (Mobile Device Management) detects policy violation (jailbreak, rooted device)
- Device encryption disabled
- OS version falls below minimum required
- Antivirus/malware protection disabled
- Device marked as lost or stolen
- Device returned to compliance

#### JSON SET Payload

```json
{
  "jti": "d0a9f7c3-4e56-4b78-0123-456789012def",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379400,
  "toe": 1737379399,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000"
      },
      "device": {
        "id": "device:aa11bb22-cc33-dd44-ee55-ff6677889900",
        "platform": "macOS",
        "platform_version": "14.2.1"
      },
      "compliance_status": "non_compliant",
      "reason": "disk_encryption_disabled"
    }
  }
}
```

**Compliance statuses:**
| Status | Description | RP Action |
|---|---|---|
| `compliant` | Device meets all policies | Allow |
| `non_compliant` | Policy violation | Require step-up or block |
| `unknown` | Cannot determine status | Conditional (org policy) |

#### GGID Integration Point

GGID's `domain.Session.DeviceInfo` (`services/auth/internal/domain/session.go`, line 15) currently stores basic browser/OS/device info as `map[string]any`. To support this event type:

1. GGID needs a **device registry** (new table: `devices`) tracking per-user device posture
2. An external MDM webhook or periodic posture check feeds compliance data
3. The event is published when compliance status transitions

**Phase 2 priority** — requires device management infrastructure not yet built in GGID.

### 3.4 assurance-level-change

**URI:** `https://schemas.openid.net/secevent/caep/event-type/assurance-level-change`

#### Trigger Conditions
- User's AAL (Authenticator Assurance Level) changes
- Downgrade: hardware key removed (AAL3 -> AAL2), MFA disabled (AAL2 -> AAL1)
- Upgrade: MFA enrolled (AAL1 -> AAL2), hardware key registered (AAL2 -> AAL3)
- Session-level: step-up auth completes (session AAL raised)

#### JSON SET Payload

```json
{
  "jti": "e1b0a8d4-5f67-4c89-1234-567890123ef0",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379500,
  "toe": 1737379499,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000"
      },
      "assurance_level": {
        "current_aal": "AAL1",
        "previous_aal": "AAL2"
      },
      "change_type": "decreased",
      "reason": "mfa_disabled"
    }
  }
}
```

#### GGID Integration Point

GGID's session metadata (`domain.Session.Metadata`, line 21) stores auth context including MFA verification status. The `domain.MFA` model tracks active MFA methods.

Mapping GGID auth methods to AAL:
- **AAL1**: Password-only authentication (low assurance)
- **AAL2**: Password + TOTP/WebAuthn (medium assurance, current GGID default with MFA)
- **AAL3**: Password + hardware-backed WebAuthn (high assurance)

When a user disables MFA, the `assurance-level-change` event fires with `change_type=decreased`, enabling RPs to enforce step-up auth before sensitive operations.

### 3.5 identifier-changed

**URI:** `https://schemas.openid.net/secevent/caep/event-type/identifier-changed`

#### Trigger Conditions
- User changes their primary email address
- Phone number updated
- Username changed
- Old identifier is being phased out (grace period)

#### JSON SET Payload

```json
{
  "jti": "f2c1b9e5-6a78-4d90-2345-678901234f01",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379600,
  "toe": 1737379599,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/identifier-changed": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000"
      },
      "identifier_type": "email",
      "old_identifier": "old@example.com",
      "new_identifier": "new@example.com",
      "effective_at": 1737380000
    }
  }
}
```

#### GGID Integration Point

The identity service manages user profiles. When a user updates their email or phone via the user management API, this event fires. RPs can update cached user profiles and verify the new identifier before using it for notifications.

**Note:** `old_identifier` may be omitted for privacy reasons — the event still signals that cached identifiers are stale.

### 3.6 token-claims-change

**URI:** `https://schemas.openid.net/secevent/caep/event-type/token-claims-change`

#### Trigger Conditions
- User added to or removed from a group/role that affects JWT claims
- RBAC policy changes (new permission granted, permission revoked)
- Organization membership changes (user joins/leaves an org)
- Custom claim values updated (e.g., department, title for ABAC policies)

#### JSON SET Payload

```json
{
  "jti": "a3d2c0f6-7b89-4e01-3456-789012345a12",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379700,
  "toe": 1737379699,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/token-claims-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "user:550e8400-e29b-41d4-a716-446655440000"
      },
      "claims": ["roles", "groups", "permissions"],
      "reason": "role_assignment_changed"
    }
  }
}
```

**Note:** The `claims` array identifies which claim categories changed, not the specific values. RPs should request a fresh token or query the IdP's userinfo endpoint for updated values.

#### GGID Integration Point

GGID's policy service manages RBAC roles and permissions. The org service manages group memberships. When either changes:

1. Policy service role assignment API fires -> `token-claims-change` event
2. Org service membership change fires -> `token-claims-change` event
3. RPs receive the event and invalidate any cached token claims
4. Next token refresh includes updated claims

This directly addresses a gap in JWT-based auth: JWTs are stateless and carry claims at issuance time. Without this event, role changes don't take effect until token expiry/refresh.

---

## 4. GGID NATS Integration Design

### 4.1 NATS Subject Naming Convention

GGID's existing NATS infrastructure (configured in `services/audit/internal/consumer/nats_consumer.go`) uses JetStream for reliable event delivery. We propose the following subject hierarchy for CAEP events:

```
caep.events                                             (wildcard - all CAEP events)
├── caep.session.revoked.{tenant_id}.{user_id}          (session revocation events)
├── caep.credential.change.{tenant_id}.{user_id}        (credential state changes)
├── caep.device.compliance.{tenant_id}.{device_id}       (device posture changes)
├── caep.assurance.change.{tenant_id}.{user_id}          (AAL level changes)
├── caep.identifier.changed.{tenant_id}.{user_id}        (identifier updates)
├── caep.token.claims.{tenant_id}.{user_id}              (token claim changes)
└── caep.verification                                    (SSE verification events)
```

**Subject design rationale:**
- `tenant_id` is always included for multi-tenant filtering
- `user_id` or `device_id` enables per-entity subscriptions
- Wildcard `caep.>` subscribes to all CAEP events (used by audit service for persistence)
- Wildcard `caep.session.revoked.>` subscribes to all session revocations across tenants (used by Gateway for Redis revocation)

```
                    NATS JetStream: CAEP_EVENTS
    ┌──────────────────────────────────────────────────────┐
    │  Subjects: caep.>                                     │
    │  Retention: Limits (72h)                              │
    │  Storage: File                                        │
    │  MaxAge: 72h, MaxBytes: 2GB                          │
    └─────────────┬───────────────────┬────────────────────┘
                  │                    │
         ┌────────┴────────┐  ┌───────┴────────┐
         │  Audit Consumer  │  │  Gateway        │
         │  (persistence)   │  │  (revocation)   │
         │  Filter: caep.>  │  │  Filter: caep.  │
         │                 │  │  session.revoked │
         └────────┬────────┘  │  .>              │
                  │           └───────┬──────────┘
                  ▼                   ▼
         ┌──────────────┐    ┌──────────────┐
         │ PostgreSQL   │    │    Redis     │
         │ audit_events │    │ revocation   │
         │              │    │ set          │
         └──────────────┘    └──────────────┘
```

### 4.2 CAEP Event Go Struct

```go
// Package caep provides CAEP (Continuous Access Evaluation Protocol)
// event types and NATS integration for GGID.
package caep

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CAEPEvent represents a Security Event Token (SET) carrying a CAEP event.
// This struct maps to the JSON SET format defined in RFC 8417.
type CAEPEvent struct {
	// JTI is the unique identifier for this SET (RFC 4122 UUID recommended).
	JTI string `json:"jti"`

	// ISS is the issuer URI of the SET (the Transmitter).
	ISS string `json:"iss"`

	// AUD is the intended audience (optional, omit for broadcast).
	AUD string `json:"aud,omitempty"`

	// IAT is the issued-at timestamp (Unix seconds).
	IAT int64 `json:"iat"`

	// TOE is the time-of-event timestamp (when the actual event occurred).
	TOE int64 `json:"toe,omitempty"`

	// Events is a map from event-type URI to event-specific payload.
	// CAEP event URIs follow the pattern:
	//   https://schemas.openid.net/secevent/caep/event-type/{type}
	Events map[string]EventPayload `json:"events"`
}

// EventPayload holds the subject and event-specific fields.
type EventPayload struct {
	Subject         Subject         `json:"subject"`
	Session         *SessionInfo    `json:"session,omitempty"`
	CredentialType  string          `json:"credential_type,omitempty"`
	ChangeType      string          `json:"change_type,omitempty"`
	Device          *DeviceInfo     `json:"device,omitempty"`
	ComplianceStatus string         `json:"compliance_status,omitempty"`
	AssuranceLevel  *AssuranceInfo  `json:"assurance_level,omitempty"`
	Identifier      *IdentifierInfo `json:"identifier,omitempty"`
	Claims          []string        `json:"claims,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	State           string          `json:"state,omitempty"` // for verification events
}

// Subject identifies the entity the event applies to.
type Subject struct {
	SubjectType string `json:"subject_type"` // iss_sub, email, phone, entity, tenant
	ISS         string `json:"iss"`
	SUB         string `json:"sub"`
	Tenant      string `json:"tenant,omitempty"`
}

// SessionInfo contains session details for session-revoked events.
type SessionInfo struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

// DeviceInfo contains device details for device-compliance-change events.
type DeviceInfo struct {
	ID              string `json:"id"`
	Platform        string `json:"platform,omitempty"`
	PlatformVersion string `json:"platform_version,omitempty"`
}

// AssuranceInfo holds AAL level information.
type AssuranceInfo struct {
	CurrentAAL  string `json:"current_aal"`  // AAL1, AAL2, AAL3
	PreviousAAL string `json:"previous_aal"`
}

// IdentifierInfo holds identifier change details.
type IdentifierInfo struct {
	Type            string `json:"identifier_type"` // email, phone, username
	OldIdentifier   string `json:"old_identifier,omitempty"`
	NewIdentifier   string `json:"new_identifier,omitempty"`
	EffectiveAt     int64  `json:"effective_at,omitempty"`
}

// --- Event type URIs ---

const (
	EventTypeSessionRevoked       = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTypeCredentialChange     = "https://schemas.openid.net/secevent/caep/event-type/credential-change"
	EventTypeDeviceCompliance     = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
	EventTypeAssuranceLevelChange = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventTypeIdentifierChanged    = "https://schemas.openid.net/secevent/caep/event-type/identifier-changed"
	EventTypeTokenClaimsChange    = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"
	EventTypeVerification         = "https://schemas.openid.net/secevent/sse/event-type/verification"
)

// NewCAEPEvent creates a new SET with the given event type and subject.
func NewCAEPEvent(issuer string, eventType string, subject Subject, payload EventPayload) *CAEPEvent {
	now := time.Now().Unix()
	return &CAEPEvent{
		JTI:    uuid.NewString(),
		ISS:    issuer,
		IAT:    now,
		TOE:    now,
		Events: map[string]EventPayload{eventType: payload},
	}
}

// Marshal serializes the SET to JSON bytes for NATS publishing.
func (e *CAEPEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalCAEPEvent deserializes a SET from JSON bytes.
func UnmarshalCAEPEvent(data []byte) (*CAEPEvent, error) {
	var event CAEPEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
```

### 4.3 NATS Publisher

The publisher lives in the audit/auth service and emits CAEP events to JetStream. It follows the same pattern as the existing audit event consumer in `services/audit/internal/consumer/nats_consumer.go`:

```go
package caep

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CAEPStreamName = "CAEP_EVENTS"
	CAEPSubjectAll = "caep.>"
)

// Publisher emits CAEP events to NATS JetStream.
type Publisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	issuer string // e.g., "https://idp.ggid.dev"
}

// NewPublisher creates a CAEP event publisher connected to NATS.
func NewPublisher(natsURL, issuer string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	// Ensure the CAEP stream exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     CAEPStreamName,
		Subjects: []string{CAEPSubjectAll},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    72 * time.Hour,
		MaxBytes:  2 << 30, // 2 GB
		Replicas:  1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create CAEP stream: %w", err)
	}

	return &Publisher{nc: nc, js: js, issuer: issuer}, nil
}

// PublishSessionRevoked emits a session-revoked CAEP event.
func (p *Publisher) PublishSessionRevoked(ctx context.Context, tenantID, userID, sessionID, reason string) error {
	subject := Subject{
		SubjectType: "iss_sub",
		ISS:         p.issuer,
		SUB:         "user:" + userID,
		Tenant:      "tenant:" + tenantID,
	}

	payload := EventPayload{
		Subject: subject,
		Session: &SessionInfo{ID: "session:" + sessionID},
		Reason:  reason,
	}

	event := NewCAEPEvent(p.issuer, EventTypeSessionRevoked, subject, payload)
	return p.publish(ctx, event, "caep.session.revoked."+tenantID+"."+userID)
}

// PublishCredentialChange emits a credential-change CAEP event.
func (p *Publisher) PublishCredentialChange(ctx context.Context, tenantID, userID, credType, changeType, reason string) error {
	subject := Subject{
		SubjectType: "iss_sub",
		ISS:         p.issuer,
		SUB:         "user:" + userID,
		Tenant:      "tenant:" + tenantID,
	}

	payload := EventPayload{
		Subject:        subject,
		CredentialType: credType,
		ChangeType:     changeType,
		Reason:         reason,
	}

	event := NewCAEPEvent(p.issuer, EventTypeCredentialChange, subject, payload)
	return p.publish(ctx, event, "caep.credential.change."+tenantID+"."+userID)
}

// PublishDeviceComplianceChange emits a device-compliance-change CAEP event.
func (p *Publisher) PublishDeviceComplianceChange(ctx context.Context, tenantID, userID, deviceID, status, reason string) error {
	subject := Subject{
		SubjectType: "iss_sub",
		ISS:         p.issuer,
		SUB:         "user:" + userID,
		Tenant:      "tenant:" + tenantID,
	}

	payload := EventPayload{
		Subject:          subject,
		Device:           &DeviceInfo{ID: "device:" + deviceID},
		ComplianceStatus: status,
		Reason:           reason,
	}

	event := NewCAEPEvent(p.issuer, EventTypeDeviceCompliance, subject, payload)
	return p.publish(ctx, event, "caep.device.compliance."+tenantID+"."+deviceID)
}

// publish serializes and publishes a SET to NATS with async ack.
func (p *Publisher) publish(ctx context.Context, event *CAEPEvent, subject string) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("marshal CAEP event: %w", err)
	}

	ack, err := p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("publish to NATS: %w", err)
	}

	log.Printf("CAEP: published event jti=%s to subject=%s stream=%s seq=%d",
		event.JTI, subject, ack.Stream, ack.Sequence)

	return nil
}

// Close releases the NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}
```

### 4.4 JetStream Consumer Configuration

The Gateway subscribes to session revocation events and maintains a Redis revocation set. The consumer follows the existing pattern from `services/audit/internal/consumer/nats_consumer.go`:

```go
package caep

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
)

const (
	CAEPConsumerName    = "gateway-revocation"
	CAEPRevocationSet   = "ggid:caep:revoked_sessions"
	CAEPRevocationTTL   = 24 * time.Hour // sessions expire anyway; prune revocation entries
)

// GatewaySubscriber listens for session-revoked CAEP events and
// adds the revoked session to Redis for real-time middleware checks.
type GatewaySubscriber struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	rdb *redis.Client
	ctx context.Context
	cancel context.CancelFunc
}

// NewGatewaySubscriber creates a subscriber that revokes sessions in Redis.
func NewGatewaySubscriber(parentCtx context.Context, natsURL string, rdb *redis.Client) (*GatewaySubscriber, error) {
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream: %w", err)
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &GatewaySubscriber{
		nc:  nc,
		js:  js,
		rdb: rdb,
		ctx: ctx,
		cancel: cancel,
	}, nil
}

// Start begins consuming session-revoked events and updating Redis.
func (s *GatewaySubscriber) Start() error {
	// Create a durable consumer filtered to session revocation events
	cons, err := s.js.CreateOrUpdateConsumer(s.ctx, CAEPStreamName, jetstream.ConsumerConfig{
		Name:          CAEPConsumerName,
		Durable:       CAEPConsumerName,
		FilterSubject: "caep.session.revoked.>", // only session revocations
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5, // retry up to 5 times on failure
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	go func() {
		log.Printf("CAEP Gateway Subscriber: consuming caep.session.revoked.>")

		for {
			select {
			case <-s.ctx.Done():
				return
			default:
			}

			batch, err := cons.FetchNoWait(10)
			if err != nil {
				if err == jetstream.ErrNoMessages {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				log.Printf("CAEP Subscriber: fetch error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			for msg := range batch.Messages() {
				if err := s.processRevocation(msg.Data()); err != nil {
					log.Printf("CAEP Subscriber: process error: %v", err)
					msg.Nak() // re-deliver
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

// processRevocation parses a CAEP SET and adds the session to Redis revocation set.
func (s *GatewaySubscriber) processRevocation(data []byte) error {
	event, err := UnmarshalCAEPEvent(data)
	if err != nil {
		return fmt.Errorf("unmarshal SET: %w", err)
	}

	// Extract session-revoked payload
	payload, ok := event.Events[EventTypeSessionRevoked]
	if !ok {
		return nil // not a session-revoked event, skip
	}

	if payload.Session == nil {
		return fmt.Errorf("session-revoked event missing session info")
	}

	// Add session ID to Redis revocation set with TTL
	// Using a Redis SET allows O(1) SISMEMBER checks in middleware
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	// Extract the raw session ID (strip "session:" prefix if present)
	sessionID := payload.Session.ID
	if len(sessionID) > 8 && sessionID[:8] == "session:" {
		sessionID = sessionID[8:]
	}

	pipe := s.rdb.TxPipeline()
	pipe.SAdd(ctx, CAEPRevocationSet, sessionID)
	// Expire individual entries by storing a TTL key alongside
	pipe.Set(ctx, "ggid:caep:revoke_ttl:"+sessionID, "1", CAEPRevocationTTL)
	pipe.SAdd(ctx, "ggid:caep:revoked_sessions", sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis SAdd revocation: %w", err)
	}

	log.Printf("CAEP: session %s revoked in Redis (user=%s, reason=%s)",
		sessionID, payload.Subject.SUB, payload.Reason)

	return nil
}

// Close shuts down the subscriber.
func (s *GatewaySubscriber) Close() {
	s.cancel()
	s.nc.Close()
}
```

### 4.5 NATS Subject Diagram

```
Publisher (Auth/Audit Service)              NATS JetStream               Subscribers
┌─────────────────────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│  PublishSessionRevoked()    │     │  Stream: CAEP_EVENTS  │     │  Audit Consumer     │
│  -> caep.session.revoked.   │────>│  Subjects: caep.>     │────>│  Filter: caep.>     │
│     {tenant}.{user}         │     │                       │     │  -> PostgreSQL      │
│                             │     │  Retention: 72h       │     │     audit_events    │
│  PublishCredentialChange()  │     │  MaxBytes: 2GB        │     └─────────────────────┘
│  -> caep.credential.change. │────>│  Storage: File        │
│     {tenant}.{user}         │     │                       │     ┌─────────────────────┐
│                             │     │  AckExplicit          │────>│  Gateway Subscriber │
│  PublishDeviceCompliance()  │     │  MaxDeliver: 5        │     │  Filter: caep.      │
│  -> caep.device.compliance. │────>│                       │     │  session.revoked.>  │
│     {tenant}.{device}       │     │  Durable Consumers:   │     │  -> Redis revoke    │
│                             │     │  - audit-persister    │     └─────────────────────┘
└─────────────────────────────┘     │  - gateway-revocation │
                                     │                       │     ┌─────────────────────┐
                                     │                       │────>│  External SSE Push  │
                                     │                       │     │  (Phase 3)          │
                                     │                       │     │  -> HTTP POST to RP │
                                     └──────────────────────┘     └─────────────────────┘
```

---

## 5. Token Revocation Flow

### 5.1 End-to-End Flow

When a session is revoked (via admin console, API call, or security automation), the following sequence occurs:

```
Admin Console            Auth Service              NATS JetStream           Gateway Middleware         Redis
     │                        │                          │                         │                     │
     │  DELETE /sessions/{id} │                          │                         │                     │
     ├───────────────────────>│                          │                         │                     │
     │                        │                          │                         │                     │
     │                        │  Session.Revoke()        │                         │                     │
     │                        │  (domain/session.go:30)  │                         │                     │
     │                        │──┐                       │                         │                     │
     │                        │  │ RevokedAt = now       │                         │                     │
     │                        │<─┘                       │                         │                     │
     │                        │                          │                         │                     │
     │                        │  Publish SET to NATS     │                         │                     │
     │                        │  caep.session.revoked.   │                         │                     │
     │                        │  {tenant}.{user}         │                         │                     │
     │                        ├─────────────────────────>│                         │                     │
     │                        │                          │                         │                     │
     │  200 OK revoked        │                          │  Deliver to consumer   │                     │
     │<───────────────────────┤                          ├────────────────────────>│                     │
     │                        │                          │                         │                     │
     │                        │                          │                         │  SADD revoked_set   │
     │                        │                          │                         │  SET ttl:session    │
     │                        │                          │                         ├────────────────────>│
     │                        │                          │                         │                     │
     │                        │                          │                         │  Ack                │
     │                        │                          │<────────────────────────┤                     │
     │                        │                          │                         │                     │
─────┼────────────────────────┼──────────────────────────┼─────────────────────────┼─────────────────────┼────
     │                        │                          │                         │                     │
     │  Next request with JWT │                          │                         │                     │
     │  Authorization: Bearer │                          │                         │                     │
     ├──────────────────────────────────────────────────────────────────────────>│                     │
     │                        │                          │                         │                     │
     │                        │                          │                         │  Extract session_id │
     │                        │                          │                         │  from JWT claim     │
     │                        │                          │                         │                     │
     │                        │                          │                         │  SISMEMBER          │
     │                        │                          │                         │  revoked_set sid    │
     │                        │                          │                         ├────────────────────>│
     │                        │                          │                         │                     │
     │                        │                          │                         │  0 (not revoked)    │
     │                        │                          │                         │<────────────────────┤
     │                        │                          │                         │                     │
     │  200 OK (request proceeds)                       │                         │                     │
     │<─────────────────────────────────────────────────────────────────────────-│                     │
```

### 5.2 Revocation Middleware (Complete Implementation)

This middleware integrates with the existing `SessionManager` in `services/gateway/internal/middleware/session.go`. It adds a Redis revocation set check before allowing the request through:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CAEPRevocationMiddleware checks whether the current session has been
// revoked via a CAEP session-revoked event. It uses Redis SISMEMBER
// which is O(1) — negligible overhead per request.
//
// This middleware expects the JWT to have already been validated by
// JWTAuth and the session_id to be present in the request context.
type CAEPRevocationMiddleware struct {
	rdb            *redis.Client
	revocationSet  string
}

// NewCAEPRevocationMiddleware creates the CAEP revocation checker.
func NewCAEPRevocationMiddleware(rdb *redis.Client) *CAEPRevocationMiddleware {
	return &CAEPRevocationMiddleware{
		rdb:           rdb,
		revocationSet: "ggid:caep:revoked_sessions",
	}
}

// Middleware returns an http.Handler that blocks requests with revoked sessions.
func (m *CAEPRevocationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip public paths (login, register, health, etc.)
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// If Redis is not configured, fail open (don't block on infra issues)
		if m.rdb == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Extract session_id from context (set by JWTAuth middleware)
		// GGID JWTs include a "sid" claim with the session UUID
		sessionID, ok := r.Context().Value(SessionIDKey).(string)
		if !ok || sessionID == "" {
			// Try extracting from Authorization header (JWT "sid" claim)
			sessionID = extractSessionFromJWT(r)
		}

		// If no session ID, let the request through (JWT validation handles auth)
		if sessionID == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		// O(1) check: is this session in the revocation set?
		isRevoked, err := m.rdb.SIsMember(ctx, m.revocationSet, sessionID).Result()
		if err != nil {
			// Redis error — fail open but log
			next.ServeHTTP(w, r)
			return
		}

		if isRevoked {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="session_revoked"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"session_revoked","error_description":"Your session has been revoked. Please re-authenticate."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractSessionFromJWT parses the "sid" (session ID) claim from the
// Authorization header JWT without full validation (JWTAuth already validated).
func extractSessionFromJWT(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	// JWT format: header.payload.signature
	// We only need the payload (index 1) to read the "sid" claim
	tokenParts := strings.Split(parts[1], ".")
	if len(tokenParts) < 2 {
		return ""
	}

	// Decode payload (base64url, no padding)
	// In production, use the crypto package's JWT parsing utilities
	// from pkg/crypto/crypto.go to avoid manual base64 handling
	payload, err := decodeBase64URL(tokenParts[1])
	if err != nil {
		return ""
	}

	// Extract "sid" claim from JSON payload
	return extractJSONField(payload, "sid")
}

// decodeBase64URL decodes a base64url string (no padding).
func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if needed
	if pad := len(s) % 4; pad > 0 {
		s += strings.Repeat("=", 4-pad)
	}
	return base64.URLEncoding.DecodeString(s)
}

// extractJSONField extracts a string field from a JSON byte slice.
func extractJSONField(data []byte, field string) string {
	// In production, use encoding/json for robust parsing
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return ""
	}
	if v, ok := claims[field].(string); ok {
		return v
	}
	return ""
}
```

### 5.3 Integration with Existing Session Middleware

GGID's existing `SessionManager` (`services/gateway/internal/middleware/session.go`) already has:
- `MarkSessionRevoked()` (line 78) — deletes the session key from Redis
- `IsSessionRevoked()` (line 69) — checks if session exists in Redis
- `SessionRevokeHandler()` (line 130) — HTTP handler for DELETE /sessions/{id}

The CAEP revocation middleware adds a **complementary** check: even if the session key still exists in Redis (e.g., due to caching delays), the CAEP revocation set provides an independent signal.

**Middleware chain order:**

```go
// In services/gateway/internal/router/router.go
handler := chain(
    recoveryMiddleware,
    requestIDMiddleware,
    corsMiddleware,
    jwtAuthMiddleware,           // validates JWT signature + expiry
    caepRevocationMiddleware,    // NEW: checks CAEP revocation set in Redis
    sessionValidationMiddleware, // existing: checks ggid:session:{id} in Redis
    rateLimitMiddleware,
    tenantMiddleware,
    proxyHandler,
)
```

### 5.4 Performance Analysis

| Operation | Complexity | Typical Latency |
|---|---|---|
| Redis SISMEMBER | O(1) | <0.1ms (local), <1ms (network) |
| JWT claim extraction (sid) | O(1) | <0.01ms |
| Total middleware overhead | - | ~0.1ms per request |
| NATS publish (async ack) | O(1) | ~0.5ms |
| End-to-end revocation propagation | - | <100ms (NATS + Redis) |

**Comparison to alternatives:**
- **Short JWT TTL (1 min):** Trades security for refresh load (every request may need refresh). Refresh adds ~50ms per request.
- **Token introspection (RFC 7662):** Synchronous call to IdP per request. Adds ~10-50ms network round-trip.
- **CAEP + Redis:** Near-zero per-request overhead. Revocation propagates asynchronously.

---

## 6. Session Assurance

### 6.1 Continuous Assurance Level Validation

CAEP enables **continuous assurance** — the ability to verify that a user's current security posture still meets the required assurance level, not just at login time but throughout the session.

Traditional authentication is a **point-in-time** check: at login, the user proves their identity, receives a token, and that token is trusted until expiry. CAEP transforms this into a **continuous** evaluation:

```
Traditional (point-in-time):
  Login ──────────────────────────────────── Token Expiry
    │                                          │
    └─ Assurance checked once ─────────────────┘── Re-check

CAEP (continuous):
  Login ─── Event ─── Event ─── Event ─── Token Expiry
    │          │         │         │           │
    └──┬───────┴────────┴─────────┘───────────┘
       │
  Each event re-evaluates: "Does the user's current posture
  still satisfy the required assurance level?"
```

### 6.2 AAL Level Mapping for GGID

The NIST SP 800-63B defines three Authenticator Assurance Levels (AAL). GGID maps these to its authentication methods:

| AAL Level | GGID Auth Method | Description | Token Claim |
|---|---|---|---|
| **AAL1** | Password only | Single-factor, something you know | `"aal": 1` |
| **AAL2** | Password + TOTP | Multi-factor, adds something you have | `"aal": 2` |
| **AAL2** | Password + SMS OTP | Multi-factor (weaker than TOTP) | `"aal": 2` |
| **AAL3** | Password + WebAuthn (hardware key) | Hardware-backed MFA | `"aal": 3` |
| **AAL3** | Password + WebAuthn (platform authenticator with user verification) | Biometric-protected key | `"aal": 3` |

**GGID JWT currently includes:**
- `sub` (user ID)
- `tenant_id`
- `roles` (from policy service)
- `session_id` (for revocation tracking)
- **Proposed addition:** `aal` claim for assurance level

### 6.3 Real-Time AAL Downgrade Detection

When a user disables MFA (e.g., from account settings), their AAL drops from 2 to 1. Without CAEP:

```
Timeline (without CAEP):
  t=0: User logs in with password + TOTP (AAL2). JWT issued with aal=2, exp=3600s
  t=600: User disables TOTP. AAL drops to AAL1.
  t=3000: JWT still valid (exp not reached). User accesses AAL2-protected resource.
         RESULT: Access granted despite AAL downgrade. Security gap: 3000s (50 min).
```

With CAEP:

```
Timeline (with CAEP):
  t=0: User logs in with password + TOTP (AAL2). JWT issued with aal=2, exp=3600s
  t=600: User disables TOTP.
         → Auth service publishes assurance-level-change event (AAL2 -> AAL1)
         → NATS delivers to Gateway in <100ms
         → Gateway updates Redis: user's current AAL = 1
  t=601: User accesses AAL2-protected resource.
         → Gateway checks Redis: current AAL=1, required AAL=2
         → Returns 403 with step-up-auth challenge
         RESULT: Security gap: ~100ms. Downgrade detected in near-real-time.
```

**Step-up auth trigger implementation:**

```go
// In the Gateway middleware, after JWT validation:
func checkAssuranceLevel(ctx context.Context, rdb *redis.Client, userID string, requiredAAL int) error {
    // Check Redis for the user's current AAL (updated by CAEP events)
    currentAAL, err := rdb.Get(ctx, "ggid:aal:"+userID).Int()
    if err == redis.Nil {
        // No CAEP update received — fall back to JWT claim
        // (safe default: trust the token's AAL until a downgrade event arrives)
        return nil
    }
    if err != nil {
        return nil // fail open on infra errors
    }

    if currentAAL < requiredAAL {
        return ErrStepUpAuthRequired
    }
    return nil
}
```

### 6.4 Device-Triggered Assurance Downgrade

A device-compliance-change event can also trigger an AAL downgrade:

```
Device loses compliance (e.g., jailbreak detected)
  → device-compliance-change event (status: non_compliant)
  → If the device was providing MFA (WebAuthn), the user's effective AAL drops
  → Gateway detects via subscriber and updates Redis AAL
  → Next request to sensitive resource requires step-up auth
```

This is particularly powerful for **Zero Trust** architectures (see `docs/design/zero-trust-implementation.md`), where device posture is a continuous trust signal.

---

## 7. Implementation Roadmap for GGID

### Phase 1: Core Event Types (MVP CAEP) — 2-3 weeks

**Goal:** Implement the three mandatory CAEP event types with internal NATS propagation.

| Task | Files | Effort |
|---|---|---|
| Define `CAEPEvent` struct & helpers | New: `pkg/caep/event.go` | 2 days |
| NATS publisher (session-revoked, credential-change) | New: `pkg/caep/publisher.go` | 2 days |
| Gateway subscriber -> Redis revocation set | New: `pkg/caep/subscriber.go` | 2 days |
| CAEP revocation middleware | New: `services/gateway/internal/middleware/caep_revocation.go` | 1 day |
| Wire publisher into `SessionRevokeHandler` | Edit: `services/gateway/internal/middleware/session.go` | 1 day |
| Wire publisher into auth service credential handlers | Edit: `services/auth/internal/service/` | 2 days |
| Add `sid` claim to JWT issuance | Edit: `pkg/crypto/crypto.go` or token service | 1 day |
| JetStream stream creation & config | Edit: `services/audit/internal/consumer/nats_consumer.go` | 1 day |
| Tests (unit + integration) | New: `pkg/caep/*_test.go` | 3 days |
| Documentation update | Edit: `docs/` | 1 day |

**Deliverables:**
- session-revoked events propagate in <100ms from revoke to Redis
- credential-change events fire on password reset / MFA change
- Gateway rejects revoked sessions in real-time

### Phase 2: Device Compliance Events — 3-4 weeks

**Goal:** Support device-compliance-change and assurance-level-change events.

| Task | Description | Effort |
|---|---|---|
| Device registry table | New migration: `devices` table with posture fields | 3 days |
| Device enrollment API | REST endpoints in identity service | 5 days |
| MDM webhook receiver | Endpoint for external MDM posture updates | 3 days |
| Device compliance publisher | Extend `pkg/caep/publisher.go` | 2 days |
| Assurance level tracking | Redis AAL tracking, step-up auth trigger | 3 days |
| AAL claim in JWT | Add `aal` claim to token issuance | 1 day |
| Tests | Integration tests for device posture + AAL flow | 5 days |

**Dependencies:** Device management infrastructure (MDM integration or internal posture agent).

### Phase 3: Multi-Domain SSE Delivery — 4-6 weeks

**Goal:** Push CAEP events to external RPs and SIEMs via the SSE Framework's HTTP delivery mechanisms.

| Task | Description | Effort |
|---|---|---|
| SSE Transmitter configuration metadata | `/.well-known/sse-configuration` endpoint | 3 days |
| Transmitter/Receiver management API | CRUD for SSE stream configurations | 5 days |
| HTTP push delivery (RFC 8935) | SET push with retry, backoff, signing | 5 days |
| HTTP poll delivery (RFC 8936) | Polling endpoint with pagination | 3 days |
| Event verification flow | Verification endpoint + verification events | 3 days |
| SET signing (JWT) | Sign SETs with GGID's signing key | 2 days |
| JWE encryption for PII | Optional encryption for sensitive payloads | 3 days |
| External RP integration tests | Test with mock RP receiver | 5 days |
| Documentation: SSE integration guide | `docs/sse-integration.md` | 2 days |

**Deliverables:**
- External applications can register as SSE receivers
- CAEP events are pushed to external endpoints in real-time
- Verification flow confirms delivery pipeline health

### Priority Summary

| Phase | Events | Internal | External | Effort | Priority |
|---|---|---|---|---|---|
| 1 | session-revoked, credential-change | NATS + Redis | - | 2-3 wks | P0 (security critical) |
| 2 | device-compliance, assurance-level | NATS + Redis | - | 3-4 wks | P1 (Zero Trust) |
| 3 | All event types | NATS | HTTP push/poll | 4-6 wks | P2 (federation) |

---

## 8. Comparison with Commercial CAEP Implementations

### 8.1 Microsoft Entra ID — Continuous Access Evaluation (CAE)

Microsoft Entra ID (formerly Azure AD) implements **Continuous Access Evaluation (CAE)**, which predates CAEP but served as inspiration for the standard.

| Feature | Entra CAE | GGID Target |
|---|---|---|
| **Event types** | User disabled, password reset, MFA change, risk detection | CAEP session-revoked, credential-change (Phase 1) |
| **Delivery** | Proprietary: Entra-aware clients receive real-time signals via long-polling / token revocation events | CAEP standard: NATS (internal) + SSE push (external) |
| **Client support** | Requires CAE-aware clients (Microsoft Graph SDK, Outlook, Teams). Standard OAuth2 clients not supported without CAE awareness. | Standard CAEP + SSE — any compliant receiver |
| **Revocation latency** | Near real-time for CAE-aware clients; standard JWT expiry for others | <100ms via NATS + Redis (internal) |
| **Token format** | Short-lived access tokens (1 hour) + CAE revocation checks | Standard JWT + `sid` claim + Redis revocation check |
| **Limitations** | Only works with first-party Microsoft apps and SDKs that implement CAE client-side logic | Standards-based: works with any SSE-compliant receiver |

**Key difference:** Microsoft's CAE is **proprietary** and **client-dependent**. GGID's CAEP implementation is **standards-based** and works with any CAEP-compliant RP.

### 8.2 Okta Identity Cloud

Okta has been an active contributor to the CAEP specification through the OpenID Foundation SSE working group.

| Feature | Okta | GGID Target |
|---|---|---|
| **Event types** | session-revoked, credential-change, device-compliance-change (CAEP Interoperability Profile compliant) | Same three event types (Phase 1-2) |
| **Delivery** | Okta Workflows / Event Hooks (HTTP push), Streaming (Event Hooks) | SSE push/poll (Phase 3) |
| **RISC support** | Yes — Risk and Incident Sharing and Coordination events | Not in scope (future) |
| **Interoperability** | CAEP Interoperability Profile 1.0 compliant (tested with Ping, Microsoft) | Target: CAEP Interoperability Profile compliance |
| **Limitations** | Event Hooks have rate limits (per-org); large orgs need batching | NATS handles high throughput natively |

### 8.3 Ping Identity

Ping Identity (now part of Thoma Bravo) supports CAEP through PingFederate and PingOne.

| Feature | Ping Identity | GGID Target |
|---|---|---|
| **Event types** | session-revoked, credential-change, device-compliance-change, assurance-level-change | Same + identifier-changed, token-claims-change |
| **Delivery** | PingFederate SOAP/REST connectors, PingOne event streams | SSE push/poll (standards-based) |
| **Federation** | Strong B2B federation support — CAEP events flow across Ping federation partners | Phase 3: multi-domain SSE |
| **Policy integration** | CAEP events feed into PingAccess policy engine for real-time access decisions | Gateway middleware integrates with RBAC/ABAC policy engine |

### 8.4 Comparison Matrix

| Capability | Entra CAE | Okta | Ping Identity | GGID (Target) |
|---|---|---|---|---|
| **Standard compliance** | Proprietary (CAE) | CAEP + RISC | CAEP | CAEP |
| **session-revoked** | Yes | Yes | Yes | Phase 1 |
| **credential-change** | Yes | Yes | Yes | Phase 1 |
| **device-compliance-change** | Partial (Intune) | Yes | Yes | Phase 2 |
| **assurance-level-change** | Yes (Conditional Access) | Yes | Yes | Phase 2 |
| **identifier-changed** | No | No | No | Phase 3 |
| **token-claims-change** | Yes | No | Partial | Phase 3 |
| **Internal delivery** | Proprietary | Okta Event System | Ping connectors | NATS JetStream |
| **External delivery** | Limited (first-party) | Event Hooks (HTTP) | REST/SOAP | SSE push/poll (RFC 8935/8936) |
| **Open source** | No | No | No | Yes (Apache 2.0) |

### 8.5 GGID Differentiators

GGID's CAEP implementation offers several advantages over commercial alternatives:

1. **Open source (Apache 2.0)** — fully auditable, no vendor lock-in
2. **NATS JetStream** — built-in reliable delivery, no external message broker needed
3. **Multi-tenant native** — CAEP subjects include tenant_id for multi-tenant event routing
4. **Standards-compliant** — follows IETF/OpenID specs, interoperable with other CAEP implementations
5. **Go-native performance** — sub-millisecond event processing, minimal resource footprint
6. **Integrated with existing infra** — leverages existing Redis session store, NATS audit pipeline, and Gateway middleware chain

---

## Appendix A: Quick Reference — CAEP Event URI Table

| Event Type | URI | Phase | Interop Required |
|---|---|---|---|
| session-revoked | `https://schemas.openid.net/secevent/caep/event-type/session-revoked` | 1 | Yes |
| credential-change | `https://schemas.openid.net/secevent/caep/event-type/credential-change` | 1 | Yes |
| device-compliance-change | `https://schemas.openid.net/secevent/caep/event-type/device-compliance-change` | 2 | Yes |
| assurance-level-change | `https://schemas.openid.net/secevent/caep/event-type/assurance-level-change` | 2 | No |
| identifier-changed | `https://schemas.openid.net/secevent/caep/event-type/identifier-changed` | 3 | No |
| token-claims-change | `https://schemas.openid.net/secevent/caep/event-type/token-claims-change` | 3 | No |
| verification | `https://schemas.openid.net/secevent/sse/event-type/verification` | 3 | Yes |

## Appendix B: GGID File Reference Map

| Component | File Path | CAEP Integration |
|---|---|---|
| Session domain model | `services/auth/internal/domain/session.go` | Session struct provides subject, session_id, tenant_id |
| Credential model | `services/auth/internal/domain/credential.go` | Triggers credential-change events |
| MFA model | `services/auth/internal/domain/mfa.go` | Triggers assurance-level-change events |
| Crypto / password hashing | `pkg/crypto/crypto.go` | Argon2id hash generation triggers credential-change |
| NATS consumer | `services/audit/internal/consumer/nats_consumer.go` | Pattern for CAEP JetStream consumer |
| Audit service main | `services/audit/cmd/main.go` | CAEP publisher initialization |
| Session middleware | `services/gateway/internal/middleware/session.go` | SessionRevokeHandler triggers CAEP event |
| Router | `services/gateway/internal/router/router.go` | Middleware chain includes CAEP revocation check |
| Revocation set | Redis: `ggid:caep:revoked_sessions` | O(1) SISMEMBER per request |
| AAL tracking | Redis: `ggid:aal:{user_id}` | Updated by assurance-level-change subscriber |

---

*This document is a living research artifact. Update with implementation findings as CAEP phases are completed.*
