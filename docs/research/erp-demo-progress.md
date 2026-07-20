# Cross-Board ERP Demo Progress Tracker



## Auth Method Adaptation Needed (7/8 pending)

| Demo | Current Auth | Required Auth | Status |
|------|-------------|--------------|--------|
| Go | Password login | OAuth2 Auth Code + PKCE | ⏳ |
| Node | Password login | Client Credentials (M2M) | ⏳ |
| React | Password login | Auth Code + PKCE (SPA) | ⏳ |
| Python | SAML (40 refs) | SAML 2.0 SSO | ✅ Likely correct |
| C# | Password | Password Grant | ✅ |
| Java | Password/scopes | SAML 2.0 SSO | ⏳ |
| Ruby | Password login | Device Code Flow | ⏳ |
| Rust | Token verify only | Token Exchange (RFC 8693) | ⏳ |

## Issues Found



## Next Steps

1. Rebuild Python demo with fixed Dockerfile
2. Build + push Ruby/Rust Docker images
3. Adapt auth methods (coordinate with team)
4. Deploy remaining demos
5. Browser verification of deployed demos
