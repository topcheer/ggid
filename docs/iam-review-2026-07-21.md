# GGID IAM 系统审视报告 — 2026-07-21

> 审视人：god_fullstack&everything（定期审视任务 cron-1 首轮）
> 方法：三路并行代码级审查（OAuth/OIDC/SAML · Auth/Identity/SCIM · 增强功能兼容性+竞品迁移），所有结论均有 file:line 证据。
> 状态：**R1 完成（已部署生产）。** R2/R3 无增量。R4（02:43）：发现 P0 安全回归 — commit 31c7e5c1e 跨租户修复重新引入裸角色名提权，已修复（be7e939ea），待 arch_pm review + 重建 gateway。

---

## 七、R4 增量审视（2026-07-22 02:43 CST）

### 发现：P0 安全回归（跨租户 token + 裸角色名提权）

erp-demo dimension 4 发现跨租户 token 被接受（commit 72cdef176），commit 31c7e5c1e 添加了 JWTAuth 租户边界校验。但修复代码的 platform admin 绕过检查了裸角色名 `"admin"` / `"administrator"` / `"platform administrator"`（middleware.go:672），**重新引入了 guardian_security 6b97c7a54 刚修复的同一提权漏洞**。

**风险**：任何租户创建名为 "admin" 的角色，该用户即可跨租户访问。

### 修复（be7e939ea）

middleware.go JWTAuth platform admin 判定改为：
- scope claim（空格分隔字符串）中匹配 `"platform:admin"`
- roles claim（string array）中匹配 `"platform:admin"`
- 不再接受裸角色名 `"admin"` / `"administrator"`

测试：`go build ./...` + `go test ./services/gateway/...` 全绿。

### Review

> 待 arch_pm review 后填写。

---

## 一、RFC 标准符合性

### OAuth / OIDC / SAML（services/oauth + services/gateway）

| 标准 | 状态 | 结论 |
|------|------|------|
| RFC 6749 OAuth 2.0 核心 | ✅ | code/refresh grant、token 响应、§5.2 错误格式、redirect_uri 精确匹配均完整 |
| RFC 7636 PKCE | ⚠️ | S256 默认但不强制；接受 `plain`；**缺 code_verifier 43-128 长度校验** |
| RFC 6750 Bearer | ✅ | WWW-Authenticate 头、仅 Authorization header、不接受 query 参数 |
| RFC 7009 Revocation | ✅ | /oauth/revoke，无效 token 也返回 200 |
| RFC 7662 Introspection | ⚠️ | 端点+active 字段完整；**client 认证仅查存在性不验凭据**（server.go:2126-2151） |
| RFC 8628 Device Flow | ✅ | device_code/user_code/interval/slow_down/authorization_pending 完整 |
| OIDC Core | ✅ | ID Token 必需 claims、/userinfo、scope=openid 处理完整 |
| OIDC Discovery / RFC 8414 | ⚠️ | **3 处广告与实际不符**（见 P0-1/2/3） |
| RFC 7517 JWKS | ✅ | kid、自动+手动轮转（RotatingKeyProvider） |
| SAML 2.0 | ⚠️ | 验签/双 binding/SP-initiated 已实现；**ACS 缺 InResponseTo/NotOnOrAfter/Issuer 校验；SLO 不真注销；IdP 签名用临时密钥** |
| RFC 7591 DCR | ✅ | /oauth/register 已实现（但 discovery 未广告） |
| RP-Initiated Logout | ⚠️ | /oauth/logout 仅 backchannel，**不支持 id_token_hint/post_logout_redirect_uri** |
| client 认证 | ⚠️ | basic/post 完整；**private_key_jwt 未实现** |

### Auth / Identity / SCIM（services/auth + services/identity）

| 标准 | 状态 | 结论 |
|------|------|------|
| RFC 6238 TOTP | ⚠️ | 30s/6 位/SHA1 正确；**skew=0 无时钟容差；secret 明文存储**（domain/mfa.go:15 注释声称加密但未实现） |
| WebAuthn | ✅ | go-webauthn 真实验证、challenge 一次性、counter 克隆检测；challenge 内存存储（多实例需 Redis，P2） |
| RFC 6749 §10.4 reuse detection | ✅ | family 持久化 PG、reuse 撤销整族；存在 TOCTOU 竞态（见增强审视） |
| 密码策略 | ✅ | Argon2id(OWASP) + pepper + 历史 + 强度 + HIBP k-anonymity |
| 密码重置 | ✅ | 一次性 token、1h TTL、防枚举、重置后撤销会话 |
| 防爆破 | ✅ | IP 限流 + 滑窗 + 账户锁定 + 风控引擎 |
| SCIM RFC 7643 | ⚠️ | /ServiceProviderConfig、/ResourceTypes 有；**/Schemas 缺失** |
| SCIM RFC 7644 | ⚠️ | Users/Groups CRUD、scim+json、ListResponse、filter 全操作符、PATCH、ETag 均有；**ServiceProviderConfig 声称 etag:false 与实际实现矛盾** |
| auth gRPC RefreshToken | ⚠️ | 返回 Unimplemented（grpc_handler.go:57），refresh 唯一路径是 OAuth HTTP 端点 — 属设计决策但需文档明示 |

## 二、增强功能审视（RFC 之上）

| 增强 | RFC 兼容性 | 业务价值 | 问题 |
|------|-----------|---------|------|
| 动态 RBAC (Task-B) | ✅ 不破坏标准端点（仅 /api/ 前缀生效，isRBACExempt 兜底） | ✅ 有 | ① hasAdminScope 把 roles 当 scope 查，角色名 "administrator" 会意外提权（rbac_dynamic.go:263）② 非 /api/ 路径规则被静默忽略 ③ roles claim 裸名无命名空间，联邦 IdP 场景有冲突风险 |
| Refresh reuse detection (Task-E) | ✅ 符合 §10.4 | ✅ 高 | ① Used 判定与 revokeFamily 间 TOCTOU，并发/重试场景误伤合法客户端 ② StoreRefreshToken 错误被 `_ =` 吞掉（oauth_service.go:1353）③ 无 familyStore 时降级为 client-wide 撤销，误伤面更大 |
| 审计哈希链 (Task-G) | ✅ 纯内部 | ⚠️ 中（tamper-evidence 非真 WORM） | ① HMAC secret 无版本管理，轮换后旧链不可验 ② canonical 拼接用 `|` 分隔符，字段值含 `|` 可碰撞 ③ secret 为空时静默禁用不报警 |
| SAML 验签 (Task-D) | ⚠️ 基本兼容 | ✅ 高 | ① ACS 不验 InResponseTo（SP-initiated 可重放）② 不验 Issuer 是否匹配配置的 IdP entityID ③ 空 Audience 放行 ④ WantAssertionsSigned 配置未在 ACS 生效 ⑤ 缺 signature 必须是 assertion 直接子元素的 XSW 防护 |
| Console Settings (Task-F) | ✅ | ✅ | 无互操作问题 |

## 三、竞品迁移成本

| # | 摩擦点 | 严重度 | 说明 |
|---|--------|--------|------|
| M1 | Keycloak realm URL 结构不兼容 | P1 | `/realms/{r}/protocol/openid-connect/*` vs `/oauth/*`，keycloak-js 需换库+改 issuer；可加路径别名缓解 |
| M2 | **Auth0 audience 参数不解析** | P0 | token 端点所有标准 grant 均忽略 `audience`，aud 恒为 issuer，下游 API 验 aud 失败 |
| M3 | claim 结构差异 | P1 | Keycloak `realm_access.roles` / Auth0 namespaced claims vs GGID 裸 `roles`；需 claim 映射配置 |
| M4 | 密码哈希不可迁移 | P0(行业共性) | 所有竞品共同摩擦，需懒迁移/重置流程（文档已覆盖） |
| M5 | Auth0 Management API v2 无别名 | P1 | `/api/v2/*` → `/api/v1/*` 无映射 |
| M6 | DCR 可用但无 client 配置导入 | P1 | 不能导入 Keycloak realm export / Auth0 apps JSON |
| M7 | Clerk 前端组件 SDK 无替代 | P2 | 前端需重写 |
| M8 | `/dbconnections/signup`、`/oauth/revoke` Auth0 兼容已有 | ✅ | 无摩擦 |
| M9 | 迁移文档缺 claim 映射/端点对照表 | P2 | 文档补充即可 |

## 四、需求列表（按优先级）

### P0 — 阻塞标准互操作/安全
1. **Discovery `device_authorization_endpoint` 路径错误**：广告 `/api/v1/oauth/device_authorize`，实际路由 `/api/v1/oauth/device_authorization`（oauth_service.go:543 vs server.go:1538）→ 本轮修复
2. **Discovery 广告未实现的 implicit**：`response_types_supported=["code","token","id_token"]`，实际仅 code（authorize 只发 code，OAuth2.1 审计模块自己也在查 implicit）→ 本轮修复
3. **Discovery 缺已有端点**：`registration_endpoint`（/oauth/register 已存在）、`pushed_authorization_request_endpoint`（/oauth/par 已存在）→ 本轮修复
4. **SCIM `/scim/v2/Schemas` 端点缺失**（RFC 7643 §4，Okta/Azure AD 客户端依赖）→ 本轮修复
5. **Auth0 audience 参数不解析**（M2）→ 本轮修复（token 端点解析 audience 并写入 aud claim）
6. **SAML ACS 缺 InResponseTo/Issuer/NotOnOrAfter 校验**（重放风险）→ 移交 guardian_security（安全域），下轮跟进
7. **Introspection client 认证不验凭据**（server.go:2126-2151 仅查存在性）→ 本轮修复

### P1 — 迁移成本/安全加固
8. PKCE code_verifier 43-128 长度校验（RFC 7636 §4.1）→ 本轮修复
9. TOTP skew=0 → ValidateCustom(skew=1)（mfa_service.go:101）→ 本轮修复
10. SCIM ServiceProviderConfig `etag.supported` false→true（handler.go:669 vs 501-520 已实现）→ 本轮修复
11. TOTP secret 明文存储 → AES 加密落盘（需迁移方案）→ 后续轮次
12. RP-Initiated Logout（id_token_hint + post_logout_redirect_uri）→ 后续轮次
13. Keycloak 路径别名 `/realms/{r}/protocol/openid-connect/*`（M1）→ 后续轮次
14. Per-client claim 映射配置（M3）→ 后续轮次
15. Reuse detection TOCTOU + StoreRefreshToken 错误吞掉 → 后续轮次（需设计 CAS/宽限窗口）
16. SAML IdP 签名临时密钥 → 持久化密钥配置（server.go:2435）→ 移交 guardian_security
17. Auth0 Management API `/api/v2/*` 别名（M5）→ 后续轮次
18. 动态 RBAC：roles 当 scope 提权风险 + 非 /api/ 规则静默忽略 → 移交 guardian_security/arch_pm

### P2 — 优化/文档
19. private_key_jwt client 认证（FAPI 场景）
20. WebAuthn challenge 内存存储 → Redis（多实例）
21. 审计哈希链：canonicalization 碰撞、secret 版本管理、禁用不报警
22. roles claim 命名空间化（`https://ggid.dev/roles`）或文档化
23. 迁移文档补 claim 映射指南 + 端点对照表（M9）
24. auth gRPC RefreshToken Unimplemented 需文档明示 refresh 唯一路径
25. Client 配置导入工具（Keycloak/Auth0 JSON → DCR）（M6）

## 五、本轮实施记录

> 审视人：god_fullstack。隔离 worktree `iam-review-round1`。全部 CI 门控通过（`go build ./...` + `go test -timeout 10m ./...` 全绿，lint 无新增）。

### 已完成（6 项 P0/P1 快赢，6 commits）

| # | Commit | 改动 | 测试 |
|---|--------|------|------|
| 1 | `28a1c867c` | OIDC discovery 对齐实现：移除 implicit `response_types`、移除未实现的 `check_session_iframe`、补 `registration_endpoint`/`pushed_authorization_request_endpoint`/`frontchannel_logout_supported` | 更新 2 个 discovery 测试断言 |
| 2 | `168c53e89` | SCIM `/scim/v2/Schemas` + `/scim/v2/Schemas/{urn}`（RFC 7643 §4）；ServiceProviderConfig `etag.supported` false→true | 新增 `schemas_test.go`（3 个测试） |
| 3 | `7af1bd175` | token 端点 4 个 grant 解析 `audience` 参数写入 `aud` claim（Auth0 迁移兼容，空值保持 client_id 默认） | 新增 `audience_test.go`（2 个测试） |
| 4 | `e0a7ba679` | introspection 真实验 client 凭据（新增 `OAuthService.AuthenticateClient`，Basic/form 验注册表，Bearer 验 active） | 重写 `TestIntrospectRequestAuthenticated`（8 个子用例） |
| 5 | `52a70ddda` | PKCE `code_verifier` 43-128 长度 + unreserved 字符集校验（RFC 7636 §4.1） | 新增 `pkce_test.go`（11 个子用例）+ 修正 2 个既有测试 |
| 6 | `f759fca12` | TOTP skew 0→1（`ValidateCustom`，±30s 时钟容差，RFC 6238 §5.2） | 既有 auth 测试全绿 |

### 团队协作移交（guardian_security 已完成，3 commits 入 main）

| # | Commit | 改动 |
|---|--------|------|
| G1 | `e545930cd` | SAML ACS：InResponseTo 重放防护 + Issuer 匹配 + 空 Audience 拒绝 + XSW signature 放置校验 |
| G2 | （同批次） | SAML IdP 签名密钥持久化（sys_config，非临时密钥） |
| G3 | `6b97c7a54` | 动态 RBAC：hasAdminScope 只认 platform:admin/tenant:admin scope，裸 roles 名不再提权 |

---

## 六、独立 Review 结论

**Review 人**：arch_pm（spec 对齐 + 部署验证）、frontend_qa（编译测试复验委托中）

**arch_pm review 结论：通过** ✅（2026-07-22 00:35 CST）

审查点逐条确认：
1. **Discovery 对齐**：移除 implicit response_types 合理（OAuth 2.1 方向），device_authorization_endpoint 路径保留正确
2. **Audience fallback**：空值 fallback 到 client_id，不影响 Console password grant flow
3. **Introspection**：Bearer token 方式（Method 3）保留，resource server 用 own token introspect 不受影响
4. **SCIM /Schemas**：结构正确，但需 Okta/Entra ID 实际对接做端到端验证（后续跟进）

**部署验证**：3 个服务已重建部署
- `oauth:v2` — password grant 200 ✅，roles API (admin) 200 ✅，discovery response_types=["code"] ✅
- `auth:latest` — TOTP skew 改动随部署生效
- `gateway:latest` — 含 guardian_security RBAC 提权修复 + ab762f527 回归修复

**guardian_security 安全修复（3 项，已随 main 部署）**：
- SAML ACS InResponseTo/Issuer/Audience/XSW 防护 ✅（3 个安全测试全绿）
- IdP 签名密钥持久化 ✅（sys_config，非临时密钥）
- 动态 RBAC hasAdminScope 只认 platform:admin/tenant:admin ✅（9 个攻击向量断言拒绝）

**遗留待跟进项**：
- SCIM /Schemas 需 Okta/Entra ID 实际对接验证
- P1/P2 后续轮次迭代（TOTP secret 加密、RP-Initiated Logout、Keycloak 路径别名、claim 映射、reuse detection TOCTOU 等）

> **本轮审视状态：完成。** 下一轮 cron-1 触发时将聚焦增量问题。
