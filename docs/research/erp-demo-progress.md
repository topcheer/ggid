# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 03:15

## Overall: Code 8/8 | Deploy 7/8 | Health OK 6/8 | Auth Adapt 2/8 | Browser Verified 4/8

| # | Lang | Code | Auth | k8s | Health | Verified | Notes |
|---|------|------|------|-----|--------|----------|-------|
| 1 | Go | ✅ | ⏳ PKCE | ✅ | ✅ HTTP+HTTPS | ✅ Browser | — |
| 2 | Node | ✅ | ⏳ M2M | ✅ | ✅ HTTP | ✅ Browser | — |
| 3 | React | ✅ | ⏳ SPA | ✅ | ✅ HTTP | ✅ Browser | — |
| 4 | Python | ✅ | ✅ SAML | ✅ | ✅ HTTPS | ✅ curl | SAML SSO, tenant ...0004 |
| 5 | C# | ✅ | ✅ Password | ✅ | ✅ HTTPS | ✅ curl | Tenant ...0005 |
| 6 | Java | ✅ | ⏳ SAML | ✅ | ✅ HTTPS | ✅ curl | — |
| 7 | Ruby | ✅ | ⏳ Device | ✅ | ⚠️ internal only | 🔲 | Sinatra HostAuth ext |
| 8 | Rust | ✅ | ⏳ TokenEx | 🔲 building | 🔲 | 🔲 | Rust 1.88 build in progress |

## Blockers
1. Ruby: External HostAuth 403 (Sinatra 4.x), internal OK. Image push slow.
2. Rust: Building with Rust 1.88 (deps need 1.86+)
3. Auth method adaptation: 6/8 need code changes

## Auth Adaptation Status
| Demo | Required | Current | Status |
|------|----------|---------|--------|
| Go | PKCE (OIDC) | Password | ⏳ |
| Node | Client Credentials (M2M) | Password | ⏳ |
| React | SPA PKCE | Password | ⏳ |
| Python | SAML SSO | SAML | ✅ |
| C# | Password Grant | Password | ✅ |
| Java | SAML SSO | Password | ⏳ |
| Ruby | Device Code | Password | ⏳ |
| Rust | Token Exchange | Password | ⏳ |
