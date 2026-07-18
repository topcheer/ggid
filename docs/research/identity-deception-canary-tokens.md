# Identity Deception Technology — Canary Tokens & Honey Identities

> Research Date: 2026-07-18
> Status: GAP — partial code exists, no research or strategy
> Priority: P2 (security hardening)

## Executive Summary

Deception technology plants fake credentials, API keys, and identity artifacts (canary tokens) that trigger alerts when accessed. GGID has `canary.go` and `shadow_mirror.go` middleware but lacks a comprehensive deception strategy covering honey-credentials, fake API keys, and decoy user accounts.

## What Is Identity Deception?

Identity deception deploys trap identities that appear legitimate to attackers. When an attacker uses a planted honey-token credential, it instantly reveals their presence — often before any real damage occurs.

### Types of Deception Tokens

| Token Type | Description | Detection Signal |
|-----------|-------------|-----------------|
| **Honey credentials** | Fake username/password pairs in config files | Login attempt with honey creds → instant alert |
| **Canary API keys** | Fake API keys planted in repos/configs | API call with canary key → attacker identified |
| **Decoy user accounts** | Fake admin accounts in directory | Any access to decoy account → intrusion confirmed |
| **Fake SAML/OAuth clients** | Planted client credentials | Token request with fake client → breach detected |
| **Honey file shares** | Decoy paths with tempting names | Access to honey path → lateral movement detected |

## Current GGID State

| Component | Status | Location |
|-----------|--------|----------|
| `canary.go` middleware | Exists (basic) | `services/gateway/internal/middleware/canary.go` |
| `shadow_mirror.go` | Exists (shadow API routes) | `services/gateway/internal/middleware/shadow_mirror.go` |
| Honey credentials | Missing | — |
| Canary API key generation | Missing | — |
| Decoy account management | Missing | — |
| Alert integration | Missing | — |

## Competitive Landscape

| Vendor | Feature |
|--------|---------|
| **Thinkst Canary** | Enterprise canary tokens, integrated alerts |
| **Attivo Networks** | Identity deception for AD, cloud IAM |
| **Microsoft Defender** | Honeytoken alerts in Entra ID |
| **Okta** | No native deception (relies on integrations) |

## Proposed Architecture

```
Admin plants canary tokens via console
         ↓
   ┌─────────────────────────────┐
   │ Token Registry (PG table)   │
   │ - token_hash (SHA-256)      │
   │ - type (cred/apikey/user)   │
   │ - planted_at timestamp      │
   │ - alert_channels            │
   └─────────────────────────────┘
         ↓
   Gateway middleware intercepts ALL auth attempts
         ↓
   Hash check against token registry
         ↓
   Match? → ITDR Critical Alert + Session IP capture
         ↓
   No match? → Normal flow
```

## Gap Items

### KB-230: Honey credential generation + detection
**Type**: Backend (services/audit)
**Priority**: P2
Generate fake credential pairs, register in canary token registry, detect usage at gateway auth layer.

### KB-231: Canary API key planting + monitoring
**Type**: Backend (services/gateway)
**Priority**: P2
Generate fake API keys with distinct prefixes, monitor for usage, alert on first use with source IP.

### KB-232: Decoy account lifecycle management
**Type**: Backend (services/identity)
**Priority**: P2
Create decoy admin/service accounts that trigger alerts when accessed or queried.

### KB-233: Deception dashboard in console
**Type**: Frontend (console/src)
**Priority**: P2
Console page for planting tokens, viewing hit alerts, and managing decoy identities.

## Implementation Estimate

| Component | Effort |
|-----------|--------|
| Token registry + generation | 2d |
| Gateway middleware integration | 2d |
| ITDR alert integration | 1d |
| Console dashboard | 2d |
| **Total** | **~7d** |

## Business Value

- **Early breach detection**: Identify credential theft before real data is accessed.
- **Attacker attribution**: Capture source IPs, user agents, and attack patterns.
- **Low false positive**: Canary tokens are never used legitimately — any hit is a real threat.
- **Compliance value**: Demonstrates proactive threat detection for SOC2/ISO 27001.
