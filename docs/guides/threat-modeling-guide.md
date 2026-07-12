# Threat Modeling Guide

This guide covers STRIDE methodology, per-component threat analysis, attack tree diagrams, risk rating, mitigation mapping, and GGID's threat model.

## STRIDE Methodology

### STRIDE Categories

| Letter | Threat Category | Security Property Violated |
|---|---|---|
| S | Spoofing | Authentication |
| T | Tampering | Integrity |
| R | Repudiation | Non-repudiation |
| I | Information Disclosure | Confidentiality |
| D | Denial of Service | Availability |
| E | Elevation of Privilege | Authorization |

### STRIDE-per-Element

| Element | S | T | R | I | D | E |
|---|---|---|---|---|---|---|
| External Entity | Yes | No | Yes | No | No | Yes |
| Process | Yes | Yes | Yes | Yes | Yes | Yes |
| Data Flow | No | Yes | Yes | Yes | Yes | No |
| Data Store | No | Yes | Yes | Yes | Yes | No |

## Per-Component Threat Analysis

### Gateway

| Threat | Category | Description | Risk |
|---|---|---|---|
| Token spoofing | S | Attacker forges JWT | High |
| Request tampering | T | Modifying request in transit | Medium |
| Rate limit bypass | D | Overwhelming gateway | Medium |
| Header injection | E | X-Tenant-ID spoofing | High |

### Auth Service

| Threat | Category | Description | Risk |
|---|---|---|---|
| Credential stuffing | D | Automated login attempts | High |
| Password leak | I | Password exposure in logs | Critical |
| MFA bypass | E | Skipping MFA check | Critical |
| Token theft | I | Stealing access tokens | High |

### OAuth Service

| Threat | Category | Description | Risk |
|---|---|---|---|
| Authorization code theft | I | Intercepting auth code | High |
| Redirect URI manipulation | S | Open redirect attack | High |
| Refresh token replay | S | Reusing stolen refresh token | High |
| Client impersonation | S | Attacker pretends to be client | High |

### Identity Service

| Threat | Category | Description | Risk |
|---|---|---|---|
| User enumeration | I | Detecting valid usernames | Medium |
| PII exposure | I | Leaking user data | Critical |
| IDOR | E | Accessing other users' data | High |

### Policy Service

| Threat | Category | Description | Risk |
|---|---|---|---|
| Policy bypass | E | Circumventing policy engine | Critical |
| Role escalation | E | Gaining unauthorized role | Critical |
| Policy tampering | T | Modifying policy rules | High |

### Audit Service

| Threat | Category | Description | Risk |
|---|---|---|---|
| Audit log tampering | T | Modifying audit records | Critical |
| Audit log deletion | T | Deleting audit records | Critical |
| PII in audit logs | I | Sensitive data in logs | High |

## Attack Tree Diagrams

### Credential Theft Attack Tree

```
Credential Theft
├── Phishing
│   ├── Email phishing (likelihood: High, impact: High)
│   ├── Spear phishing (likelihood: Medium, impact: High)
│   └── Smishing (likelihood: Low, impact: High)
├── Credential Stuffing
│   ├── Breached password lists (likelihood: High, impact: High)
│   └── Password spray (likelihood: Medium, impact: High)
├── Social Engineering
│   ├── Helpdesk impersonation (likelihood: Low, impact: Critical)
│   └── Pretexting (likelihood: Low, impact: High)
└── Technical Attack
    ├── Session hijacking (likelihood: Medium, impact: High)
    ├── XSS credential theft (likelihood: Low, impact: Critical)
    └── Man-in-the-middle (likelihood: Low, impact: Critical)
```

### Token Theft Attack Tree

```
Token Theft
├── Network Interception
│   ├── No TLS (likelihood: Very Low, impact: Critical)
│   ├── TLS downgrade (likelihood: Low, impact: Critical)
│   └── Cert spoofing (likelihood: Low, impact: Critical)
├── Client-Side Theft
│   ├── XSS token extraction (likelihood: Low, impact: Critical)
│   ├── localStorage access (likelihood: Medium, impact: High)
├── Token Leakage
│   ├── Referrer header leakage (likelihood: Low, impact: Medium)
│   └── Log file exposure (likelihood: Medium, impact: High)
└── Refresh Token Reuse
    ├── Stolen refresh token (likelihood: Medium, impact: High)
    └── Family not revoked (likelihood: Low, impact: High)
```

## Risk Rating

### Likelihood x Impact Matrix

| Likelihood \ Impact | Low | Medium | High | Critical |
|---|---|---|---|---|
| Very Low | 1 | 2 | 3 | 4 |
| Low | 2 | 4 | 6 | 8 |
| Medium | 3 | 6 | 9 | 12 |
| High | 4 | 8 | 12 | 16 |

### Risk Levels

| Score | Level | Action |
|---|---|---|
| 1-4 | Low | Accept, monitor |
| 5-8 | Medium | Mitigate within 90 days |
| 9-12 | High | Mitigate within 30 days |
| 13-16 | Critical | Mitigate immediately |

## Mitigation Mapping

### STRIDE Mitigations

| Threat | Mitigation | GGID Implementation |
|---|---|---|
| Spoofing | MFA, mutual TLS, token signing | TOTP, WebAuthn, RS256 JWT, gRPC mTLS |
| Tampering | Digital signatures, hash chains | JWT signatures, audit hash chain |
| Repudiation | Audit logging, non-repudiation | Immutable audit trail, hash chain |
| Info Disclosure | Encryption, access control, PII masking | AES-256-GCM, RLS, pii.Obfuscate |
| Denial of Service | Rate limiting, circuit breakers | Token bucket, per-IP/user/tenant limits |
| Elevation of Privilege | RBAC + ABAC, least privilege, JIT | Policy engine, PAM, step-up auth |

## GGID Threat Model Summary

### Top Threats (Critical)

| # | Threat | Component | Score | Status |
|---|---|---|---|---|
| 1 | MFA bypass | Auth | 16 | Mitigated (UV flag enforced) |
| 2 | Policy bypass | Policy | 16 | Mitigated (default-deny, ABAC) |
| 3 | Audit tampering | Audit | 16 | Mitigated (hash chain, append-only) |
| 4 | PII exposure | Identity | 16 | Mitigated (encryption, RLS, masking) |
| 5 | Role escalation | Policy | 16 | Mitigated (RBAC + approval workflow) |

### Residual Risks

| Threat | Score | Residual Risk | Notes |
|---|---|---|---|
| 0-day vulnerability | 12 | Accepted | Patch within 24h of disclosure |
| Insider threat | 9 | Mitigated | PAM, access certification, audit |
| Supply chain attack | 8 | Mitigated | SLSA, dependency scanning |
| Social engineering | 8 | Accepted | User training, MFA reduces impact |

## OWASP Threat Dragon Integration

### Integration Steps

1. Create threat model diagram in Threat Dragon
2. Map each GGID component as a process
3. Add data flows between components
4. Apply STRIDE per element
5. Export model as JSON
6. Store in repository: `/docs/threat-models/ggid-threat-model.json`
7. Link mitigations to code and docs

### Model Structure

```json
{
  "title": "GGID Threat Model",
  "summary": "Identity platform with 7 microservices",
  "diagram": {
    "nodes": [
      {"id": "gateway", "type": "process", "name": "API Gateway"},
      {"id": "auth", "type": "process", "name": "Auth Service"},
      {"id": "oauth", "type": "process", "name": "OAuth Service"},
      {"id": "identity", "type": "process", "name": "Identity Service"},
      {"id": "policy", "type": "process", "name": "Policy Service"},
      {"id": "audit", "type": "process", "name": "Audit Service"},
      {"id": "db", "type": "store", "name": "PostgreSQL"},
      {"id": "redis", "type": "store", "name": "Redis"}
    ],
    "edges": [
      {"from": "gateway", "to": "auth", "label": "JWT verification"},
      {"from": "gateway", "to": "identity", "label": "User CRUD"},
      {"from": "gateway", "to": "policy", "label": "Policy eval"}
    ]
  },
  "threats": [
    {
      "id": "T001",
      "title": "JWT Spoofing",
      "type": "Spoofing",
      "status": "mitigated",
      "severity": "high",
      "mitigation": "RS256 signature verification via JWKS"
    }
  ]
}
```

## Best Practices

1. **Model before implementing** — Threat model during design phase
2. **Update regularly** — Reassess when architecture changes
3. **Rate every threat** — Use likelihood x impact consistently
4. **Map to mitigations** — Every threat should have a mitigation
5. **Track residual risk** — Document what's accepted vs mitigated
6. **Involve the team** — Developers, security, and operations
7. **Use diagrams** — Visual models are easier to understand
8. **Test mitigations** — Verify controls actually work
9. **Prioritize by risk** — Fix critical threats first
10. **Document assumptions** — Record what was assumed during modeling