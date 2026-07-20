# Cross-Board ERP Demo Progress Tracker

> **Started**: 2026-07-21 | **Last Updated**: 2026-07-21 01:45

## Overall: Code 8/8 | Deploy 7/8 | Health OK 5/8 | Auth Adapt 2/8 | Browser Verified 0/8

| # | Lang | Code | Auth Method | Dockerfile | k8s | Health | Verified |
|---|------|------|------------|-----------|-----|--------|----------|
| 1 | Go | ✅ | ⏳ PKCE | ✅ | ✅ Run | ✅ ok | 🔲 |
| 2 | Node | ✅ | ⏳ M2M | ✅ | ✅ Run | ✅ ok | 🔲 |
| 3 | React | ✅ | ⏳ SPA PKCE | ✅ | ✅ Run | ✅ html | 🔲 |
| 4 | Python | ✅ | ✅ SAML | ✅ | ✅ Run | ⚠️ no /health | 🔲 |
| 5 | C# | ✅ | ✅ Password | ✅ | ✅ Run | ✅ ok | 🔲 |
| 6 | Java | ✅ | ⏳ SAML | ✅ | ✅ Run | ✅ ok | 🔲 |
| 7 | Ruby | ✅ | ⏳ Device | ✅ | ✅ Run | ❌ HostAuth | 🔲 |
| 8 | Rust | ✅ | ⏳ TokenEx | ✅ | 🔲 Missing | 🔲 | 🔲 |

## Known Issues

1. **Ruby HostAuth**: Sinatra 4.x Rack::Protection::HostAuthorization blocks ingress requests. Local Docker image has Sinatra::Base fix but registry push timed out. Need to re-push image.
2. **Rust**: No k8s deployment created yet.
3. **Auth method mismatch**: 6/8 demos still use password login instead of assigned auth flow.

## Next Steps

1. Re-push Ruby Docker image with Sinatra::Base fix
2. Create Rust k8s deployment
3. Begin auth method adaptation (PKCE/M2M/Device Code/Token Exchange/SAML)
4. Browser verification of deployed demos
