# GGID IAM Code Review — 2026-07-21

## Review Summary

**Date:** 2026-07-21  
**Perspective:** RFC Standards Compliance (OAuth 2.0 / OIDC / PKCE / SCIM / WebAuthn / JWT)  
**Reviewer:** Automated 30-minute cron-based code review system  
**Findings:** 5 confirmed defects (P0 x1 / P1 x2 / P2 x2)

---

## Confirmed Defects

### P0-14: /oauth/revoke 无 client 认证 (RFC 7009 §2.1)

- **Severity:** P0 (Critical)
- **Category:** OAuth 2.0 / RFC 7009
- **File:** `services/oauth/internal/service/oauth_service.go`
- **Description:** The token revocation endpoint (`/oauth/revoke`) does not require client authentication. Per RFC 7009 §2.1, the revocation endpoint MUST require client authentication for confidential clients. Without this, any unauthenticated party can revoke any token, creating a DoS risk.
- **Evidence:** No client authentication check found in revocation handler
- **Recommendation:** Add client authentication to the revocation endpoint. For confidential clients, require `client_secret_basic` or `client_secret_post`. For public clients, require the `client_id` parameter and verify it matches the token's client.
- **Status:** Confirmed — needs fix

### P1-15: OIDC discovery 缺 end_session_endpoint

- **Severity:** P1 (High)
- **Category:** OIDC Core / RFC 8414
- **File:** `services/oauth/internal/service/oauth_service.go`
- **Description:** The OIDC discovery document is missing the `end_session_endpoint` field. This endpoint is required for OIDC RP-initiated logout (OIDC Front-Channel/Back-Channel Logout). Without it, OIDC clients cannot discover the logout endpoint.
- **Evidence:** `end_session_endpoint` not present in `GetDiscoveryConfig()`
- **Recommendation:** Add `end_session_endpoint` to the discovery document, pointing to `/oauth/logout`.
- **Status:** Confirmed — needs fix

### P1-16: PKCE code_challenge_method=plain 应拒绝或降级为 S256

- **Severity:** P1 (High)
- **Category:** PKCE (RFC 7636)
- **File:** `services/oauth/internal/domain/models.go`
- **Description:** The `plain` PKCE method is still supported. RFC 7636 §4.2 and OAuth 2.1 mandate S256 only. The `plain` method is vulnerable if the code challenge is leaked.
- **Evidence:** `CodeChallengeMethod string // "plain" or "S256"` in models.go
- **Recommendation:** Deprecate `plain` method. Only accept S256 for new clients. Allow `plain` only for legacy clients with a migration timeline.
- **Status:** Confirmed — needs fix

### P2-14: SCIM 缺 /Me 端点 (RFC 7643 §3.4)

- **Severity:** P2 (Medium)
- **Category:** SCIM 2.0 (RFC 7643)
- **File:** `services/identity/internal/scim/`
- **Description:** The SCIM implementation is missing the `/Me` endpoint. RFC 7643 §3.4 defines the `/Me` endpoint which allows authenticated users to retrieve and modify their own user resource without knowing their user ID.
- **Evidence:** No `/Me` endpoint found in SCIM routes
- **Recommendation:** Implement the `/Me` endpoint per RFC 7643 §3.4. This endpoint should resolve the authenticated principal to their user resource and return it.
- **Status:** Confirmed — needs fix

### P2-15: JWT access token 缺 aud 运行时验证

- **Severity:** P2 (Medium)
- **Category:** JWT (RFC 7519)
- **File:** `services/oauth/internal/service/oauth_service.go`
- **Description:** Access tokens are issued with an `aud` claim, but the resource server (gateway) does not validate the `aud` claim at runtime. Without audience validation, a token issued for one service can be used to access another service (token confusion attack).
- **Evidence:** No `aud` validation found in gateway token verification
- **Recommendation:** Add `aud` claim validation in the gateway's token verification middleware. The gateway should verify that the token's `aud` includes the expected audience for the target service.
- **Status:** Confirmed — needs fix

---

## Review Process

1. Automated review system ran at 20:00 (RFC Standards Compliance perspective)
2. Findings were sent to the ggid team PM via lanchat
3. PM performed secondary review and confirmed 5 defects
4. Defects are now assigned to teammates for fixing

## Next Review

- **Time:** 2026-07-21 20:30
- **Perspective:** Security & Migration Analysis (token security, password/session security, competitor migration gaps)

### arch_pm Review 结论 (05:45)

**P0-14 (/oauth/revoke 无 client 认证)**: arch_pm 确认修复正确，已部署。

**误报澄清（之前轮次）**:
- Passkey PKID 竞争条件 → 误报（pkSeq/fmtPKID 是死代码，实际用 UUID）
- Gateway 中间件链顺序 → 误报（Go inside-out wrapping，JWT 先执行正确）

**确认但降级**:
- AuthGuard 401 级联 → P1（已有 refresh token 重试）
- password grant → 保留（Console 登录必须，第一方应用）

**本轮新增问题已分派**:
- P0 WebAuthn sessionStore 无 TTL → ggcxf_backend
- P1 Gateway JWKAudience 默认空 → ggcxf_cli
- P1 Gateway per-route timeout 未生效 → ggcxf_cli
- P1 OAuth AltName 双格式 → ggcxf_backend
- P1 parseUUIDSafe 静默吞错 → ggcxf_backend
- P1 sysconfig broadcast 无错误处理 → ggcxf_backend

**make test 全部通过。**
