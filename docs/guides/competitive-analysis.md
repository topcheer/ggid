# Competitive Analysis: GGID vs Auth0 vs Okta vs Keycloak vs Cognito vs Ping

Feature matrix, gaps, advantages, and roadmap priorities.

## Feature Matrix

| Feature | GGID | Auth0 | Okta | Keycloak | Cognito | Ping |
|---------|------|-------|------|----------|---------|------|
| **Auth Standards** | | | | | | |
| OAuth 2.0 / OIDC | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SAML 2.0 | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| SCIM 2.0 | ✅ | ✅ | ✅ | ⚠️ | ❌ | ✅ |
| WebAuthn / FIDO2 | ✅ | ✅ | ✅ | ⚠️ | ❌ | ✅ |
| Passwordless | ✅ | ✅ | ✅ | ❌ | ❌ | ⚠️ |
| **Architecture** | | | | | |
| Open Source | ✅ Apache 2.0 | ❌ | ❌ | ✅ | ❌ | ❌ |
| Self-hosted | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️ |
| Multi-tenant | ✅ (RLS) | ✅ | ✅ | ⚠️ | ✅ | ✅ |
| Microservices | ✅ (7 services) | ❌ monolith | ❌ monolith | ❌ monolith | ❌ | ❌ |
| gRPC internal | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Security** | | | | | | |
| RBAC + ABAC | ✅ | ⚠️ RBAC | ⚠️ RBAC | ✅ | ⚠️ RBAC | ✅ |
| Audit hash chain | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| PII obfuscation | ✅ | ❌ | ❌ | ❌ | ❌ | ⚠️ |
| mTLS internal | ✅ | N/A | N/A | ❌ | N/A | ⚠️ |
| Conditional access | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| **AI/Agent** | | | | | | |
| Agent identity | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| MCP auth | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Token delegation chain | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **SDK** | | | | | | |
| Go SDK | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| Node SDK | ✅ | ✅ | ✅ | ❌ | ✅ | ⚠️ |
| Java SDK | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| React SDK | ✅ | ✅ | ✅ | ❌ | ⚠️ | ❌ |
| **Deployment** | | | | | | |
| Docker Compose | ✅ | N/A | N/A | ✅ | N/A | ⚠️ |
| Kubernetes | ✅ | N/A | N/A | ✅ | N/A | ✅ |
| Multi-region | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ |
| Data residency | ✅ | ✅ | ✅ | ⚠️ | ⚠️ | ✅ |

## GGID Advantages

1. **Open source Apache 2.0** — No vendor lock-in, self-host, audit code
2. **Microservices architecture** — Scale individual services independently
3. **AI agent identity** — First IAM with native MCP auth and delegation chains
4. **Audit hash chain** — Tamper-evident audit logs (competitors lack this)
5. **ABAC engine** — Fine-grained attribute-based access (most competitors are RBAC-only)
6. **Multi-SDK** — Go, Node, Java, React SDKs with full parity
7. **Data residency control** — Self-hosted means full data sovereignty

## GGID Gaps

| Gap | Competitors That Have It | Priority |
|-----|------------------------|----------|
| Hosted managed service | Auth0, Okta, Cognito | P2 (roadmap) |
| No-code workflow builder | Okta (Lifecycle Mgmt) | P2 |
| Marketplace/integrations | Auth0, Okta | P2 |
| Universal login (hosted) | Auth0, Okta | P1 |
| Push notification MFA | Auth0, Okta | P2 |
| Social login breadth (20+) | Auth0, Okta, Cognito | P1 |
| Enterprise directory sync | Okta, Ping | P1 |
| Step-up authentication UI SDK | Auth0 | P2 |
| Adaptive MFA (ML-based) | Auth0, Okta | P2 |

## Positioning

```
Enterprise Grade ───────────────────────────── Developer Experience
     Ping ──────────── Okta ──── GGID ──── Auth0 ──── Cognito
                         Keycloak ──────┘
```

- **GGID** sits between Okta (enterprise) and Auth0 (developer-first)
- **Differentiator**: Open source microservices + AI agent identity
- **Target**: Mid-market companies wanting Okta features without vendor lock-in

## Roadmap Priorities

| Priority | Feature | Rationale |
|----------|---------|-----------|
| P0 | Hosted managed option | Remove deployment barrier |
| P0 | Universal login page | Simplify integration |
| P1 | Enterprise directory sync (AD/LDAP bi-directional) | Enterprise adoption |
| P1 | More social providers (20+) | B2C/CIAM adoption |
| P1 | Push notification MFA | Common enterprise requirement |
| P2 | No-code workflow builder | Compete with Okta Lifecycle |
| P2 | Integration marketplace | Community ecosystem |
| P2 | ML-based adaptive MFA | Advanced risk detection |

## See Also

- [Architecture Overview](../research/architecture-overview.md)
- [Authentication Flows](authentication-flows.md)
- [AI Agent Identity](ai-agent-identity.md)
- [Keycloak Migration Guide](../research/keycloak-migration.md)
