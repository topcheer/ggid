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

## 深度审视 2026-07-26 06:00 — R27 增量审查

### 上次审视后 4 个新 commit

1. **bf1bf6f08** fix(auth): JWT alg:none bypass 修复 — 检查所有 JWT 解析路径强制 RSA 签名。审计通过。
2. **6314e8306** fix(security): 10 bugs fixed across 7 packages — bcrypt/argon2id 编码不一致修复。审计通过。
3. **511604f77** fix(auth): WebAuthn sessionStore 后台清理 — P0 修复（arch_pm 分派）。审计通过。
4. **c04db04dd** docs: passkey fix verified。

### 新发现问题

无。3 个安全修复 commit 均为之前已识别问题的正确修复，无新引入漏洞。

### 验证结果

- `go build ./...` — PASS
- `go test ./services/auth/...` — PASS
- `go test ./services/oauth/...` — PASS
- `go test ./pkg/crypto/...` — PASS
- `go test ./pkg/auth/multihash/...` — 超时（30s），make test 10m 超时下通过（已知计算密集型）

### 上轮未修复问题状态

| 编号 | 问题 | 状态 |
|------|------|------|
| P0-14 | /oauth/revoke 无 client 认证 | **已修复已部署** |
| P1-15 | OIDC discovery 缺 end_session_endpoint | 待修复 |
| P1-16 | PKCE plain 方法 | 待修复 |
| P2-14 | SCIM 缺 /Me 端点 | 待修复 |
| P2-15 | JWT aud 运行时验证 | 待修复 |
| P0 WebAuthn sessionStore TTL | **已修复** (511604f77) |
| P1 JWT alg:none bypass | **已修复** (bf1bf6f08) |

## R28 增量审视 2026-07-26 06:15

R27 后 2 个新 commit，均为安全增强：
1. faf1435d7 — RSA key pair 验证 + 生产环境拒绝静默重新生成（修复 RSA key mismatch 根因）
2. 6c2a9d4f8 — 密码 MaxLength 限制（NIST 800-63B DoS 防护）

审计通过，无新问题。`go build` PASS。

## R29 增量审视 2026-07-26 06:30

R28 后零新增 commit。无增量问题。

## R30 增量审视 2026-07-26 06:45

R29 后 2 个新 commit：
1. 62fa77ff2 — audit repair-chain SELECT 对齐 tamper-check（NULLIF/COALESCE）— 修复审计哈希链误报
2. 62c2551eb — gateway 403 添加 request_id + PKCE 注释清理

审计通过，无新问题。`go build` PASS。
