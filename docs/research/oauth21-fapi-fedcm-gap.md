# OAuth 2.1 / FAPI 2.0 / FedCM 研究 — 2026-07-15

## 趋势来源
- OAuth 2.1 became an RFC in 2026 (final)
- FAPI 2.0 Security Profile final, reached financial-services production
- Chrome / Edge shipped FedCM by default in 2026
- Passkeys are now baseline, not novelty

## GGID 现状

### OAuth 2.1
- `services/oauth/internal/server/oauth21_audit_handler.go` exists (60 lines)
- BUT it returns a hardcoded static response:
  - `password grant` status is "non_compliant" with static client list (c-004, c-005)
  - `PKCE` status is hardcoded as "compliant"
  - No actual client config inspection
- There is no real OAuth 2.1 compliance analyzer

### FAPI 2.0
- No dedicated FAPI 2.0 profile or configuration flag
- PKCE exists, DPoP exists, PAR exists — but not combined as a FAPI-2.0-ready profile
- No FAPI conformance test endpoint

### FedCM
- No FedCM endpoints found in gateway config or oauth server
- No credential manager / accounts endpoint
- Not a current market requirement for B2B IAM, but becoming table stakes for consumer identity

## 新 GAP（productization）

| ID | Gap | 文件 | 状态 | 优先级 | Owner |
|----|-----|------|------|--------|-------|
| 23 | OAuth 2.1 compliance audit is a stub | services/oauth/internal/server/oauth21_audit_handler.go | [NEW] | MEDIUM | backend |
| 24 | FAPI 2.0 profile/config not exposed | services/oauth | [NEW] | LOW | backend |
| 25 | FedCM not implemented | services/oauth | [NEW] | LOW | backend |

## 建议实现

### OAuth 2.1 compliance checker
- Read all OAuth clients from store
- Check each client for:
  - grant_types excludes `implicit` and `password`
  - `token_endpoint_auth_method` is S256 or private_key_jwt
  - redirect_uris use HTTPS
  - PKCE required for public clients
- Return dynamic JSON report

### FAPI 2.0 profile
- Add `fapi_2_0` boolean flag to OAuth client
- When enabled, enforce:
  - PAR only
  - PKCE S256
  - DPoP
  - response_type=code
  - Pushed authorization requests
- Add `/api/v1/oauth/fapi-config` endpoint

### FedCM (future)
- Add `/fedcm/config.json`, `/fedcm/accounts`, `/fedcm/accounts/login` endpoints
- Requires browser origin allowlist
- Lower priority for B2B IAM

## Backlog 建议
- backend: 实现 OAuth 2.1 dynamic compliance analyzer (services/oauth/internal/server/oauth21_audit_handler.go)
- backend: 实现 FAPI 2.0 profile enforcement (services/oauth/internal/server/server.go + client config)
- docs: 编写 OAuth 2.1 / FAPI 2.0 配置指南
