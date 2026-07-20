# ERP Demo Tenant + Auth Method Assignment

每个 demo 使用独立租户 + 不同认证方式，全面测试 GGID 的多租户 + 多协议支持。

## Assignment Matrix

| Demo | Tenant ID | Tenant Name | Auth Method | Protocol |
|------|-----------|-------------|-------------|----------|
| Go | 00000001-0000-0000-0000-000000000001 | Go Corp | OAuth2 Authorization Code + PKCE | OIDC |
| Node | 00000002-0000-0000-0000-000000000001 | Node Industries | OAuth2 Client Credentials | M2M |
| React | 00000003-0000-0000-0000-000000000001 | React Retail | OAuth2 Authorization Code + PKCE (frontend SPA) | OIDC |
| Python | 00000004-0000-0000-0000-000000000001 | Python Logistics | SAML 2.0 SSO | SAML |
| C# | 00000005-0000-0000-0000-000000000001 | CSharp Manufacturing | OAuth2 Password Grant | OAuth2 |
| Java | 00000006-0000-0000-0000-000000000001 | Java Enterprise | SAML 2.0 SSO | SAML |
| Ruby | 00000007-0000-0000-0000-000000000001 | Ruby Commerce | OAuth2 Device Code Flow | OAuth2 |
| Rust | 00000008-0000-0000-0000-000000000001 | Rust Systems | OAuth2 Token Exchange (RFC 8693) | OAuth2 |

## Coverage

- OAuth2 flows: 5 different grant types (Auth Code+PKCE, Client Credentials, Password, Device Code, Token Exchange)
- SAML SSO: 2 demos (Python + Java)
- Multi-tenant: 8 independent tenants
- OIDC: 2 demos with full OIDC (discovery, userinfo)
- M2M: 1 demo (Node, machine-to-machine)
- Agent Identity: 1 demo (Rust, token exchange delegation)

## GGID Features Tested

| Feature | Demo(s) |
|--------|---------|
| OAuth2 Authorization Code + PKCE | Go, React |
| OAuth2 Client Credentials | Node |
| OAuth2 Password Grant | C# |
| OAuth2 Device Code | Ruby |
| OAuth2 Token Exchange (RFC 8693) | Rust |
| SAML 2.0 SSO | Python, Java |
| JWT permissions claim | All 8 |
| Multi-tenant isolation | All 8 (different tenants) |
| RBAC + fine-grained permissions | All 8 |
| User CRUD via SDK | Go, Node, Python, Ruby, Java, C# |
| Organization hierarchy | Go, Ruby |
| Audit log | All 8 |
