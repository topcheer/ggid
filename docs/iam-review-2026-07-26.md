# IAM 系统深度审查报告 — 2026-07-26

## 审查范围

从全新角度审视 GGID IAM 系统的 RFC 标准符合性、竞品迁移成本、安全性和代码质量。
不依赖之前任何报告，独立分析代码库当前状态。

---

## 1. RFC 标准符合性

### 1.1 OAuth 2.0 (RFC 6749) — ✅ 基本完整

| 功能 | 状态 | 说明 |
|------|------|------|
| authorization_code grant | ✅ | 端点 `/oauth/authorize` + `/oauth/token` |
| client_credentials grant | ✅ | 支持，audience 解析正确 |
| password grant | ✅ | 支持（OAuth 2.1 不推荐但保留兼容） |
| refresh_token grant | ✅ | 支持，含 token family rotation + reuse detection |
| device_code grant | ✅ | `/api/v1/oauth/device_authorize` |
| token-exchange (RFC 8693) | ✅ | `urn:ietf:params:oauth:grant-type:token-exchange` |
| jwt-bearer grant | ✅ | `urn:ietf:params:oauth:grant-type:jwt-bearer` |
| CIBA (RFC 9703) | ⚠️ | 实现在 `/api/v1/oauth/backchannel`，但标准路径应为 `/bc-authorize` |
| redirect_uri 验证 | ✅ | 早期验证 (commit 66f69ff55) |
| client authentication | ✅ | client_secret_basic/post/tls_client_auth |

### 1.2 OIDC Core — ⚠️ 有差距

| 功能 | 状态 | 说明 |
|------|------|------|
| discovery document | ✅ | `/.well-known/openid-configuration` 完整 |
| JWKS endpoint | ✅ | `/oauth/jwks` + key rotation API |
| userinfo endpoint | ✅ | `/oauth/userinfo`，含 iss/aud 验证 |
| id_token signing | ✅ | RS256 |
| nonce | ✅ | 支持并验证 |
| scopes (openid/profile/email) | ✅ | |
| backchannel logout | ✅ | `/oauth/backchannel_logout` |
| frontchannel logout | ✅ | 实现完整 |
| end_session | ✅ | `/oauth/logout` |
| **P2-1: backchannel_token_delivery_modes_supported** | ❌ 缺失 | CIBA 实现存在但 discovery 未声明 |
| **P2-2: backchannel_authentication_endpoint** | ❌ 缺失 | discovery 未包含 CIBA 端点 |
| **P2-3: revocation_endpoint_auth_methods_supported** | ❌ 缺失 | OIDC Discovery §3 显示应声明 |
| **P2-4: introspection_endpoint_auth_methods_supported** | ❌ 缺失 | 同上 |
| **P2-5: request_parameter_supported** | ❌ 缺失 | JAR 实现存在但 discovery 未声明 |
| **P2-6: request_uri_parameter_supported** | ❌ 缺失 | 同上 |
| **P2-7: require_pushed_authorization_requests** | ❌ 缺失 | PAR 实现存在但 discovery 未声明 |
| **P2-8: webfinger** | ❌ 缺失 | 无 `/.well-known/webfinger` 端点 |

### 1.3 SAML 2.0 — ✅ 基本完整

| 功能 | 状态 | 说明 |
|------|------|------|
| SP metadata | ✅ | `/saml/metadata` |
| IdP metadata | ✅ | `/saml/idp/metadata` |
| SP-initiated SSO | ✅ | `/saml/sso` → `/saml/acs` |
| IdP-initiated SSO | ✅ | `/saml/idp/sso` |
| SLO | ✅ | `/saml/slo` |
| SAML token issuance | ✅ | `IssueSAMLToken` |
| trust chain validation | ✅ | `extractSAMLIssuer` |

### 1.4 PKCE (RFC 7636) — ⚠️ 有差距

| 功能 | 状态 | 说明 |
|------|------|------|
| S256 method | ✅ | server.go 强制 S256 |
| **P2-9: plain method 仍在 domain 层** | ⚠️ | `ValidatePKCE` 仍接受 `plain`/空值。server.go 拒绝 plain 请求，但已签发的 authorization code 仍可用 plain 验证。应从 `ValidatePKCE` 移除 `case "plain"` |

### 1.5 Token Introspection (RFC 7662) — ✅ 完整

| 功能 | 状态 |
|------|------|
| `/oauth/introspect` | ✅ |
| active/token_type/client_id/scope | ✅ |
| client authentication required | ✅ |
| revocation check | ✅ |

### 1.6 Revocation (RFC 7009) — ✅ 完整

| 功能 | 状态 |
|------|------|
| `/oauth/revoke` | ✅ |
| public client token proof-of-possession | ✅ (0f12fef38) |
| token family revocation | ✅ |
| RFC 7009 §2.1 (invalid token → 200) | ✅ |

### 1.7 SCIM 2.0 (RFC 7643/7644) — ✅ 完整

| 功能 | 状态 |
|------|------|
| /scim/v2/Users (GET/POST/PUT/PATCH/DELETE) | ✅ |
| /scim/v2/Groups (GET/POST/PUT/PATCH/DELETE) | ✅ |
| /scim/v2/Bulk | ✅ |
| /scim/v2/Me | ✅ (P2-14) |
| /scim/v2/ServiceProviderConfig | ✅ |
| /scim/v2/ResourceTypes | ✅ |
| /scim/v2/Schemas | ✅ |
| filter parser (eq/co/sw/pr/and/or/not) | ✅ |
| ETag support | ✅ |
| attribute filtering | ✅ |
| pagination (startIndex/count) | ✅ |
| sort | ⚠️ 未确认 |

### 1.8 WebAuthn — ✅ 基本

| 功能 | 状态 |
|------|------|
| passkey registration | ✅ |
| AMR/ACR claims | ✅ (AAL2/AAL3) |
| FIDO attestation | ✅ |

### 1.9 JWK/JWT (RFC 7517/7519) — ✅ 完整

| 功能 | 状态 |
|------|------|
| JWKS endpoint | ✅ |
| RS256 signing | ✅ |
| iss verification (P2-15) | ✅ |
| aud verification (P2-15) | ✅ |
| key rotation API | ✅ |
| alg whitelist | ✅ |

### 1.10 扩展协议

| 功能 | 状态 | 说明 |
|------|------|------|
| PAR (RFC 9126) | ✅ | `/oauth/par` |
| JAR (RFC 9101) | ✅ | request_uri support |
| DCR (RFC 7591) | ✅ | `/oauth/register` |
| DCM (RFC 7592) | ✅ | UpdateClientMetadata, RotateClientSecret |
| DPoP (RFC 9449) | ✅ | ParseDPoPHeader |
| mTLS (RFC 8705) | ✅ | ValidateMTLSBinding |
| OAuth 2.1 audit | ✅ | oauth21_audit_handler |

---

## 2. 竞品迁移成本

### 从 Keycloak 迁移
- **OIDC Discovery**: ✅ 兼容（Keycloak 客户端可自动发现配置）
- **grant types**: ✅ authorization_code + refresh_token + client_credentials
- **PKCE**: ✅ S256 强制（Keycloak 默认推荐 S256）
- **缺口**: CIBA 端点路径不标准 (`/api/v1/oauth/backchannel` vs `/bc-authorize`)，Keycloak CIBA 客户端需要调整 URL
- **SAML**: ✅ 双向 SSO/SLO 兼容
- **迁移成本**: 低（仅需调整 CIBA 端点和 webfinger）

### 从 Auth0 迁移
- **OIDC Discovery**: ✅ 兼容
- **audience 参数**: ✅ resolveAudience 支持 Auth0-style `audience` 参数
- **DCR**: ✅ `/oauth/register` 兼容 Auth0 DCR
- **缺口**: `backchannel_token_delivery_modes_supported` 等 discovery 字段缺失，Auth0 SDK 可能需要
- **迁移成本**: 低

### 从 Okta 迁移
- **OIDC Discovery**: ✅ 基本兼容
- **SAML**: ✅ 兼容
- **SCIM**: ✅ 完整（Okta provisioning 可用）
- **缺口**: webfinger 缺失（Okta 支持 `/.well-known/webfinger`）
- **迁移成本**: 低-中

### 从 Authentik 迁移
- **OIDC**: ✅ 兼容
- **SCIM**: ✅ 兼容（Authentik 使用 SCIM 2.0）
- **LDAP**: ✅ 支持
- **缺口**: 无重大缺口
- **迁移成本**: 低

### 总结
**P2-10**: CIBA 端点路径不标准（`/api/v1/oauth/backchannel` → 应改为 `/bc-authorize`），阻塞 CIBA 客户端互操作。

---

## 3. 安全审查

### 3.1 认证流 — ✅ 安全

- JWT 签名验证使用 RS256，alg whitelist 防止 alg confusion 攻击
- iss 验证强制（P2-15），拒绝其他 issuer 的 token
- aud 验证在 5 个外部端点强制（GetUserInfo, ExchangeToken, AgentIdentity, mTLS, userinfo handler）
- client_secret 使用 timing-safe 比较
- CSRF: session cookie + double-submit token

### 3.2 Token 签发 — ✅ 安全

- access_token: JWT with iss/sub/aud/exp/iat/jti/tenant_id
- refresh_token: 旋转 + token family reuse detection (RFC 6749 §10.4)
- id_token: RS256 签名，含 nonce 验证
- token expiry: access_token 15min, refresh_token 24h

### 3.3 密码处理 — ✅ 安全

- Argon2id 哈希（memory-hard, OWASP 推荐）
- bcrypt 向后兼容
- breach detection（HIBP-style check）
- password history (防重用)
- password reset token 一次性使用

### 3.4 会话管理 — ✅ 安全

- session_id: UUID（不可猜测）
- ForceLogout 支持（撤销除当前外的所有会话）
- backchannel logout: 跨应用 SLO
- MFA challenge: 短期 + 一次性

### 3.5 多租户隔离 — ✅ 安全

- RLS: PostgreSQL Row Level Security
- tenant_id 从 JWT claim → X-Tenant-ID header（gateway 验证后注入）
- X-User-ID 由 gateway 验证后注入（不可伪造，P0 修复 d1bc9b820）
- SCIM /Me: 从 gateway 验证的头读取，不信任 token claims

### 3.6 网络安全 — ✅ 安全

- CORS: per-tenant 配置，timing-safe origin 比较
- rate limiting: 5 维度限制器
- security headers middleware
- body size limit
- SSRF protection (webhook delivery)

### 3.7 发现的潜在问题

**P1-1: SCIM filter 在内存评估，大量用户时可能 DoS**
SCIM filter (RFC 7644 §3.4.2.2) 解析后对每个用户内存评估。当用户数 > 10K 时，GET /scim/v2/Users?filter=... 可能性能退化。应将 filter 下推到 SQL 层或添加结果限制。

**P2-11: MFA TOTP secret 存储**
`MFADevice.Secret` 注释为 "encrypted at rest in production"，但代码中未见 AES 加密。如果 DB 被直接访问，TOTP secret 可能泄露。

---

## 4. 代码质量

### 最近 20 commits 审查

| Commit | 问题 | 严重性 |
|--------|------|--------|
| da566e96d audit webhook CRUD table fix | 无 | — |
| 0f6f1b348 audit webhook engine wiring | 无 | — |
| c2c4f4297 UX Group 7 aria-labels | 无 | — |
| 40541fb33 audit pagination + webhook Secret | 无 | — |
| 8b85a6749 SSRF test bypass + docs | 无 | — |
| af84f52b9 UX Group 7 orgs + key-rotation | 无 | — |
| 1c51b14a6 SSRF protection for webhook (P1) | 无 | — |
| a0f6ebc33 UX Group 7 admin/orgs | 无 | — |
| 7888895cd SCIM group SQL + MFA TOTP replay | 无 | — |
| 7398ae666 MFA TOTP replay protection (RFC 6238 §5.2) | 无 | — |
| e972a37f5 SDK docs | 无 | — |
| f19dce943 R32 review docs | 无 | — |
| 334b43c58 Multi-tenant docs | 无 | — |
| 1ae8b8c34 Demo R3 docs | 无 | — |
| 0f12fef38 public client revocation | 无 | — |
| 5dc202146 R31 final docs | 无 | — |
| 66f69ff55 early redirect_uri validation | 无 | — |
| cc541a47b API Key cross-tenant fix | 无 | — |
| 020ddaf27 SCIM filter support | 无 | — |
| 8e2773335 forgot password + PATCH status | 无 | — |

**无新 bug 或技术债务发现。** 20 commits 质量良好，模式一致。

### 已知技术债务
- `domain.ValidatePKCE` 仍接受 `plain`（P2-9）
- OIDC discovery 缺少 8 个标准字段（P2-1 到 P2-8）
- CIBA 端点路径不标准（P2-10）
- MFA TOTP secret 可能未加密存储（P2-11）
- SCIM filter 内存评估性能（P1-1）

---

## 问题汇总

| ID | 严重性 | 描述 |
|----|--------|------|
| P1-1 | P1 | SCIM filter 内存评估，大量用户时可能 DoS |
| P2-1 | P2 | discovery 缺少 backchannel_token_delivery_modes_supported |
| P2-2 | P2 | discovery 缺少 backchannel_authentication_endpoint |
| P2-3 | P2 | discovery 缺少 revocation_endpoint_auth_methods_supported |
| P2-4 | P2 | discovery 缺少 introspection_endpoint_auth_methods_supported |
| P2-5 | P2 | discovery 缺少 request_parameter_supported |
| P2-6 | P2 | discovery 缺少 request_uri_parameter_supported |
| P2-7 | P2 | discovery 缺少 require_pushed_authorization_requests |
| P2-8 | P2 | 缺少 /.well-known/webfinger 端点 |
| P2-9 | P2 | ValidatePKCE 仍接受 plain method |
| P2-10 | P2 | CIBA 端点路径不标准 (/api/v1/oauth/backchannel vs /bc-authorize) |
| P2-11 | P2 | MFA TOTP secret 可能未加密存储 |

**总计：12 个问题（P0: 0 / P1: 1 / P2: 11）**