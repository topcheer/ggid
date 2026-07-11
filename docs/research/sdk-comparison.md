# SDK Comparison: GGID vs Competitors

> Coverage comparison of GGID SDKs vs Auth0, Clerk, and Ory SDKs.

---

## SDK Coverage Matrix

| Feature | GGID Go | GGID Node | GGID Python | GGID Java | Auth0 | Clerk | Ory |
|---------|---------|-----------|-------------|-----------|-------|-------|-----|
| JWT Verify | `VerifyToken` | `JWTVerifier` | `verify_token` | `JwtVerifier` | Yes | Yes | Yes |
| HTTP Middleware | `Middleware()` | `expressAuth()` | `GGIDMiddleware` | `GGIDAuthFilter` | Yes | Yes | Yes |
| User CRUD | `client.GetUser` | `client.getUser` | `client.get_user` | `getUser` | Yes | Yes | Yes |
| Role/Permission | `RequirePermission` | `requirePermission` | `check_permission` | `checkPermission` | Yes (Actions) | Yes | Yes (Keto) |
| Tenant Switch | `WithTenant(id)` | `switchTenant` | `switch_tenant` | `setTenant` | No | No | No |
| Agent Identity | `ExchangeAgentToken` | Planned | Planned | Planned | No | No | No |
| Auto Refresh | Yes | Yes | Manual | Manual | Yes | Yes | Manual |
| JWKS Caching | `WithJWKS(ttl)` | Built-in | Built-in | Built-in | Yes | Yes | Yes |
| WebAuthn | Planned | Planned | Planned | Planned | Yes | Yes | Yes |
| Bundle size | — | ~15KB | — | — | 35KB | 120KB | 25KB |

---

## Language Coverage

| Language | GGID | Auth0 | Clerk | Ory |
|----------|------|-------|-------|-----|
| Go | Yes | Yes | No | Yes |
| Node.js | Yes | Yes | Yes | Yes |
| React | Planned | Yes | Yes | No |
| Python | Yes | Yes | No | Yes |
| Java | Yes | Yes | No | No |
| .NET | No | Yes | No | No |
| iOS | No | Yes | Yes | No |
| Android | No | Yes | Yes | No |

---

## GGID Unique SDK Features

1. **Multi-tenant native** — `WithTenant(id)` in Go, `switchTenant()` in Node (no competitor)
2. **Agent Identity** — `ExchangeAgentToken` for AI agent delegation (no competitor)
3. **Lightweight** — Node SDK ~15KB vs Clerk 120KB
4. **ABAC check** — `RequirePermission` with ABAC policies built into SDK

---

## GGID SDK Gaps

1. **No React SDK** — planned (9-day estimate, see [React SDK Analysis](react-sdk-analysis.md))
2. **No mobile SDK** — iOS/Android not planned for v1
3. **No .NET SDK** — low demand, community can build via REST API
4. **No auto-refresh in Python/Java** — manual refresh only

---

## Recommendation

| Priority | Task | Effort |
|----------|------|--------|
| P0 | React SDK (GGIDProvider + hooks) | 9 days |
| P1 | Python auto-refresh | 1 day |
| P1 | Java auto-refresh | 1 day |
| P2 | Mobile SDKs (React Native bridge) | 5 days |
| P3 | .NET SDK | 5 days |

---

*See: [SDK Quickstart](../quickstart/sdk-quickstart.md) | [React SDK Analysis](react-sdk-analysis.md) | [3-Line Integration](../quickstart/3-line-integration.md)*

*Last updated: 2025-07-11*
