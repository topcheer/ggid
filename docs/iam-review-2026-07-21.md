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

## 深度审视 2026-07-26 07:00 — R31 全新角度安全扫描

### R30 后 1 个新 commit

**41f960400** fix(identity): SetUserStatus to active clears deleted_at — 数据完整性修复，防止已恢复用户的凭证被清理脚本删除。审计通过。

### 全新角度安全扫描结果

| 检查项 | 结果 | 说明 |
|--------|------|------|
| SQL injection | PASS | 所有查询用 `$N` 参数化占位符，whereClause 通过 strings.Join 构建，无拼接风险 |
| 时序攻击（密码比较） | PASS | 密码验证走 argon2id/bcrypt，无直接 `==` 比较 |
| CORS 配置 | PASS | per-tenant CORS，fallback 非 `*` 通配符 |
| Rate limiting | PASS | 登录有 MaxAttempts + lockout，OTP 有 3次/小时限制 |
| JWT expiry 验证 | PASS | ParseAccessToken 检查 exp claim，过期返回错误 |
| JWT alg:none | PASS | 所有 JWT 解析路径强制 RSA 签名方法验证（之前已修复） |
| redirect_uri 匹配 | PASS | 精确匹配（==），非前缀匹配 |
| PKCE 强制 | PASS | 公开客户端强制 PKCE（OAuth 2.1 mandate） |
| Refresh token reuse | PASS | token family registry 检测重放 |

### 新发现问题

**无新问题。** 系统安全状态良好。

### 待修复项（已知，无变化）

| 编号 | 问题 | 优先级 |
|------|------|--------|
| P1-15 | OIDC discovery 缺 end_session_endpoint | P1 |
| P1-16 | PKCE plain 方法应拒绝/降级为 S256 | P1 |
| P2-14 | SCIM 缺 /Me 端点 | P2 |
| P2-15 | JWT access token 缺 aud 运行时验证 | P2 |

## R31 补充 — RFC 合规性逐条检查 + 竞品迁移成本

### 1. RFC 标准符合性

| RFC | 功能 | 端点 | 状态 | 差距 |
|-----|------|------|------|------|
| RFC 6749 | OAuth 2.0 authorize | /oauth/authorize | PASS | — |
| RFC 6749 | OAuth 2.0 token | /oauth/token | PASS | — |
| RFC 6749 | password grant | 在 /oauth/token | PASS | 保留（第一方 Console 登录，arch_pm 确认） |
| RFC 6749 | client_credentials | 在 /oauth/token | PASS | — |
| RFC 6749 | refresh_token | 在 /oauth/token | PASS | — |
| RFC 6749 | token revocation | /oauth/revoke | PASS | P0-14 已修复（加 client 认证） |
| RFC 7009 | Revocation client auth | /oauth/revoke | PASS | 已修复 |
| RFC 7662 | Token introspection | /oauth/introspect | PASS | client 认证 + 标准字段 |
| RFC 7636 | PKCE | /oauth/authorize | **不一致** | **P1-16: 实际支持 plain 但 discovery 只声明 S256** |
| RFC 8628 | Device flow | /api/v1/oauth/device_authorize | PASS | — |
| RFC 7591 | DCR | /oauth/register | PASS | — |
| RFC 9126 | PAR | /oauth/par | PASS | — |
| OIDC Core | Discovery | /.well-known/openid-configuration | PASS | — |
| OIDC Core | UserInfo | /oauth/userinfo | PASS | — |
| OIDC Core | JWKS | /oauth/jwks | PASS | — |
| OIDC Core | end_session_endpoint | /oauth/logout | **PASS** | **P1-15 误报修正：discovery 已包含 EndSessionEndpoint** |
| OIDC Core | id_token RS256 | /oauth/token | PASS | — |
| OIDC Core | backchannel logout | /api/v1/oauth/backchannel-logout | PASS | — |
| RFC 8414 | OAuth server metadata | /.well-known/oauth-authorization-server | PASS | — |
| SAML 2.0 | Metadata | /saml/metadata | PASS | — |
| SAML 2.0 | SSO/ACS | /saml/sso, /saml/acs | PASS | — |
| SCIM 7643 | /Users | /scim/v2/Users | PASS | — |
| SCIM 7643 | /Groups | /scim/v2/Groups | PASS | — |
| SCIM 7643 | /Bulk | /scim/v2/Bulk | PASS | — |
| SCIM 7643 | /ServiceProviderConfig | /scim/v2/ServiceProviderConfig | PASS | — |
| SCIM 7643 | /ResourceTypes | /scim/v2/ResourceTypes | PASS | — |
| SCIM 7643 | /Schemas | /scim/v2/Schemas | PASS | — |
| SCIM 7644 | PATCH | /scim/v2/Users/{id} | PASS | — |
| SCIM 7643 | ETag | — | PASS | — |
| SCIM 7643 | /Me | — | **缺失** | **P2-14: 无 /scim/v2/Me 端点** |
| WebAuthn | register | /api/v1/auth/webauthn/register/begin+finish | PASS | — |
| WebAuthn | passwordless | /api/v1/auth/webauthn/passwordless/begin+finish | PASS | — |
| WebAuthn | autofill | /api/v1/auth/webauthn/autofill | PASS | — |
| RFC 7517 | JWK | /oauth/jwks | PASS | RSA key, kid, RS256 |
| RFC 7519 | JWT | access token + id token | **不一致** | **P2-15: token 签发含 aud/iss 但 ParseAccessToken 不验证 aud** |

### 2. 竞品迁移成本评估

| 维度 | 状态 | 说明 |
|------|------|------|
| OIDC discovery | PASS | 标准端点，含 issuer/auth/token/userinfo/jwks/revoke/introspect/end_session/device/register/par |
| OAuth server metadata | PASS | RFC 8414 /.well-known/oauth-authorization-server |
| SCIM 2.0 | 基本PASS | Users/Groups/Bulk/PATCH/ETag 齐全，缺 /Me |
| SDK | PASS | 13 种语言 SDK (Go/Java/Python/Node/Ruby/Rust/C#/Dart/PHP/React/ReactNative/curl) |
| DCR | PASS | RFC 7591 /oauth/register |
| PAR | PASS | RFC 9126 /oauth/par |
| Device flow | PASS | RFC 8628 |
| SAML 2.0 | PASS | metadata + SSO + ACS |
| WebAuthn | PASS | register + passwordless + autofill |
| **迁移差距** | | 1. SCIM /Me 缺失 2. JWT aud 运行时验证缺失 3. PKCE plain 支持与 discovery 声明不一致 |

### 3. 安全扫描结果

| 检查项 | 结果 |
|--------|------|
| SQL injection | PASS — 所有查询参数化 |
| 时序攻击 | PASS — argon2id/bcrypt |
| CORS | PASS — per-tenant，非通配符 |
| Rate limiting | PASS — login lockout + OTP 3/h |
| JWT alg:none | PASS — 强制 RSA |
| JWT expiry | PASS — exp 验证 |
| redirect_uri | PASS — 精确匹配 |
| PKCE 强制 | PASS — 公开客户端强制 |
| Refresh token reuse | PASS — family registry |
| /oauth/revoke auth | PASS — P0-14 已修复 |

### 4. 代码质量 — 最近 commit

- 41f960400 SetUserStatus active 清理 deleted_at — 正确修复数据完整性问题
- 62c2551eb gateway 403 request_id — 正确
- 62fa77ff2 audit repair-chain NULLIF/COALESCE — 正确

### 修正

- **P1-15 误报修正**：OIDC discovery **已包含** end_session_endpoint，无需修复
- **P1-16 降级为 P2-16**：discovery 声明 S256 only，但代码接受 plain — 不影响互操作（客户端按 discovery 走 S256 没问题），仅是内部不一致
