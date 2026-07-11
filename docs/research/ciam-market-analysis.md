# CIAM Market Analysis: Competitive Landscape & TCO

## Overview

This document analyzes the Customer Identity and Access Management (CIAM) market, comparing major vendors — Okta, Auth0, Keycloak, AWS Cognito, Ping Identity, Microsoft Entra — against GGID on features, pricing, and total cost of ownership.

> **Related**: [CIAM Market 2026](ciam-market-2026.md), [Pricing Comparison](pricing-comparison-2026.md), [Competitive Analysis](competitive-analysis.md), [Okta/Duo 2026](okta-duo-2026.md)

## Market Size & Growth

| Year | Market Size | Growth Rate |
|------|-------------|-------------|
| 2023 | $8.3B | — |
| 2024 | $10.2B | 23% |
| 2025 | $12.8B | 25% |
| 2026 (proj) | $16.1B | 26% |
| 2030 (proj) | $38.5B | ~25% CAGR |

**Key drivers**: Zero trust adoption, AI agent identity, passkey adoption, regulatory compliance (GDPR, PSD2), cloud migration.

## Vendor Comparison Matrix

### Feature Comparison

| Feature | Okta | Auth0 | Keycloak | Cognito | Ping | Entra ID | **GGID** |
|---------|------|-------|----------|---------|------|----------|----------|
| **Core Auth** |
| Password | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| MFA TOTP | Yes | Yes | Yes | No | Yes | Yes | Yes |
| WebAuthn | Yes | Yes | Partial | No | Yes | Yes | Yes (7 fmt) |
| Passwordless | Yes | Yes | Partial | No | Yes | Yes | Yes |
| **Federation** |
| SAML 2.0 | Yes | Yes | Yes | No | Yes | Yes | Yes |
| OIDC | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Social login | Yes | Yes | Yes | Yes | Yes | Yes | 9 providers |
| LDAP | Yes | Add-on | Yes | No | Yes | Yes | Yes |
| **Authorization** |
| RBAC | Yes | Yes | Yes | Groups | Yes | Yes | Yes |
| ABAC | Yes | Add-on | Partial | No | Yes | Yes | Yes |
| Policy engine | Yes | No | No | No | Yes | Yes | Yes |
| **Developer** |
| SDK (Go) | No | No | No | Yes | No | No | **Yes** |
| SDK (Node) | Yes | Yes | No | Yes | No | Yes | **Yes** |
| SDK (Java) | Yes | Yes | No | Yes | No | Yes | **Yes** |
| SCIM 2.0 | Yes | Yes | Yes | No | Yes | Yes | Yes |
| GraphQL | No | No | No | No | No | No | No |
| **Security** |
| Audit hash chain | No | No | No | No | No | No | **Yes** |
| SIEM forwarding | Yes | Add-on | No | CW | Yes | Yes | Yes |
| PII obfuscation | Partial | No | No | No | Partial | Partial | Yes |
| RLS multi-tenant | Yes | Yes | No | Yes | Yes | Yes | Yes |
| **AI/Agents** |
| Agent identity | No | No | No | No | No | No | **Yes** |
| Token delegation | Partial | No | No | No | No | Partial | Yes |
| MCP auth | No | No | No | No | No | No | **Yes** |
| **Deployment** |
| Self-hosted | No | No | Yes | No | Yes | No | **Yes** |
| Multi-cloud | Yes | Yes | Self | AWS | Yes | Azure | **Yes** |
| Open source | No | No | Yes | No | No | No | **Yes** |

### Pricing Comparison (Monthly, 10,000 MAU)

| Vendor | Base Price | Per-MAU | MFA Add-on | Total/mo | Annual TCO |
|--------|-----------|---------|------------|----------|------------|
| **Okta** (WIC) | $2/MAU | — | $1/MAU | $30,000 | $360,000 |
| **Auth0** (B2C) | $0.07/auth | — | $0.013/auth | ~$23,000 | ~$276,000 |
| **Keycloak** | $0 | Self-host | $0 | ~$2,000 (infra) | ~$24,000 |
| **AWS Cognito** | $0.0055/MAU | — | $0.015/MAU | $205 | $2,460 |
| **Ping** | Custom | — | Included | ~$25,000 | ~$300,000 |
| **Entra ID** P2 | $0.05/user/mo | — | Included | $500 | $6,000 |
| **GGID** (self) | $0 | Self-host | $0 | ~$2,000 (infra) | ~$24,000 |

> MAU = Monthly Active Users. Pricing as of early 2025, may vary by contract.

### TCO Analysis (3-Year, 10K MAU)

| Cost Category | Okta | Auth0 | Cognito | Keycloak | GGID |
|---------------|------|-------|---------|----------|------|
| Licensing | $1,080,000 | $828,000 | $7,380 | $0 | $0 |
| Infrastructure | $0 | $0 | $0 | $72,000 | $72,000 |
| Engineering (ops) | $60,000 | $60,000 | $90,000 | $150,000 | $120,000 |
| Support/SLA | $120,000 | $90,000 | AWS support | $0 | $0 |
| **3-Year TCO** | **$1,260,000** | **$978,000** | **$97,380** | **$222,000** | **$192,000** |

**GGID advantage**: 84% cheaper than Okta, 80% cheaper than Auth0, comparable to Keycloak but with Go performance and AI agent features.

## Market Segments

### Enterprise (1M+ MAU)

| Need | Best Fit | Why |
|------|----------|-----|
| Full-featured CIAM | Okta, Ping | Mature, enterprise support |
| Developer-first | Auth0 | Excellent DX, actions |
| Cost-conscious | GGID, Keycloak | Self-hosted, no per-user cost |
| Azure shop | Entra ID | Native Azure integration |

### Mid-Market (10K-1M MAU)

| Need | Best Fit | Why |
|------|----------|-----|
| Quick start | Auth0, Cognito | Managed, no ops |
| Custom control | GGID, Keycloak | Open source, self-hosted |
| Budget | Cognito, GGID | Low cost at scale |

### Developer/Startup (<10K MAU)

| Need | Best Fit | Why |
|------|----------|-----|
| Free tier | Cognito, Keycloak | $0 at low volume |
| AI agents | GGID | Only option with native agent identity |
| Go stack | GGID | Native Go SDK |

## GGID Differentiators

### Unique to GGID

1. **AI Agent Identity** — First IAM with native MCP auth, delegation chains, agent token exchange
2. **Audit Hash Chain** — Tamper-evident audit log (cryptographic chain)
3. **Go-native** — All SDKs and services in Go (best for Go microservice shops)
4. **7 Attestation Formats** — Full WebAuthn attestation verification (most only do 1-2)
5. **Apache 2.0 Open Source** — No vendor lock-in, self-hostable

### GGID Gaps vs Competitors

| Gap | Priority | Competitor Advantage |
|-----|----------|---------------------|
| Hosted/managed offering | P1 | Okta/Auth0 fully managed |
| Pre-built UI components | P2 | Auth0 Universal Login |
| Adaptive/risk-based auth | P1 | Okta/Entra risk engine |
| Behavioral biometrics | P2 | Okta (enterprise) |
| Workflow automation | P2 | Okta Lifecycle, Auth0 Actions |
| Marketplace/integrations | P2 | Okta Integration Network (7000+) |

## Selection Decision Tree

```
Need hosted/managed?
├─ Yes → Budget > $10K/mo?
│        ├─ Yes → Okta or Auth0
│        └─ No  → Cognito or Entra ID
└─ No (self-hosted OK)
     ├─ Need AI agent identity?
     │   ├─ Yes → GGID (only option)
     │   └─ No  → Need enterprise support?
     │            ├─ Yes → Ping Identity
     │            └─ No  → GGID or Keycloak
```

## References

- [Gartner MQ: Access Management (2024)](https://www.gartner.com/reviews/market/access-management)
- [KuppingerCole Leadership Compass CIAM](https://www.kuppingercole.com/research/lciam)
- [Forrester Wave: Identity-As-A-Service](https://www.forrester.com/report/the-forrester-wave/)

## See Also

- [CIAM Market 2026](ciam-market-2026.md)
- [Pricing Comparison 2026](pricing-comparison-2026.md)
- [Competitive Analysis](competitive-analysis.md)
- [Migration from Auth0/Okta](../guides/migration-from-auth0-okta.md)
