# ZTNA Broker Integration: Making GGID the Identity Provider for Zero Trust Network Access

> **Focus**: How GGID integrates with commercial ZTNA platforms (Zscaler, Cloudflare Access, Twingate, Tailscale) as the identity provider, device posture signal source, and continuous verification engine — replacing traditional VPNs with identity-aware, per-request access control.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: [zero-trust-iam-patterns.md](./zero-trust-iam-patterns.md) covers internal ZT maturity (SPIFFE/SPIRE, mTLS). This document covers **external ZTNA broker integration** — GGID as IdP for commercial VPN-replacement products.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is ZTNA?](#2-what-is-ztna)
3. [Why VPNs Are Obsolete](#3-why-vpns-are-obsolete)
4. [ZTNA Architecture Models](#4-ztna-architecture-models)
5. [Industry Landscape](#5-industry-landscape)
6. [How ZTNA Brokers Consume Identity](#6-how-ztna-brokers-consume-identity)
7. [GGID Current State Analysis](#7-ggid-current-state-analysis)
8. [Gap Analysis](#8-gap-analysis)
9. [Proposed Architecture](#9-proposed-architecture)
10. [Device Posture API](#10-device-posture-api)
11. [Continuous Verification Engine](#11-continuous-verification-engine)
12. [Database Schema](#12-database-schema)
13. [API Design](#13-api-design)
14. [ZTNA Provider Integration Guides](#14-ztna-provider-integration-guides)
15. [Performance Considerations](#15-performance-considerations)
16. [Security Considerations](#16-security-considerations)
17. [Console UI Design](#17-console-ui-design)
18. [Competitive Differentiation](#18-competitive-differentiation)
19. [Implementation Backlog](#19-implementation-backlog)

---

## 1. Executive Summary

Organizations replacing legacy VPNs with Zero Trust Network Access (ZTNA) need their identity provider to serve as the **trust anchor** — the system that authenticates users, provides group/role attributes for policy decisions, supplies device posture signals, and enables continuous session verification.

GGID already implements SAML IdP endpoints and OIDC discovery, which are the primary protocols ZTNA brokers use for identity federation. However, GGID is **missing the ZTNA-specific integration layer**: device posture APIs, SCIM group provisioning, session continuity tokens, and per-request re-evaluation webhooks that modern ZTNA platforms require.

**Recommendation**: Build a **ZTNA Integration Module** that:
1. Exposes device posture data via a standardized API that ZTNA brokers can query
2. Provides SCIM 2.0 group provisioning so ZTNA policies can reference GGID groups
3. Supports continuous session verification via CAEP events or webhook callbacks
4. Generates provider-specific setup guides (Zscaler, Cloudflare, Twingate, Tailscale) with copy-paste configuration
5. Enforces device trust as a conditional access policy within GGID's own Gateway

**Estimated effort**: 3 sprints for MVP (posture API + SCIM groups + provider guides) + 2 sprints for continuous verification + Console UI.

---

## 2. What is ZTNA?

Zero Trust Network Access is a security model that replaces network-level trust (VPN tunnel → full network access) with **identity-level trust** (authenticate → authorize per application → verify continuously).

### Core ZTNA Principles

| Principle | VPN Model | ZTNA Model |
|-----------|-----------|------------|
| **Trust boundary** | Network perimeter (inside VPN = trusted) | Identity + device + context (per request) |
| **Access granularity** | Network segment (broad) | Individual application (narrow) |
| **Authentication** | Once at tunnel establishment | Per-session or per-request |
| **Device posture** | Not checked | Checked before and during access |
| **Lateral movement** | Possible (full network access) | Prevented (only authorized apps visible) |
| **Network exposure** | Apps exposed to VPN subnet | Apps "cloaked" — invisible until authorized |
| **Session revocation** | Kill VPN tunnel (disruptive) | Revoke specific app session (surgical) |

### ZTNA vs VPN: The Shift

```
┌─────────────────────────────────────────────────────┐
│                   VPN MODEL                          │
│                                                     │
│   User ──VPN Tunnel──► [ Entire Network ]           │
│                         ├── DB Server               │
│                         ├── App Server              │
│                         ├── File Server             │
│                         ├── Admin Console           │
│                         └── Internal Wiki           │
│                                                     │
│   Problem: One credential = full network access     │
│            No per-app control                       │
│            No device posture check                  │
│            Lateral movement trivial                 │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                   ZTNA MODEL                         │
│                                                     │
│   User ──► ZTNA Broker ──► [ Specific App Only ]   │
│              │                                       │
│              ├── Verify identity (SAML/OIDC)        │
│              ├── Check device posture               │
│              ├── Evaluate access policy             │
│              ├── Grant per-app session              │
│              └── Continuously re-verify             │
│                                                     │
│   Benefit: Least privilege per application           │
│            Device health enforced                    │
│            No lateral movement possible              │
│            Apps invisible to unauthorized users      │
└─────────────────────────────────────────────────────┘
```

---

## 3. Why VPNs Are Obsolete

### 2025-2026 Threat Landscape

According to the Verizon 2025 Data Breach Investigations Report:
- **Edge devices and VPNs accounted for 22% of vulnerabilities** used in breaches (up from 3% the prior year — an 8x increase)
- Only 54% of VPN-related vulnerabilities were fully patched
- Median patch time: 32 days
- VPN credentials are the #1 initial access vector in ransomware attacks

### VPN Vulnerabilities

| Vulnerability | Impact | ZTNA Mitigation |
|--------------|--------|-----------------|
| Credential theft → full network access | Attacker reaches all internal services | Per-app access — stolen credential grants only authorized apps |
| No device posture check | Compromised/infected devices access network | Device health verified before every session |
| Broad lateral movement | Attacker pivots between services | No network access — only specific app endpoints |
| Static session (no re-evaluation) | revoked users retain access until token expiry | Continuous verification via CAEP/webhooks |
| Complex firewall rules | Misconfigurations expose services | Identity-based policies, no IP-based rules |
| Performance overhead | All traffic through VPN concentrator | Direct-to-app or edge-routed traffic |

---

## 4. ZTNA Architecture Models

### Model 1: Overlay / Peer-to-Peer (Tailscale, Twingate)

```
    User Device                    Internal App
         │                              │
         │   WireGuard/QUIC Tunnel      │
         ├──────────────────────────────►│
         │   (direct P2P when possible)  │
         │                              │
         │                              │
    ┌────┴──────────┐          ┌───────┴───────┐
    │  Client Agent  │          │  Connector    │
    │  (on device)   │          │  (in network) │
    └────┬──────────┘          └───────┬───────┘
         │                              │
         └──────────┬───────────────────┘
                    │
            ┌───────▼───────┐
            │  Control Plane │
            │  (SaaS)        │
            │                │
            │  Auth: GGID    │
            │  Policy: Per   │
            │  resource      │
            └───────────────┘
```

**Characteristics**:
- Data flows directly device-to-resource (lowest latency)
- Control plane is SaaS (or self-hosted for Tailscale/Headscale)
- Only metadata transits the vendor's cloud
- Best for: engineering teams, low-latency requirements

### Model 2: Broker / Proxy (Cloudflare Access, Zscaler ZPA)

```
    User Device           ZTNA Edge            Internal App
         │                    │                     │
         │ 1. Connect to edge │                     │
         ├───────────────────►│                     │
         │                    │                     │
         │ 2. GGID SAML/OIDC  │                     │
         │    authentication   │                     │
         ├───────────────────►│                     │
         │                    │                     │
         │                    │ 3. Connect to app   │
         │                    │    via connector    │
         │                    ├────────────────────►│
         │                    │                     │
         │                    │ 4. Broker session   │
         │◄───────────────────┼────────────────────►│
         │                    │                     │
         │ 5. All traffic      │                     │
         │    through edge     │                     │
         │◄───────────────────►│◄───────────────────►│
```

**Characteristics**:
- All traffic transits vendor's edge network
- Enables inline inspection (DLP,SWG, sandboxing)
- Higher latency (extra hop) but richer security
- Best for: regulated enterprises, compliance requirements

### Model 3: Identity-Aware Proxy (GGID Native)

GGID's own Gateway can serve as a lightweight ZTNA for HTTP-based applications:

```
    User Device           GGID Gateway          Internal App
         │                    │                     │
         │ 1. HTTP Request    │                     │
         ├───────────────────►│                     │
         │                    │ 2. JWT validation   │
         │                    │    Device posture   │
         │                    │    Policy check     │
         │                    ├────────────────────►│
         │                    │                     │
         │                    │ 3. Response         │
         │◄───────────────────┼◄────────────────────┤
```

This is what GGID's Gateway already does for API access — it can be extended to serve as a full identity-aware proxy for internal web applications.

---

## 5. Industry Landscape

### Comparison Matrix

| Feature | Zscaler ZPA | Cloudflare Access | Twingate | Tailscale | **GGID Gateway** |
|---------|------------|-------------------|----------|-----------|------------------|
| **Architecture** | Broker/Proxy | Broker/Proxy | Overlay/P2P | Overlay/P2P | Reverse Proxy |
| **Data path** | Through edge | Through edge | Direct P2P | Direct P2P | Through gateway |
| **IdP integration** | SAML + OIDC + SCIM | SAML + OIDC + SCIM | SAML + OIDC + SCIM | SAML + OIDC | **Native (GGID is IdP)** |
| **Device posture** | Profile-based | Custom API | OS checks | Attribute-based | **API + conditional** |
| **Clientless access** | Web apps | Excellent (SSH/VNC/RDP) | Limited | Limited | **Yes (HTTP)** |
| **Inline inspection** | DLP, SWG, sandbox | DLP, CASB, SWG | No | No | **Audit logging** |
| **SCIM provisioning** | Yes | Yes | Yes | Yes | **Yes (existing)** |
| **Continuous re-check** | At brokering | Per HTTP request | At session setup | Per connection | **Per request (configurable)** |
| **Self-hosted option** | Private Service Edge | No | No | Yes (Headscale) | **Yes (on-prem)** |
| **FIPS 140-2** | Yes | Cipher mode | No | No | **Via Go crypto** |
| **FedRAMP** | High + IL5 | Moderate | None | None | **N/A (self-hosted)** |
| **Open source** | No | No | No | Partial (client) | **Yes (Apache 2.0)** |
| **Protocol support** | TCP/UDP | L4-L7 | TCP/UDP/ICMP | Full IP layer | **HTTP/HTTPS/gRPC** |

### When to Use GGID Gateway vs External ZTNA

| Scenario | Recommended Solution |
|----------|---------------------|
| HTTP/REST API access to microservices | **GGID Gateway** (native, no extra hop) |
| Access to internal web apps (HTTP) | **GGID Gateway** (identity-aware reverse proxy) |
| Access to RDP/SSH/non-HTTP protocols | **External ZTNA** (Zscaler/Cloudflare/Twingate) |
| Access from unmanaged/contractor devices | **Cloudflare Access** (clientless browser) |
| Regulated environment (FedRAMP) | **Zscaler ZPA** (FedRAMP High) |
| Engineering team, mesh network | **Tailscale** (WireGuard mesh, self-hostable) |
| Need inline DLP/SWG | **Zscaler or Cloudflare** (full SSE platform) |

---

## 6. How ZTNA Brokers Consume Identity

ZTNA brokers need three things from the identity provider:

### 1. Authentication (SAML/OIDC)

The broker redirects unauthenticated users to GGID for login. GGID returns a SAML assertion or OIDC authorization code that the broker validates.

```
User → ZTNA Broker → Redirect to GGID login → User authenticates (password + MFA)
→ GGID issues SAML assertion → Broker validates assertion → Grants session
```

**GGID status**: **Implemented** — `/saml/idp/sso` and OIDC authorization endpoint exist.

### 2. Group/Role Attributes (SAML Claims + SCIM)

The broker needs to know which groups a user belongs to, to apply per-group access policies (e.g., "Engineering group can access GitLab, Finance group can access QuickBooks").

**Via SAML claims**:
```xml
<saml:Attribute Name="groups">
  <saml:AttributeValue>engineering</saml:AttributeValue>
  <saml:AttributeValue>on-call</saml:AttributeValue>
</saml:Attribute>
```

**Via SCIM 2.0** (for pre-provisioning):
```
GGID pushes group memberships to ZTNA broker via SCIM POST /Groups
```

**GGID status**: **Partial** — SAML attributes exist but no standardized "groups" claim. SCIM endpoints exist but SCIM client (push to external) is missing.

### 3. Device Posture Signals (API or Webhook)

The broker queries the identity provider (or an MDM/EDR system) to check device health before granting access:

```
ZTNA Broker → GET https://ggid.corp.com/api/v1/devices/{device_id}/posture
→ Response: { "compliant": true, "os_version": "14.5", "encrypted": true, "managed": true }
```

**GGID status**: **Missing** — no device posture API exists.

### 4. Continuous Verification (CAEP / Webhook)

After initial authentication, the broker needs to know if the user's security state changes mid-session:

```
GGID detects: user password compromised
→ GGID sends CAEP event: session-revoked
→ ZTNA Broker receives event → terminates active sessions for that user
```

**GGID status**: **Researched** (CAEP analysis exists) but **not implemented**.

---

## 7. GGID Current State Analysis

### Existing ZTNA-Relevant Components

| Component | File | Status |
|-----------|------|--------|
| SAML IdP SSO | `services/oauth/internal/server/server.go:979` | **Implemented** — `/saml/idp/sso` |
| SAML IdP metadata | `services/oauth/internal/server/server.go:967` | **Implemented** — `/saml/idp/metadata` |
| OIDC discovery | `services/oauth/internal/server/server.go:213` | **Implemented** — `/.well-known/openid-configuration` |
| JWKS endpoint | `services/oauth/internal/server/server.go:225` | **Implemented** — `/oauth/jwks` |
| JWT validation (Gateway) | `services/gateway/internal/middleware/jwt_claims.go` | **Implemented** — per-request token validation |
| GeoIP middleware | `services/gateway/internal/middleware/geoip.go` | **Implemented** — IP geolocation |
| IP allowlist | `services/gateway/internal/middleware/ipallowlist.go` | **Implemented** — IP-based filtering |
| Rate limiting | `services/gateway/internal/middleware/ratelimit.go` | **Implemented** — per-IP and per-tenant |
| Host validation | `services/gateway/internal/middleware/host_validation.go` | **Implemented** — DNS rebinding defense |
| Session timeout | `services/gateway/internal/middleware/session_timeout.go` | **Implemented** — idle/absolute timeout |
| Trusted devices | `services/auth/internal/server/trusted_devices_handler.go` | **Implemented** — device trust tracking |
| Device tracking | `services/auth/internal/service/device_tracking.go` | **Implemented** — known device records |
| Device binding | `services/auth/internal/service/device_binding.go` | **Implemented** — token-to-device binding |
| Risk assessment | `services/auth/internal/service/risk_auth.go:36` | **Implemented** — login risk scoring |
| Conditional access | `services/policy/internal/server/conditional_access_handler.go` | **Implemented** — policy-based conditions |
| SCIM 2.0 endpoints | `services/identity/internal/scim/handler.go` | **Implemented** — user/group CRUD |
| Per-tenant IdP config | `services/identity/internal/idpconfig/idpconfig.go` | **Implemented** — federation config |
| Anomaly detection | `services/auth/internal/service/anomaly_detection.go` | **Implemented** — impossible travel, brute force |
| Session revocation | `services/auth/internal/service/session_revocation.go` | **Implemented** — token/session invalidation |

### What's Missing for ZTNA Integration

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No device posture API** | ZTNA brokers cannot query device health from GGID |
| 2 | **No standardized groups claim in SAML** | ZTNA policies can't reference GGID groups via SAML attributes |
| 3 | **No SCIM client (outbound)** | Groups not pushed to ZTNA brokers; only inbound SCIM works |
| 4 | **No continuous verification webhook** | ZTNA brokers can't receive real-time session change notifications |
| 5 | **No per-request device trust enforcement** | Gateway doesn't check device posture on each API request |
| 6 | **No ZTNA setup wizard** | No guided configuration for Zscaler/Cloudflare/Twingate/Tailscale |
| 7 | **No device posture evaluation engine** | No configurable rules: "require managed + encrypted + OS >= X" |
| 8 | **No session continuity tokens** | No mechanism to tie ZTNA session to GGID session for unified revocation |
| 9 | **No network context in policy decisions** | ABAC policies don't include "accessed via ZTNA" vs "direct" context |
| 10 | **No ZTNA access logging** | No unified view of "who accessed what via which ZTNA broker" |

---

## 8. Gap Analysis

### Use Cases That Fail Today

| # | Use Case | Current Behavior | Expected Behavior |
|---|----------|-----------------|-------------------|
| 1 | "Configure Zscaler ZPA to use GGID as IdP" | Manual SAML metadata exchange, no groups claim | Wizard generates Zscaler-specific config + SCIM groups push |
| 2 | "Block access from non-compliant devices" | No device posture check in Gateway | Gateway middleware queries posture API, blocks if non-compliant |
| 3 | "Revoke user's ZTNA session when password changes" | No event sent to ZTNA broker | CAEP `session-revoked` event pushed to ZTNA broker |
| 4 | "Map GGID roles to Cloudflare Access groups" | Manual SAML attribute mapping | Auto-mapped via SCIM group provisioning |
| 5 | "Show which apps a user accessed via ZTNA" | No ZTNA access logging | Unified audit trail: GGID login + ZTNA app access |
| 6 | "Require corporate device for admin console" | No device-based conditional access | Policy: `if resource == admin && device.managed == false → deny` |

---

## 9. Proposed Architecture

### ZTNA Integration Module

```
                    ┌──────────────────────────────────────────────┐
                    │              GGID Platform                    │
                    │                                              │
                    │  ┌────────────────────────────────────────┐  │
                    │  │         ZTNA Integration Module        │  │
                    │  │                                        │  │
                    │  │  ┌──────────────┐ ┌─────────────────┐ │  │
                    │  │  │ Device       │ │ SCIM Group      │ │  │
                    │  │  │ Posture      │ │ Provisioning    │ │  │
                    │  │  │ API          │ │ (outbound)      │ │  │
                    │  │  └──────┬───────┘ └──────┬──────────┘ │  │
                    │  │         │                │            │  │
                    │  │  ┌──────┴────────────────┴──────────┐ │  │
                    │  │  │  Continuous Verification Engine   │ │  │
                    │  │  │                                  │ │  │
                    │  │  │  - CAEP event transmitter        │ │  │
                    │  │  │  - Webhook callback              │ │  │
                    │  │  │  - Session revocation broadcast   │ │  │
                    │  │  └──────────────────────────────────┘ │  │
                    │  │                                        │  │
                    │  │  ┌──────────────────────────────────┐  │  │
                    │  │  │  Provider Config Registry        │  │  │
                    │  │  │  (Zscaler, Cloudflare, Twingate, │  │  │
                    │  │  │   Tailscale configs per tenant)  │  │  │
                    │  │  └──────────────────────────────────┘  │  │
                    │  └────────────────────────────────────────┘  │
                    │                      │                       │
                    │  ┌───────────────────▼───────────────────┐   │
                    │  │  Gateway Middleware                   │   │
                    │  │  (Device Posture Check)               │   │
                    │  │  + SAML Groups Claim                  │   │
                    │  └───────────────────────────────────────┘   │
                    └──────────────────────────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
                    ▼                   ▼                   ▼
             ┌────────────┐    ┌──────────────┐    ┌────────────┐
             │ Zscaler    │    │ Cloudflare   │    │ Twingate / │
             │ ZPA        │    │ Access       │    │ Tailscale  │
             └────────────┘    └──────────────┘    └────────────┘
```

---

## 10. Device Posture API

### Design

GGID exposes a standardized device posture API that ZTNA brokers (and GGID's own Gateway) can query:

```
GET /api/v1/devices/{device_id}/posture
Authorization: Bearer {service_token}

Response:
{
    "device_id": "dev_a1b2c3d4",
    "user_id": "uuid",
    "tenant_id": "uuid",
    "compliant": true,
    "posture_score": 85,
    "checks": {
        "managed": true,
        "encrypted": true,
        "os_name": "macOS",
        "os_version": "14.5.0",
        "os_up_to_date": true,
        "antivirus_active": true,
        "firewall_enabled": true,
        "disk_encrypted": true,
        "screen_lock_enabled": true,
        "jailbroken": false,
        "edr_installed": true,
        "edr_provider": "crowdstrike",
        "last_seen": "2026-07-17T09:45:00Z"
    },
    "policy_results": [
        {
            "policy": "corporate-managed",
            "result": "compliant",
            "message": "Device meets all requirements"
        }
    ],
    "evaluated_at": "2026-07-17T09:45:32Z",
    "expires_at": "2026-07-17T09:50:32Z"
}
```

### Posture Signal Sources

GGID collects device posture signals from multiple sources:

| Source | Signal | Collection Method |
|--------|--------|-------------------|
| **Auth token** | Device ID, user agent, IP | Extracted during login |
| **Client SDK** | OS version, encryption, jailbreak | GGID SDK reports on login |
| **MDM integration** | Managed status, compliance | API poll (Intune/Jamf/Kandji) |
| **EDR integration** | AV active, threat status | API poll (CrowdStrike/SentinelOne) |
| **Gateway observation** | Known/unknown device, IP reputation | Inferred from access patterns |
| **Admin registration** | Device enrollment, trust level | Manual or automated enrollment |

### Posture Evaluation Engine

```go
// services/auth/internal/service/device_posture.go

// PosturePolicy defines device requirements for a given context.
type PosturePolicy struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    Name            string              // "corporate-managed"
    Description     string
    Requirements    []PostureRequirement
    Priority        int
    Enabled         bool
}

type PostureRequirement struct {
    Field       string  // "managed", "encrypted", "os_version"
    Operator    string  // "==", ">=", "in", "exists"
    Value       any     // true, "14.0", ["macOS", "iOS"]
}

// Evaluate checks a device against all posture policies.
func (s *PostureService) Evaluate(
    ctx context.Context,
    tenantID uuid.UUID,
    deviceID string,
) (*PostureResult, error) {
    // 1. Gather device signals from all sources
    signals, err := s.gatherSignals(ctx, deviceID)
    if err != nil { return nil, err }

    // 2. Get applicable policies for tenant
    policies, err := s.repo.GetActivePolicies(ctx, tenantID)
    if err != nil { return nil, err }

    // 3. Evaluate each policy
    results := []PolicyResult{}
    overallCompliant := true
    for _, policy := range policies {
        result := s.evaluatePolicy(policy, signals)
        results = append(results, result)
        if !result.Compliant {
            overallCompliant = false
        }
    }

    return &PostureResult{
        DeviceID:    deviceID,
        Compliant:   overallCompliant,
        Signals:     signals,
        PolicyResults: results,
        EvaluatedAt: time.Now(),
    }, nil
}
```

---

## 11. Continuous Verification Engine

### Session Lifecycle

```
1. User authenticates via GGID (password + MFA)
   → GGID creates session, records device posture

2. ZTNA broker authenticates user via SAML/OIDC
   → GGID issues assertion with session reference

3. ZTNA broker grants per-app session
   → Session tied to GGID session ID

4. GGID continuously monitors user/device state:
   ├── Password changed? → CAEP: credential-change
   ├── Device fell out of compliance? → CAEP: device-compliance-change
   ├── Session revoked by admin? → CAEP: session-revoked
   ├── User role changed? → CAEP: token-claims-change
   └── Impossible travel detected? → CAEP: session-revoked

5. ZTNA broker receives CAEP event
   → Terminates affected sessions immediately
   → User must re-authenticate
```

### CAEP Event Transmission

GGID transmits CAEP events to configured ZTNA brokers:

```go
// services/oauth/internal/service/caep_transmitter.go

// TransmitEvent sends a CAEP security event to all registered receivers.
func (s *CAEPTransmitter) TransmitEvent(ctx context.Context, event *CAEPEvent) error {
    receivers, err := s.repo.GetActiveReceivers(ctx, event.TenantID)
    if err != nil { return err }

    // Build SET (Security Event Token) per RFC 8417
    set := s.buildSET(event)

    for _, receiver := range receivers {
        // Push via HTTP (RFC 8935)
        go s.pushSET(ctx, receiver, set)
    }
    return nil
}

// buildSET creates a Security Event Token (JWT) for CAEP.
func (s *CAEPTransmitter) buildSET(event *CAEPEvent) string {
    claims := map[string]any{
        "iss":    s.issuerURL,
        "sub":    event.SubjectID,
        "iat":    time.Now().Unix(),
        "jti":    uuid.New().String(),
        "events": map[string]any{
            "https://schemas.openid.net/secevent/caep/event-type/" + event.EventType: event.Payload,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.keyProvider.Metadata().KeyID
    signed, _ := token.SignedString(s.keyProvider.Private())
    return signed
}
```

---

## 12. Database Schema

```sql
-- Device posture signals (collected from multiple sources)
CREATE TABLE device_posture_signals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    device_id       VARCHAR(256) NOT NULL,
    user_id         UUID,

    -- OS information
    os_name         VARCHAR(64),
    os_version      VARCHAR(64),
    os_up_to_date   BOOLEAN,

    -- Security state
    managed         BOOLEAN DEFAULT false,
    encrypted       BOOLEAN DEFAULT false,
    disk_encrypted  BOOLEAN DEFAULT false,
    firewall_enabled BOOLEAN DEFAULT false,
    antivirus_active BOOLEAN DEFAULT false,
    jailbroken      BOOLEAN DEFAULT false,
    screen_lock     BOOLEAN DEFAULT false,

    -- EDR/MDM integration data
    mdm_provider    VARCHAR(64),                 -- 'intune', 'jamf', 'kandji'
    edr_provider    VARCHAR(64),                 -- 'crowdstrike', 'sentinelone'
    edr_active      BOOLEAN DEFAULT false,
    edr_zta_score   INT,                         -- 0-100 risk score from EDR

    -- Metadata
    device_name     VARCHAR(256),
    device_type     VARCHAR(32),                 -- 'laptop', 'mobile', 'desktop'
    fingerprint     VARCHAR(256),
    last_seen       TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, device_id)
);

-- Posture evaluation policies
CREATE TABLE device_posture_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(128) NOT NULL,
    description     TEXT,
    requirements    JSONB NOT NULL,              -- Array of PostureRequirement
    priority        INT DEFAULT 0,
    enabled         BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Posture evaluation results (cached for queries)
CREATE TABLE device_posture_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    device_id       VARCHAR(256) NOT NULL,
    compliant       BOOLEAN NOT NULL,
    posture_score   INT NOT NULL,
    policy_results  JSONB NOT NULL,
    evaluated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL         -- Cache TTL (default 5 min)
);

-- ZTNA provider configurations
CREATE TABLE ztna_provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    provider        VARCHAR(64) NOT NULL,         -- 'zscaler', 'cloudflare', 'twingate', 'tailscale'
    name            VARCHAR(128) NOT NULL,
    config_json     JSONB NOT NULL,               -- Provider-specific config
    scim_endpoint   TEXT,                          -- SCIM push endpoint
    scim_token_enc  TEXT,                          -- Encrypted SCIM bearer token
    caep_endpoint   TEXT,                          -- CAEP event push endpoint
    enabled         BOOLEAN DEFAULT true,
    last_sync_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, provider, name)
);

-- SCIM provisioning state (outbound to ZTNA)
CREATE TABLE ztna_scim_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_config_id UUID NOT NULL REFERENCES ztna_provider_configs(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,
    entity_type     VARCHAR(32) NOT NULL,         -- 'user' or 'group'
    ggid_entity_id  UUID NOT NULL,
    remote_id       VARCHAR(256),                 -- ID in ZTNA provider
    status          VARCHAR(32) NOT NULL,         -- 'pending', 'provisioned', 'failed', 'deprovisioned'
    last_sync_at    TIMESTAMPTZ,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_posture_signals_device ON device_posture_signals (tenant_id, device_id);
CREATE INDEX idx_posture_signals_user ON device_posture_signals (tenant_id, user_id);
CREATE INDEX idx_posture_policies_tenant ON device_posture_policies (tenant_id, enabled, priority);
CREATE INDEX idx_posture_results_device ON device_posture_results (tenant_id, device_id, expires_at DESC);
CREATE INDEX idx_ztna_configs_tenant ON ztna_provider_configs (tenant_id, provider);
CREATE INDEX idx_scim_state_entity ON ztna_scim_state (provider_config_id, entity_type, ggid_entity_id);
```

---

## 13. API Design

### Device Posture

```
# Get device posture
GET /api/v1/devices/{device_id}/posture

Response:
{
    "device_id": "dev_a1b2c3d4",
    "compliant": true,
    "posture_score": 85,
    "checks": { ... },
    "policy_results": [ ... ],
    "evaluated_at": "2026-07-17T09:45:32Z",
    "expires_at": "2026-07-17T09:50:32Z"
}

# Report device posture (from SDK/agent)
POST /api/v1/devices/{device_id}/posture/report
{
    "os_name": "macOS",
    "os_version": "14.5.0",
    "encrypted": true,
    "firewall_enabled": true,
    "antivirus_active": true,
    "disk_encrypted": true,
    "screen_lock_enabled": true
}

# Create posture policy
POST /api/v1/policy/device-posture
{
    "name": "corporate-managed",
    "description": "Require managed, encrypted device with current OS",
    "requirements": [
        { "field": "managed", "operator": "==", "value": true },
        { "field": "encrypted", "operator": "==", "value": true },
        { "field": "os_up_to_date", "operator": "==", "value": true },
        { "field": "edr_active", "operator": "==", "value": true }
    ],
    "priority": 10
}
```

### ZTNA Provider Management

```
# Register ZTNA provider
POST /api/v1/ztna/providers
{
    "provider": "zscaler",
    "name": "Corporate Zscaler ZPA",
    "config": {
        "saml_entity_id": "https://zscaler.net/sp/saml",
        "acs_url": "https://login.zscaler.net/sso/saml",
        "scim_endpoint": "https://api.zscaler.net/scim/v2",
        "caep_endpoint": "https://api.zscaler.net/secevent"
    },
    "scim_token": "encrypted-bearer-token",
    "enable_scim_push": true,
    "enable_caep_push": true
}

# Test provider connection
POST /api/v1/ztna/providers/{id}/test
{
    "test_type": "saml_metadata"
}

# Get setup guide (provider-specific)
GET /api/v1/ztna/providers/{id}/setup-guide

Response:
{
    "provider": "zscaler",
    "steps": [
        {
            "step": 1,
            "title": "Download SAML Metadata",
            "action": "download",
            "url": "https://ggid.corp.com/saml/idp/metadata"
        },
        {
            "step": 2,
            "title": "Configure SAML in Zscaler",
            "instructions": "In Zscaler admin: Authentication > SAML > Add Provider...",
            "terraform_snippet": "..."
        },
        {
            "step": 3,
            "title": "Configure SCIM",
            "instructions": "In Zscaler admin: Identity Provider > SCIM...",
            "api_example": "curl ..."
        }
    ]
}
```

### Continuous Verification (CAEP Events)

```
# Register CAEP event receiver (ZTNA broker provides this)
POST /api/v1/ztna/providers/{id}/caep-receiver
{
    "endpoint": "https://api.zscaler.net/secevent",
    "events_subscribed": [
        "session-revoked",
        "credential-change",
        "device-compliance-change",
        "assurance-level-change"
    ]
}

# Manually trigger event (admin action)
POST /api/v1/ztna/events/trigger
{
    "event_type": "session-revoked",
    "user_id": "uuid",
    "reason": "Security incident - suspected compromise"
}
```

---

## 14. ZTNA Provider Integration Guides

### Zscaler ZPA

**GGID Configuration**:
1. GGID SAML metadata URL: `https://ggid.corp.com/saml/idp/metadata`
2. SAML attributes: `groups`, `department`, `email`, `display_name`
3. SCIM endpoint: GGID pushes groups to Zscaler via SCIM 2.0
4. CAEP endpoint: GGID pushes security events to Zscaler

**Zscaler Configuration** (generated Terraform):
```hcl
resource "zpa_saml_attribute" "ggid_groups" {
  name = "groups"
  saml_attribute = "groups"
}

resource "zpa_policy_access_rule" "engineering" {
  name = "Engineering App Access"
  conditions {
    condition {
      lhs = "group"
      op  = "equals"
      rhs = "engineering"
    }
  }
  action = "ALLOW"
}
```

### Cloudflare Access

**GGID Configuration**:
1. Add GGID as SAML IdP in Cloudflare Zero Trust dashboard
2. GGID SAML metadata: `https://ggid.corp.com/saml/idp/metadata`
3. SCIM: GGID pushes groups to Cloudflare

**Cloudflare Configuration**:
```hcl
resource "cloudflare_access_identity_provider" "ggid" {
  account_id = var.cloudflare_account_id
  name       = "GGID"
  type       = "saml"

  config {
    issuer_url     = "https://ggid.corp.com/saml/idp/metadata"
    sso_url        = "https://ggid.corp.com/saml/idp/sso"
    certificate    = file("ggid-idp-cert.pem")
    sign_request   = false
  }
}

resource "cloudflare_access_policy" "engineering" {
  application_id = cloudflare_access_application.internal_app.id
  name           = "Engineering Access"
  precedence     = 1

  include {
    email_domain = ["corp.com"]
    saml         = ["engineering"]
  }
}
```

### Twingate

**GGID Configuration**:
1. Add GGID as OIDC IdP in Twingate admin
2. OIDC discovery: `https://ggid.corp.com/.well-known/openid-configuration`
3. SCIM: GGID pushes users/groups to Twingate

**Twingate Configuration**:
```hcl
resource "twingate_remote_network" "corp" {
  name = "Corporate Network"
}

resource "twingate_security_policy" "engineering" {
  name          = "Engineering Access"
  remote_network_id = twingate_remote_network.corp.id
  access_group_ids = [twingate_group.engineering.id]
}
```

### Tailscale

**GGID Configuration**:
1. Add GGID as OIDC IdP in Tailscale admin (ACL settings)
2. OIDC discovery: `https://ggid.corp.com/.well-known/openid-configuration`
3. Group claims map to Tailscale tags

**Tailscale ACL**:
```json
{
  "groups": {
    "group:engineering": ["engineering@corp.com"],
    "group:admin": ["admin@corp.com"]
  },
  "acls": [
    {
      "action": "accept",
      "src": ["group:engineering"],
      "dst": ["tag:internal-services:*"]
    }
  ]
}
```

---

## 15. Performance Considerations

### Posture API Latency

| Operation | Latency | Notes |
|-----------|---------|-------|
| Cached posture check (Redis) | <1ms | TTL: 5 minutes |
| Fresh posture evaluation | 5-20ms | Policy evaluation over signals |
| EDR/MDM external API poll | 50-200ms | Async, not in critical path |
| Full posture (all sources) | 50-500ms | Parallel signal gathering |

### Gateway Device Posture Check

Adding device posture check to Gateway middleware chain:

```
Existing chain: Recovery → RequestID → RateLimit → JWT → TenantContext → Proxy
New chain:      Recovery → RequestID → RateLimit → JWT → DevicePosture → TenantContext → Proxy
```

**Performance impact**: <2ms per request (Redis cache hit). Cache miss triggers async refresh.

### SCIM Push Throughput

| Operation | Batch Size | Users/Second | Notes |
|-----------|-----------|-------------|-------|
| Group membership push | 50 | ~100 | Batch SCIM requests |
| Initial full sync (10K users) | 100 | ~200 | ~50 seconds |
| Incremental sync (delta) | 10 | ~50 | Triggered on change |

---

## 16. Security Considerations

### Trust Model

| Component | Trust Relationship | Validation |
|-----------|-------------------|------------|
| GGID → ZTNA broker (SAML) | Broker trusts GGID's signed assertions | X.509 cert exchange |
| ZTNA broker → GGID (posture API) | GGID trusts broker's service token | Bearer token validation |
| GGID → ZTNA broker (SCIM push) | GGID pushes to broker's SCIM endpoint | Bearer token + TLS |
| GGID → ZTNA broker (CAEP push) | GGID pushes security events | JWT SET signed by GGID |

### Threat Mitigation

| Threat | Mitigation |
|--------|-----------|
| **Stolen ZTNA session** | CAEP `session-revoked` event terminates within seconds |
| **Compromised device** | Device posture check blocks non-compliant devices before session |
| **Privilege escalation** | SCIM group sync detects role changes, updates ZTNA policies |
| **Replay attack** | Short-lived posture cache (5 min), per-request JWT validation |
| **ZTNA broker compromise** | GGID maintains independent audit log; sessions revocable from GGID |
| **Posture API abuse** | Service token required; rate-limited; tenant-scoped |

---

## 17. Console UI Design

### ZTNA Integration Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  Zero Trust Network Access                                       │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Active ZTNA   │  │  Devices       │  │  Posture       │     │
│  │  Providers: 3  │  │  Tracked: 1247 │  │  Compliant: 89%│     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Configured Providers                                            │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ● Zscaler ZPA       Healthy   SCIM: synced 2m ago         │  │
│  │   847 users provisioned | 12 groups synced                 │  │
│  │   [Configure] [View Logs] [Test Connection]                │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Cloudflare Access  Healthy   SCIM: synced 5m ago         │  │
│  │   650 users provisioned | 8 groups synced                   │  │
│  │   [Configure] [View Logs] [Test Connection]                │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Twingate          Warning   CAEP: not configured         │  │
│  │   320 users provisioned | 5 groups synced                   │  │
│  │   [Configure] [Setup CAEP] [Test Connection]               │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  + Add ZTNA Provider                                             │
│    [Zscaler] [Cloudflare] [Twingate] [Tailscale] [Custom]      │
│                                                                  │
│  Posture Policy Compliance                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Corporate-Managed:  89% compliant (1,110 / 1,247 devices)  │  │
│  │ BYOD Allowed:       67% compliant (420 / 627 devices)      │  │
│  │ Regulated Access:   95% compliant (230 / 242 devices)      │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Recent CAEP Events                                              │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ 10:32  session-revoked      alice@corp.com  → Zscaler     │  │
│  │ 10:28  credential-change    bob@corp.com    → Cloudflare  │  │
│  │ 09:15  device-compliance    carol@corp.com  → Twingate    │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 18. Competitive Differentiation

| Feature | GGID (proposed) | Okta + Zscaler | Auth0 + Cloudflare | Keycloak + Tailscale |
|---------|-----------------|----------------|--------------------|---------------------|
| **Native IdP for ZTNA** | **Yes** | Yes (Okta) | Yes (Auth0) | Yes (Keycloak) |
| **Device posture API** | **Yes (built-in)** | Via Okta + Zscaler | Via Cloudflare | No |
| **SCIM outbound push** | **Yes** | Yes | Yes | No |
| **CAEP event push** | **Yes** | Yes (Okta) | No | No |
| **Provider setup wizard** | **Yes (4 providers)** | Via Zscaler docs | Via Cloudflare docs | Manual |
| **Posture policy engine** | **Yes** | Via Zscaler | Via Cloudflare | No |
| **Unified audit (IdP + ZTNA)** | **Yes** | Partial | No | No |
| **Self-hosted option** | **Yes** | No | No | Yes |
| **Open source** | **Yes (Apache 2.0)** | No | No | Partial |

**Key differentiator**: GGID would be the only **open-source IAM** with built-in device posture API, ZTNA provider registry, and CAEP continuous verification — eliminating the need for a separate device posture management product.

---

## 19. Implementation Backlog

### P0 — Core Integration (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Device posture data model | PostgreSQL tables for signals, policies, results | 2 days |
| 2 | Device posture API | GET /devices/{id}/posture + POST report | 3 days |
| 3 | Posture evaluation engine | Policy-based evaluation with caching (Redis) | 4 days |
| 4 | Gateway device posture middleware | Per-request posture check in middleware chain | 2 days |
| 5 | SAML groups claim | Standardized groups attribute in SAML assertions | 2 days |
| 6 | SCIM outbound client | Push users/groups to ZTNA providers | 4 days |
| 7 | ZTNA provider config registry | CRUD for provider configurations | 2 days |
| 8 | Unit tests | 90%+ coverage | 3 days |

### P1 — Continuous Verification (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 9 | CAEP event transmitter | Build + push SET tokens on security state changes | 4 days |
| 10 | Session revocation broadcast | Trigger CAEP on session/password/role changes | 2 days |
| 11 | Device compliance change detection | Monitor EDR/MDM signals, trigger CAEP on change | 3 days |
| 12 | Provider setup guide generator | Auto-generate config snippets for each provider | 3 days |
| 13 | Integration tests | End-to-end ZTNA flow tests | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 14 | ZTNA dashboard | Provider status, posture compliance, CAEP events | 3 days |
| 15 | Provider setup wizard | Multi-step wizard per provider with Terraform export | 4 days |
| 16 | Posture policy editor | Create/edit posture requirements with live preview | 2 days |
| 17 | Device list + posture | All devices with compliance status, filter, drill-down | 2 days |
| 18 | SCIM sync monitor | Show provisioning state, errors, last sync | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 19 | EDR/MDM native integration | Direct API integration with CrowdStrike, SentinelOne, Intune, Jamf |
| 20 | Posture-based conditional access | ABAC policies that reference device posture score |
| 21 | Network context in JWT claims | Add "access_method" claim (direct/vpn/ztna) to tokens |
| 22 | GGID-native ZTNA proxy | Extend Gateway as full identity-aware reverse proxy for internal apps |
| 23 | Session risk scoring | Real-time session risk based on behavioral signals |
| 24 | Multi-provider federation | Single user session valid across multiple ZTNA brokers |
| 25 | Break-glass ZTNA access | Emergency access via ZTNA with enhanced logging and approval |

---

## References

- [NIST SP 800-207: Zero Trust Architecture](https://nvd.nist.gov/pubsearch/detail.cfm?pub_id=929262) — The definitive ZT standard
- [Verizon 2025 Data Breach Investigations Report](https://www.verizon.com/business/resources/reports/dbir/) — VPN/edge device attack statistics
- [Zscaler ZPA Documentation](https://help.zscaler.com/zpa) — ZTNA broker configuration
- [Cloudflare Access Identity Providers](https://developers.cloudflare.com/cloudflare-one/integrations/identity-providers/) — SAML/OIDC integration
- [Twingate Documentation](https://docs.twingate.com/) — Overlay ZTNA setup
- [Tailscale ACLs](https://tailscale.com/kb/1018/acls) — WireGuard mesh policy
- [CAEP Specification](https://openid.net/specs/openid-caep-spec-1_0.html) — Continuous Access Evaluation Protocol
- [RFC 8417: Security Event Token (SET)](https://www.rfc-editor.org/rfc/rfc8417) — Event token format
- [RFC 8935: SET Push Delivery](https://www.rfc-editor.org/rfc/rfc8935) — Event push protocol
- [SCIM 2.0 Protocol (RFC 7644)](https://datatracker.ietf.org/doc/html/rfc7644) — Provisioning protocol
- [Tailscale vs Twingate vs Cloudflare vs Zscaler Comparison](https://technologymatch.com/blog/tailscale-vs-twingate-vs-cloudflare-access-vs-zscaler-private-access-ztna) — Technical ZTNA comparison (July 2026)
