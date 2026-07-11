# Zero Trust Architecture: NIST 800-207 Mapping & GGID Readiness

## Overview

Zero Trust replaces perimeter-based security with continuous verification of every request, regardless of network location. This document maps the NIST SP 800-207 Zero Trust Architecture standard and Google's BeyondCorp model to GGID's current capabilities and identifies gaps.

> **Related**: [Zero Trust IAM Implementation](zero-trust-iam.md) (1198 lines, comprehensive design), [Zero Trust Architecture Guide](../guides/zero-trust-architecture.md) (brief overview)

## NIST SP 800-207: Zero Trust Tenets

NIST SP 800-207 defines seven tenets of Zero Trust:

### Tenet 1: Data Source Integrity

> All data sources and computing services are considered resources.

| GGID Implementation | Status |
|---------------------|--------|
| Every service (gateway, identity, auth, oauth, policy, org, audit) is an individually authenticated resource | Done |
| gRPC TLS between services (policy, org) | Done |
| gRPC TLS for remaining services (identity, auth, audit) | Gap |

### Tenet 2: All Communication Secured

> All communication is secured regardless of network location.

| GGID Implementation | Status |
|---------------------|--------|
| TLS for all external traffic (HTTPS) | Done |
| gRPC TLS between gateway and services | Partial |
| LDAP START_TLS support | Partial |
| NATS TLS | Gap |
| Redis TLS | Gap |

### Tenet 3: Per-Session Access

> Access to individual enterprise resources is granted on a per-session basis.

| GGID Implementation | Status |
|---------------------|--------|
| JWT validation on every request | Done |
| JWT `jti` anti-replay (Redis SETNX) | Done |
| Session timeout enforcement | Done |
| No persistent VPN or network-level trust | Done |

### Tenet 4: Dynamic Policy

> Access to resources is determined by dynamic policy — identity, application/service, security posture, time, location.

| GGID Implementation | Status |
|---------------------|--------|
| Identity-based policy (JWT claims) | Done |
| RBAC + ABAC policy engine | Done |
| Device posture evaluation | Gap |
| Time-based access rules | Gap |
| Location/geo-based policy | Partial (geoip.go exists) |
| Risk-based adaptive policy | Gap (roadmap) |

### Tenet 5: Integrity & Security Posture

> The enterprise monitors and measures the integrity and security posture of all owned and associated assets.

| GGID Implementation | Status |
|---------------------|--------|
| Audit logging with hash chain | Done |
| Health endpoints per service | Done |
| Device registry/fingerprinting | Gap |
| Security posture scoring | Gap |
| Continuous monitoring (SIEM forward) | Partial |

### Tenet 6: Authentication & Authorization Dynamic

> All resource authentication and authorization are dynamic and strictly enforced before access is allowed.

| GGID Implementation | Status |
|---------------------|--------|
| Per-request JWT validation | Done |
| Per-request scope enforcement (`HasScope()`) | Done |
| Per-request tenant RLS enforcement | Done |
| Dynamic re-evaluation (CAEP events) | Gap |
| Real-time policy revocation | Partial (jti blacklist) |

### Tenet 7: Telemetry Collection

> The enterprise collects as much information as possible about the current state of assets, network infrastructure, and communications.

| GGID Implementation | Status |
|---------------------|--------|
| Audit events for all auth/policy/admin actions | Done |
| Request logging (method, path, status, latency) | Done |
| Rate limit hit tracking | Done |
| Circuit breaker state monitoring | Done |
| Behavioral analytics | Gap |
| UEBA (User & Entity Behavior Analytics) | Gap |

## BeyondCorp Model Comparison

Google's BeyondCorp pioneered the zero trust access pattern with five core principles:

| BeyondCorp Principle | Description | GGID Implementation |
|---------------------|-------------|---------------------|
| Device inventory & trust | Every device is inventoried and assigned a trust level | Gap — no device registry |
| User trust | Users authenticated via MFA, SSO | Done — MFA TOTP, WebAuthn, LDAP, OAuth |
| Trust-based access | Access decisions based on user + device trust | Partial — user trust only |
| Dynamic access rules | Rules evaluated per-request | Done — JWT + RBAC/ABAC |
| Access proxy pattern | All access through identity-aware proxy | Done — API Gateway |

### BeyondCorp Access Proxy vs GGID Gateway

```
BeyondCorp Access Proxy                GGID API Gateway
─────────────────────                  ────────────────
Device certificate validation          JWT signature validation
User identity (SSO)                    User identity (JWT claims)
Device trust score                     IP allowlist + rate limit
Per-app access policy                  Per-route RBAC scope
Continuous re-evaluation               Per-request validation
```

## Zero Trust Pillars (CISA Model)

The CISA Zero Trust Maturity Model defines five pillars:

### Pillar 1: Identity

| Capability | GGID Status | Maturity Level |
|------------|-------------|----------------|
| MFA (TOTP) | Done | Advanced |
| WebAuthn/FIDO2 | Done | Optimal |
| SSO (SAML, OIDC) | Done | Advanced |
| Risk-based authentication | Gap | Traditional |
| Identity proofing | Gap | Traditional |

**Maturity: Advanced**

### Pillar 2: Devices

| Capability | GGID Status | Maturity Level |
|------------|-------------|----------------|
| Device registry | Gap | Traditional |
| Device health attestation | Gap | Traditional |
| Managed device policy | Gap | Traditional |
| Certificate-based device auth | Partial (WebAuthn) | Initial |

**Maturity: Traditional** (biggest gap)

### Pillar 3: Networks

| Capability | GGID Status | Maturity Level |
|------------|-------------|----------------|
| Network segmentation | Done (microservices) | Advanced |
| Encrypted transport (mTLS) | Partial | Initial |
| Macro-segmentation (per-tenant) | Done (RLS) | Advanced |
| Micro-segmentation (per-service) | Done | Advanced |

**Maturity: Advanced**

### Pillar 4: Applications & Workloads

| Capability | GGID Status | Maturity Level |
|------------|-------------|----------------|
| API authentication | Done (JWT + API keys) | Advanced |
| Per-request authorization | Done (RBAC + ABAC) | Optimal |
| Container security | Partial (Docker) | Initial |
| Service mesh (mTLS everywhere) | Gap | Traditional |

**Maturity: Advanced**

### Pillar 5: Data

| Capability | GGID Status | Maturity Level |
|------------|-------------|----------------|
| Data classification | Gap | Traditional |
| Encryption at rest | Partial (DB-level) | Initial |
| Encryption in transit | Done (TLS) | Advanced |
| Data loss prevention (DLP) | Partial (PII obfuscation) | Initial |
| Data retention policies | Done (retention.go) | Advanced |

**Maturity: Initial**

## GGID Zero Trust Readiness Score

| Pillar | Maturity | Weight | Score |
|--------|----------|--------|-------|
| Identity | Advanced (3/4) | 25% | 18.75/25 |
| Devices | Traditional (0.5/4) | 20% | 2.5/20 |
| Networks | Advanced (3/4) | 15% | 11.25/15 |
| Applications | Advanced (3/4) | 25% | 18.75/25 |
| Data | Initial (2/5) | 15% | 6/15 |
| **Total** | | **100%** | **57.25/100** |

**Overall: "Initial-to-Advanced Transition"**

## Priority Gap Closure

| Priority | Gap | Effort | Impact on Score |
|----------|-----|--------|-----------------|
| P0 | Device registry & fingerprinting | Large | +8 (Devices→Initial) |
| P1 | mTLS for all internal services | Medium | +5 (Networks→Optimal) |
| P1 | Risk-based authentication | Medium | +5 (Identity→Optimal) |
| P1 | Data classification & DLP | Medium | +4 (Data→Advanced) |
| P2 | Continuous Access Evaluation (CAEP) | Large | +3 (Applications→Optimal) |
| P2 | Behavioral analytics (UEBA) | Large | +3 (Telemetry) |

## Industry Zero Trust Frameworks

| Framework | Focus | GGID Alignment |
|-----------|-------|----------------|
| NIST SP 800-207 | Architecture tenets | 5/7 tenets substantially met |
| CISA ZTMM | Maturity model (5 pillars) | Advanced in 3/5 pillars |
| DoD ZT Reference | 7 pillars | Identity + Network strong |
| BeyondCorp | Access proxy pattern | Gateway implements this |
| OSA ZT Capabilities | Capability catalog | 60% coverage |

## References

- [NIST SP 800-207: Zero Trust Architecture](https://csrc.nist.gov/pubs/sp/800/207/final)
- [CISA Zero Trust Maturity Model 2.0](https://www.cisa.gov/zero-trust-maturity-model)
- [Google BeyondCorp: A New Approach to Enterprise Security](https://research.google/pubs/beyondcorp-a-new-approach-to-enterprise-security/)

## See Also

- [Zero Trust IAM Implementation](zero-trust-iam.md)
- [STRIDE Threat Analysis](stride-analysis.md)
- [Adaptive Authentication](adaptive-authentication.md)
- [Security Overview](../architecture/security-overview.md)
