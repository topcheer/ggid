# GGID IAM 安全体系全面审计报告

日期：2026-07-23
审计员：guardian_security
范围：全部 10 个服务 + 网关 + SDK + 部署配置

---

## 审计方法论

从 7 个维度系统性审查，对标 OWASP Top 10、OAuth 2.0 Security BCP (RFC 9700)、NIST 800-63B：
1. 认证与凭据安全
2. 授权与访问控制
3. 会话与令牌安全
4. 传输与存储加密
5. 输入验证与注入防护
6. 审计与可观测性
7. 架构与运维安全

每项标注严重度（P0 阻塞 / P1 高 / P2 中 / P3 低）和当前状态。

---

## 一、认证与凭据安全

### 已落实 ✅
- Argon2id 密码哈希（OWASP 推荐参数）+ pepper + 历史检查 + HIBP k-anonymity breach 检测
- 客户端密钥用 Argon2id 存储（`verifyClientSecret` → `crypto.VerifyPassword`，constant-time）
- JWT `alg=none` 拒绝（rfc7523.go:53）；HS 系列算法拒绝用于 client_assertion
- MFA：TOTP（RFC 6238）、WebAuthn/Passkey、RADIUS、YubiKey
- 密码策略可配置（长度/复杂度/历史/过期）

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| A1 | **LDAP InsecureSkipVerify=true** | P1 | `ldap_sync_service.go:123` StartTLS 跳过证书验证，MITM 风险。生产必须验证 IdP 证书。 |
| A2 | **GGID_INTERNAL_SECRET 有 dev fallback** | P1 | `scim_token_middleware.go:83` fallback `"dev-internal-secret"`，`secret_broker.go:247` 同款。生产部署若漏配 env，内部认证降级为已知弱密钥。 |
| A3 | **PASSWORD_PEPPER 生产未强制** | P1 | auth/cmd/main.go:105 仅 `if pepper != ""`。未配置时不报错不告警，密码哈希少一层保护。 |
| A4 | **TOTP secret 明文存储** | P2 | domain/mfa.go 注释声称加密但未实现（review 报告已标出）。 |
| A5 | **WebAuthn challenge 内存存储** | P2 | 多实例部署时 challenge 不可共享，影响 Passkey 注册/认证可用性。需迁移 Redis。 |
| A6 | **JWT key 生成仅 2048-bit** | P3 | RSA 2048 目前安全但 NIST 建议新系统用 3072-bit。key_rotation.go:103。 |

---

## 二、授权与访问控制

### 已落实 ✅
- Gateway 三层 RBAC（动态 role_route_permissions + 静态 RequireAdminScope + checkRouteScope）
- RLS 行级租户隔离（`SET app.tenant_id` + FORCE RLS）
- 自服务白名单精确匹配（SelfServicePaths map）
- 平台/租户 admin scope 分离（`platform:admin` / `tenant:admin`）
- SAML ACS 完整校验（签名验证 + InResponseTo + Issuer + Audience + XSW 防护）
- GDPR 删除需密码确认 + WORM 审计

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| B1 | **`platform:admin` scope 签发侧无租户过滤** | P1 | auth `http_identity_client.go` 按角色 key 生成 scope，租户自建 role key=`platform:admin` 可伪造平台管理员 scope。三层 RBAC 已无法防御（因为 scope 本身被伪造）。根治必须在签发侧验证角色归属。 |
| B2 | **CORS `Access-Control-Allow-Origin: *` 作为 fallback** | P2 | audit/http.go:715 和 per_tenant_cors.go:90,93 在未匹配到允许源时回退 `*`，允许任意跨域。生产应 fail-closed（拒绝而非通配）。 |
| B3 | **OAuth token scope 未与 client 注册 scope 交叉验证** | P2 | authorize 端点接受请求 scope 但未严格校验是否在 client 注册的 allowed_scopes 内（oauth_service.go IssueToken 路径）。可导致 scope 扩张。 |
| B4 | **consent cascade mock 路径** | P3 | `cascade.go:251` 返回 `tok_mock_1` 等硬编码值，生产 pool 为 nil 时可能走到。 |

---

## 三、会话与令牌安全

### 已落实 ✅
- Access token TTL 15 分钟（符合 OAuth 2.1 BCP）
- Refresh token family + reuse detection（RFC 6749 §10.4）
- CAE（JTI blocklist + session revocation）
- Cookie 安全属性（HttpOnly + SameSite=Strict + Secure）
- Session 表存储 token_hash（非明文）

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| C1 | **revokedTokens 和 stateStore 用 sync.Map（内存态）** | P1 | oauth_service.go:1223-1224。多实例部署时一个 pod 吊销的 token 在其他 pod 仍有效。必须迁移 Redis。 |
| C2 | **DPoP token cache 内存态** | P2 | dpop_token_bind.go:14 `dpopCache sync.Map`，同 C1 问题。 |
| C3 | **Session cleanup 依赖定时扫描** | P3 | auth/cmd/main.go:460 周期清理过期 session，期间过期 session 仍在 DB 中可被滥用（TTL 内）。影响低但非最佳。 |
| C4 | **Refresh token TOCTOU 竞态** | P2 | Used 判定与 revokeFamily 之间存在时间窗口（review 已标出），并发请求可能误伤合法客户端。需 CAS 或 SELECT FOR UPDATE。 |

---

## 四、传输与存储加密

### 已落实 ✅
- gRPC TLS（MinVersion 1.2）identity/audit/org 服务
- SIEM forwarder TLS（MinVersion 1.2）
- Cookie Secure flag（生产 HTTPS）

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| D1 | **LDAP InsecureSkipVerify** | P1 | 同 A1。StartTLS 后不验证证书 = 加密但无身份验证。 |
| D2 | **审计 HMAC secret 无版本管理** | P2 | hash_chain.go 全局单一 secret，轮换后旧链不可验证。需 secret ID + 多版本支持。 |
| D3 | **审计 HMAC canonicalization 碰撞风险** | P2 | canonical 拼接用 `|` 分隔符，字段值含 `|` 可碰撞。需固定长度前缀或 protobuf。 |
| D4 | **JWT key 轮换 grace period 过长** | P3 | key_rotation.go gracePeriod 内旧 key 仍签发，泄漏后窗口更大。应缩短至 <5 分钟。 |

---

## 五、输入验证与注入防护

### 已落实 ✅
- 参数化查询为主（$1, $2 占位符）
- RLS + `SET app.tenant_id` 隔离
- Gateway 请求体大小限制
- Security headers（X-Frame-Options, CSP, HSTS, X-Content-Type-Options）

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| E1 | **fmt.Sprintf 拼接 SQL 列名** | P2 | identity/pg_repo.go 多处 `fmt.Sprintf("SELECT %s FROM ...", userColumns)`。虽然列名是常量非用户输入（无直接注入），但模式不安全——后续若有人改列名来源则引入注入。应改为固定字符串或白名单验证。 |
| E2 | **map_repo.go 表名 fmt.Sprintf** | P2 | `fmt.Sprintf("SELECT ... FROM %s", table)` — table 参数若来自用户可控路径则为注入。当前调用方传硬编码表名，但缺乏防御性校验。 |
| E3 | **20 处错误吞噬 `_, _ = pool.Exec`** | P2 | 审计和安全操作的 DB 错误被静默忽略，可能导致安全事件丢失。应至少 slog.Warn。 |
| E4 | **OAuth state 参数仅内存校验** | P3 | stateStore sync.Map，多实例不一致可能导致 CSRF 保护失效。 |

---

## 六、审计与可观测性

### 已落实 ✅
- 哈希链 tamper-evidence（HMAC-SHA256 + WORM 触发器 + 自动告警）
- ITDR 威胁检测引擎（NATS consumer + 规则匹配）
- 审计事件包含 actor/IP/UA/device/request_id
- 结构化日志（slog 部分采用）
- Prometheus metrics 端点

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| F1 | **HMAC secret 未配置时静默禁用** | P1 | hash_chain.go `IsHashChainEnabled()` 返回 false 时所有事件不建链，无告警。生产必须 fail-closed 或至少 loud warning。 |
| F2 | **审计事件大量 log.Printf** | P3 | 部分服务仍用 log.Printf 而非 slog，不利于日志聚合和安全分析。 |
| F3 | **告警规则未配置** | P2 | Grafana dashboards 已建但无告警规则。tamper detection 写了 audit_incidents 但无主动通知渠道（email/Slack/webhook 未对接）。 |

---

## 七、架构与运维安全

### 已落实 ✅
- 微服务隔离（独立 deployment + DB 连接池）
- K8s Secrets 挂载（JWT key、DB 密码）
- 网络：K8s ClusterIP + Ingress TLS
- 镜像最小化（Alpine + ca-certificates）

### 薄弱点

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| G1 | **无服务间 mTLS** | P2 | 服务间 gRPC 调用虽有 TLS 但未做双向证书验证。K8s 网络内任何 pod 可冒充。Istio/Linkerd 或自签 CA 可解。 |
| G2 | **DB 凭据共享** | P2 | 所有服务用同一 `ggid:ggid-k3s` 连同一个 PG 实例。无读写分离、无 per-service 权限隔离。 |
| G3 | **无密钥管理服务集成** | P2 | JWT key、HMAC secret、PASSWORD_PEPPER 全从 env/文件读取。无 Vault/KMS 集成，轮换需手动重启。 |
| G4 | **无 Helm/版本化发布** | P3 | 部署靠手工 docker build + kubectl rollout。无回滚 runbook，安全补丁部署慢。 |
| G5 | **dependency 供应链安全缺失** | P2 | 无 `govulncheck` CI、无 SBOM、无 dependabot/renovate。依赖漏洞不可见。 |

---

## 优先级排序与修复建议

### 立即修复（P0/P1，安全阻塞）

| 优先级 | 项 | 修复方案 | 预估工作量 |
|--------|-----|---------|-----------|
| 1 | **B1** scope 签发侧租户过滤 | `http_identity_client.go` 只为平台租户的 `platform:admin` 角色生成 scope | 2h |
| 2 | **A2** 内部密钥 dev fallback 移除 | 删除 `"dev-internal-secret"` fallback，空值时 panic | 30min |
| 3 | **A1/D1** LDAP 证书验证 | InsecureSkipVerify 改为读取配置的 CA 证书 | 1h |
| 4 | **C1** revokedTokens 迁移 Redis | sync.Map → Redis SET + TTL | 4h |
| 5 | **F1** HMAC secret 缺失告警 | boot 时检查，缺失则 panic 或强制降级模式 | 30min |
| 6 | **A3** PASSWORD_PEPPER 强制检查 | boot 时检查，缺失则 panic 或 loud warning | 15min |
| 7 | **B2** CORS fail-closed | 未匹配源时返回 403 而非 `*` | 1h |

### 近期修复（P2，企业交付前）

| 优先级 | 项 | 修复方案 |
|--------|-----|---------|
| 8 | A4 TOTP secret 加密 | AES-GCM 加密落盘，密钥从 KMS/env |
| 9 | C4 Refresh TOCTOU | SELECT FOR UPDATE 或 optimistic lock |
| 10 | E3 错误吞噬 | 全局排查 `_, _ =` → slog.Warn |
| 11 | F3 告警通知 | tamper incident → email/Slack webhook |
| 12 | B3 OAuth scope 校验 | IssueToken 校验 requested scope ⊆ client.allowed_scopes |
| 13 | D2/D3 审计 secret 版本化 + canonical | secret ID 列 + protobuf canonical |
| 14 | G5 依赖安全 | `govulncheck` CI + SBOM 生成 |

### 后续迭代（P3）

15. A6 JWT key 升级 3072-bit
16. A5/C2 WebAuthn challenge + DPoP 迁移 Redis
17. E1/E2 SQL 列名/表名白名单校验
18. G1 服务间 mTLS
19. G3 KMS/Vault 集成
20. G4 Helm chart + 回滚 runbook

---

## 总结

GGID 经过近期密集安全加固，核心认证/授权/审计链路已达到企业级安全基线。**但存在一个系统性薄弱点：多实例状态管理**。revokedTokens、stateStore、dpopCache、WebAuthn challenge 四处用 sync.Map 存储安全状态，在 K8s 多副本环境下会失效——这是当前最大的架构级安全风险。

第二级风险是**密钥管理薄弱**：dev fallback、未强制配置、无轮换自动化、无 KMS 集成。生产部署清单缺少这些强制检查项。

第三级风险是**签发侧信任链不完整**：`platform:admin` scope 的生成不验证角色租户归属，这是 RBAC 层面所有修复的上游根因。

建议按上表优先级逐项修复，前 7 项可在 1 天内完成，使系统达到生产安全就绪。

---

## 巡航日志

### 巡航 #1 | 维度 2: 授权与访问控制 | 2026-07-23

**S1-S4 部署验证**：
- S1 scope filter: deployed (commit 787270449, binary contains platform-reserved filter)
- S2 dev secrets removed: scim_token_middleware + secret_broker + sdjwt_handler (commit 4c326e28a)
- S3 HMAC secret alert: deployed (commit 787270449)
- S4 LDAP TLS: deployed (commit 787270449)
- encryption_key.go dev fallback: FIXED (commit 0200102fa)

**RBAC 单元测试**: ✅ 全绿 (router 7/7 + middleware 7/7)
**生产匿名访问**: ✅ /users 401, /oauth/clients 401, /system/config 401

**新发现**：
| # | 发现 | 严重度 | 位置 | 状态 |
|---|------|--------|------|------|
| P2-8 | jwt_claims.go:139 仍用裸 "admin"/"superadmin"/"*" 匹配 admin scope | P2 | gateway/middleware/jwt_claims.go | OPEN |
| P2-9 | router.go:724 "administrator"/"platform administrator" 裸名匹配 hasPlatform | P2 | gateway/router/router.go | OPEN |
| P2-10 | router.go:986 "admin"/"ggid:admin" 裸名检查 | P2 | gateway/router/router.go | OPEN |

**内存态安全存储**（历史项，仍 OPEN）：
- revokedTokens sync.Map: ⚠️ Redis 优先+内存 fallback（C1, P1）
- stateStore sync.Map: ⚠️ 同上（C1, P1）
- dpopCache sync.Map: ❌ 纯内存无 Redis（C2, P2）

**结论**: S1 根因修复已部署，三层 RBAC 防护完整。3 个裸名残留为 P2（提权面收窄但未完全消除，需下轮统一为 scope-only）。

### 修复状态汇总（2026-07-23 final）

| 项 | 严重度 | 状态 | commit |
|---|--------|------|--------|
| S1 scope 签发侧租户过滤 | P1 | ✅ FIXED+DEPLOYED+VERIFIED | 787270449 |
| S2 dev fallback 密钥移除 | P1 | ✅ FIXED+DEPLOYED | 787270449+4c326e28a |
| S3 HMAC secret 缺失告警 | P1 | ✅ FIXED+DEPLOYED | 787270449 |
| S4 LDAP 证书验证 | P1 | ✅ FIXED+DEPLOYED | 787270449 |
| encryption_key.go dev fallback | P1 | ✅ FIXED+DEPLOYED | 0200102fa |
| P2-1 TOTP secret 加密落盘 | P2 | ✅ FIXED+DEPLOYED | f1920ce55 |
| P2-6 HMAC secret 版本管理 | P2 | ✅ FIXED+DEPLOYED | 63ed9054f |
| P2-7 canonicalization 碰撞修复 | P2 | ✅ FIXED+DEPLOYED+REPAIRED | 63ed9054f |
| P2-8 jwt_claims.go 裸名匹配 | P2 | ✅ FIXED+DEPLOYED | 7bc8c4572 |
| P2-9 router.go:724 裸名匹配 | P2 | ✅ FIXED+DEPLOYED | 7bc8c4572 |
| P2-10 router.go:986 裸名匹配 | P2 | ✅ FIXED+DEPLOYED | 7bc8c4572 |
| C1 revokedTokens sync.Map | P1 | ❌ OPEN (Redis-first, mem fallback) | — |
| C2 dpopCache sync.Map | P2 | ❌ OPEN (pure memory) | — |
| C4 stateStore sync.Map | P2 | ❌ OPEN (Redis-first, mem fallback) | — |

### 巡航 #2 | 维度 3: 会话与令牌安全 | 2026-07-23

**历史项复查**：
| # | 发现 | 之前状态 | 当前状态 |
|---|------|---------|---------|
| C1 | revokedTokens sync.Map | ❌ OPEN | ✅ FIXED (DB-backed, commit 0b2cd2a48) |
| C2 | dpopCache sync.Map | ❌ OPEN | ❌ 仍 OPEN (纯内存, P2) |
| C4 | stateStore sync.Map | ❌ OPEN | ⚠️ Redis优先+内存fallback (P2) |
| C4b | Refresh TOCTOU (SELECT FOR UPDATE) | 未追踪 | ❌ OPEN (0处FOR UPDATE, P2) |

**安全实践确认 ✅**：
- Cookie: HttpOnly + SameSite (Lax/Strict) + Secure ✅
- PKCE: 公开客户端强制 + per-client RequirePKCE ✅
- Token comparison: subtle.ConstantTimeCompare + hmac.Equal ✅
- Access token TTL: 15min (符合 OAuth 2.1 BCP) ✅
- Security headers live: HSTS + X-Frame DENY + X-Content-Type nosniff ✅

**新发现**：
| # | 发现 | 严重度 | 详情 |
|---|------|--------|------|
| P3-1 | Session fixation: 无 session ID 轮换 | P3 | 登录成功后未生成新 session ID（auth_service.go 无 RotateSession 调用）。JWT jti 有 blocklist 但 session 表记录固定。影响低（JWT-based，非 cookie-session）。 |
| P3-2 | CORS fallback 仍可能返回 * | P3 | per_tenant_cors.go 有 fallback 配置，若未配置 allowed origins 可能通配。已有 security_headers.go HardenCookie 但 CORS 层未 fail-closed。 |

**近期 commit 安全审查**：
- `8e95c7758` social login publicPath 去重 — 无安全问题
- `83809081e` CORS fail-closed + PEPPER warning — 正面改进
- `c24a19645` JWT permissions 去重 — 无安全问题

**结论**: C1 已修复（DB-backed revokedTokens）。剩余 C2/C4 为 P2（Redis-first 已有缓解）。无新 P0/P1 发现。

### 巡航 #3 | 维度 4: 传输与存储加密 | 2026-07-23

**历史项复查**：
| # | 发现 | 状态 |
|---|------|------|
| D1 | LDAP InsecureSkipVerify | ✅ FIXED (comment-only, actual code loads CA cert) |
| D2 | HMAC secret 版本管理 | ✅ FIXED (commit 63ed9054f, 12 refs) |
| D3 | canonicalization 碰撞 | ✅ FIXED (length-prefix %04x) |
| A4 | TOTP secret 加密 | ✅ FIXED (3 refs in mfa_pg_repo) |
| A1 | TLS MinVersion | ✅ 7处 TLS 1.2+ |

**错误吞噬增长**: 20→25处 `_, _ =` (增加5处, P2, 建议排查新增来源)
**近期 commit**: R1-02 social login migration + ERP stability — 无安全回归
**新发现**: 无 P0/P1。错误吞噬增加 5 处标记为 P3 跟踪。
