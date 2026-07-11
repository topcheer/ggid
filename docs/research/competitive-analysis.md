# Competitive Analysis: GGID vs Auth0 vs Keycloak vs Ory vs Casdoor

> Feature comparison for choosing the right IAM platform.

---

## Quick Comparison

| Feature | GGID | Auth0 | Keycloak | Ory | Casdoor |
|---------|------|-------|----------|-----|---------|
| **License** | Apache 2.0 | Proprietary | Apache 2.0 | Apache 2.0 | Apache 2.0 |
| **Language** | Go | Node.js | Java | Go | Go |
| **Self-hosted** | Yes | No (SaaS) | Yes | Yes | Yes |
| **SDKs** | Go, Node, Python, Java | 10+ | Java, JS | Go, JS | Go, JS |
| **Integration time** | 3 lines / 5 min | 10 min | 30 min | 20 min | 15 min |
| **Multi-tenant** | RLS-based | Hosted domains | Realms | Projects | Organizations |
| **RBAC** | Yes | Yes (extension) | Yes | Yes (Ory Keto) | Yes |
| **ABAC** | Yes | Custom rules | No | Yes (Ory Keto) | No |
| **SCIM 2.0** | Yes | Yes | No | No | No |
| **Audit hash chain** | Yes | No | No | No | No |
| **WebAuthn/Passkey** | Yes | Yes | Yes | Yes (Ory Kratos) | No |
| **i18n** | 827 keys | Yes | Yes | No | Yes |
| **Compliance templates** | PCI/HIPAA/SOC2/GDPR | Enterprise plan | No | No | No |
| **Admin Console** | Next.js 15 | Hosted | Built-in | No (Ory Console) | Built-in |
| **Docker Compose** | 12 containers | N/A | 2 containers | 4 containers | 2 containers |

---

## Strengths by Platform

### GGID
- Fastest integration (3 lines)
- ABAC + compliance templates out of the box
- Tamper-evident audit hash chain
- PostgreSQL RLS for tenant isolation
- SCIM 2.0 for enterprise HR sync

### Auth0
- Largest SDK ecosystem
- Mature marketplace and integrations
- Enterprise SSO connectors
- Best documentation depth

### Keycloak
- Battle-tested (Red Hat)
- Full-featured admin UI
- Protocol support (SAML, OIDC, Kerberos)
- Large community

### Ory
- Cloud-native architecture
- Strong security focus
- Ory Keto for permission engine
- GitOps-friendly

### Casdoor
- Lightweight and fast
- Built-in UI for non-technical users
- Good for Chinese market
- Simple deployment

---

## When to Choose GGID

- You need **self-hosted** IAM with **Apache 2.0** license
- You want **RBAC + ABAC** without building custom rules
- You need **SCIM 2.0** for enterprise HR system integration
- You require **tamper-evident audit** for compliance (PCI-DSS, HIPAA)
- You want **Go performance** with a modern **Next.js admin console**
- You need **multi-tenant isolation** via PostgreSQL RLS

---

## Migration Paths

| From | Guide |
|------|-------|
| Auth0 | [Auth0 Migration](../migration-from-auth0.md) |
| Keycloak | [Keycloak Migration](../migration-from-keycloak.md) |
| Clerk | [Clerk Migration](../migration-from-clerk.md) |
| Any | [SDK Migration Guide](../guides/sdk-migration-guide.md) |

---

*See: [Feature Matrix](../feature-matrix.md) | [Gap Closure Report](gap-closure-report.md) | [Architecture](../architecture/microservices.md)*

*Last updated: 2025-07-11*
