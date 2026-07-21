# GGID IAM 产品深度审查报告

日期：2026-07-20
范围：全代码库（10 个服务，~215k 行 Go 代码）+ K8s 部署 + SDK 生态 + 测试体系

---

## 1. 架构与可扩展性

### 优点
- 微服务拆分合理：auth/identity/oauth/policy/audit/org/gateway 职责清晰
- Gateway 统一入口（JWT 验证、RLS 上下文、RBAC、CAE、限速）已形成完整中间件链
- PostgreSQL RLS + `app.tenant_id` 会话变量实现真正的行级多租户隔离

### 欠缺
1. **水平扩展隐患**：~30+ 个服务内功能仍使用进程内 map（统计 198 个文件含 map，大部分是缓存但部分是状态）。Pod 重启后数据丢失，多副本部署时状态不一致。P0 持久化修复了 OTP/incident/webhook 等关键项，但 dashboard widgets、alert rules、composite rules、SCIM group store 等仍是内存态。
2. **单点依赖**：所有服务直连同一 PostgreSQL。无读写分离、无分片策略。`audit_events` 按月分区是好实践，但其他大表（sessions、audit_events 索引策略）在百万级用户下缺乏验证。
3. **NATS 异步链路脆弱**：audit 事件 `PublishAsync` 历史上丢消息（已改同步 Publish，但回调仍 fire-and-forget）。无 DLQ、无重放机制。
4. **服务间调用无熔断的统一实现**：gateway 有 circuit breaker，但 auth→identity、oauth→identity 的内部 HTTP 调用无超时/重试/熔断标准。

---

## 2. 安全态势

### 本季度已修复的高危项
- Gateway RBAC 完全缺失（普通用户可读全部用户/审计）→ 已加 RequireAdminScope
- JWT scope 用 UUID 而非 role key → 已修复
- CAE middleware 顺序错误（JTI 吊销不生效）→ 已修复
- RLS 上下文未设置导致跨租户创建失败 → 已修复
- Consent middleware 阻止 platform:admin → 已修复 break-glass
- SAML 断言无签名校验/audience 检查 → 已加
- MFA disable 无重新认证（KB-269）→ 已加密码验证
- 设备删除跨租户（KB-278）→ 已加 tenant 过滤

### 仍欠缺
1. **SAML 验签是表面级**：仅检查 `<ds:Signature` 元素存在，不做 RSA 验签。生产环境必须接入 crewjam/saml 或自实现 XML-DSig 验证。
2. **密钥管理**：JWT RSA 密钥从 K8s Secret 挂载 + 90 天轮换（KB-329 已实现）。但 Argon2 参数硬编码、PASSWORD_PEPPER 未在 K8s 配置、加密密钥（BIOMETRIC/CRED_VAULT AES key）用 derived dev key —— 生产部署清单缺这些强制检查。
3. **RBAC 是硬编码前缀匹配**：`RequireAdminScope` 用固定 adminPrefixes + hasAdminScope。PM 提出的动态 RBAC（role_permissions + Redis cache）未落地 —— 当前加新角色需要改代码。
4. **审计不可篡改性**：audit_events 是普通表，无哈希链/WORM 存储。合规场景（SOX/HIPAA）需要 tamper-evident ledger。
5. **限流维度**：登录限流已实现（IP+user），但 API 级 DDoS 防护（per-tenant quota、per-endpoint cost）粗糙。
6. **Session 固定攻击**：登录成功后未轮换 session ID；JWT jti 有 blocklist 但 refresh token 家族检测（reuse detection）未实现。

---

## 3. 功能完整性（对标 Keycloak/Auth0/Okta）

### 已有 ✅
- OAuth 2.0 + OIDC（授权码、client credentials、refresh、introspection、RFC 7591/7592 动态注册）
- SAML 2.0 SP/IdP 基础流程
- MFA: TOTP、WebAuthn/Passkey、RADIUS、YubiKey（含 test mode）
- SCIM 2.0 基础（用户/组、bulk）
- LDAP provider + 同步配置
- CAE（Continuous Access Evaluation，jti blocklist + session revocation）
- ITDR（identity threat detection，NATS 消费 + engine）
- 条件访问策略（conditional access）
- SoD（职责分离）、privilege creep、standing access、JIT 访问
- Webhook 订阅 + 签名
- 多语言 SDK（12 种，Java/Go 最完整）
- 设备绑定、impossible travel、session hijack 检测（KB-275/276 刚改为真实查询）

### 缺失或半成品 ❌
1. **用户自助门户不完整**：Profile 页刚补真实数据；账户删除端点刚加（KB-262）但无 grace period/恢复机制；邮件变更流程有代码但未验证。
2. **Social login**：social.Registry 存在但 Google/GitHub/企业微信连接器未配置验证过。
3. **组织层级**：organizations 表有 ltree path，但 UI 无树形管理、无跨 org 用户移动、无 org 级策略继承。
4. **授权码 PKCE 强制**：RequirePKCE 字段存在但默认 false，公开客户端不强制 PKCE。
5. **Token 生命周期管理**：无 token 家族/滚动刷新令牌轮换策略的用户可见界面；无批量吊销 UI（API 有）。
6. **租户级品牌化**：tenant_branding 表刚建，Console 无品牌化配置 UI（logo/颜色/登录页文案）。
7. **Identity governance**：access review campaign 有 repo 但无完整审批工作流 UI；joiner-mover-leaver 流程只有 joiner-dashboard。
8. **合规报表**：SOC2/GDPR/HIPAA 映射表存在（compliance_mapping），但无一键导出审计报告功能。
9. **B2B 功能**：无 organization invitation flow、无 delegated admin、无 SSO enforcement per-org。
10. **密码策略 UI**：多处 handler 存在（强度分布、历史、breach 检查）但 Console 配置页是 mock/404（R12 发现的 settings 404）。

---

## 4. 代码质量与可维护性

### 量化数据
| 服务 | 行数 | 测试覆盖 |
|------|------|---------|
| gateway | 47,047 | router 62.2% |
| identity | 38,884 | 低 |
| auth | 37,639 | service **16.9%** ⚠ |
| oauth | 31,782 | 中 |
| audit | 26,478 | 中 |
| policy | 20,996 | 中 |

### 问题
1. **auth service 覆盖率 16.9%** 是最大风险——认证核心逻辑（MFA、session、token、password）大部分无单元测试，回归靠 Playwright E2E 兜底。
2. **handler 文件巨型化**：auth/http.go 超 3000 行、router.go 超 1300 行。建议按 domain 拆分。
3. **重复的 tenant 解析逻辑**：injectTenant、tenantIDFromHeader、tenantIDFromContext、X-Tenant-ID fallback 在十几个文件中重复实现，本轮已因优先级反转出过 P0 bug。应统一为一个 TenantResolver 并写进包级文档。
4. **nil 上下文传递**：本轮发现 `QueryRow(nil, ...)`、`RevokeUser(nil, ...)` 两处 panic 隐患，说明 code review 缺少静态检查（go vet 不查这个，需要自定义 linter 或 review checklist）。
5. **错误吞噬**：`_, _ = pool.Exec(...)` 模式在 20+ 处出现，KB-277 就是其中一例。应全局排查改为记录日志。
6. **golangci-lint 本地/CI 版本不一致**（v2 vs v1.64.8）导致本地验证与 CI 脱节。

---

## 5. 多租户成熟度

- **数据隔离**：RLS + relforcerowsecurity 已生效（本轮 P0 修复后跨租户创建验证通过）
- **租户生命周期**：创建/删除/查询已 DB 化（gateway 直连）✅
- **跨租户管理**：platform:admin break-glass + impersonation consent 模型设计完成，表已建，UI 未做
- **欠缺**：租户级配额执行（max_users 字段存在但无强制）、租户级限速配置（TenantBucketLimiter 是全局默认）、租户数据导出/删除（GDPR 租户级）、子域名路由仅部分工作

---

## 6. 开发者体验

### 优点
- 12 语言 SDK，Java/Go 功能最全
- OpenAPI spec + Swagger UI 内置于 gateway
- Demo 应用覆盖 OAuth/SAML/ERP

### 欠缺
1. **SDK 功能差距大**：Java 24 类 vs PHP/Ruby/C# 仅基础登录。WebAuthn 仅 4 语言补齐。无版本化/发布流程（无 npm/pypi/maven 发布）。
2. **文档碎片化**：docs/ 下多个报告但无面向开发者的 getting-started 指南；API 变更无 changelog。
3. **本地开发环境**：all-in-one Dockerfile 存在但文档少；新成员上手需要读 Makefile + 猜端口。
4. **CLI 工具**：ggid-cli 只有 3.7k 行，功能远不如 SDK。

---

## 7. 测试体系

| 层 | 现状 | 缺口 |
|----|------|------|
| 单元 | gateway 62%, auth 17% | auth/identity 核心服务急需补 |
| 集成 | services 内 repository 测试较好 | 跨服务集成测试少 |
| E2E | Playwright 65 passed | 刚修复稳定性；租户隔离/ABAC 数据维度未测 |
| 安全 | security-test.sh 存在 | 未接入 CI 定期执行 |
| 性能 | k6 baseline 存在 | 无定期回归 |

---

## 8. 可观测性与运维

- Prometheus metrics 端点各服务有 ✅
- Grafana dashboards 3 个 JSON（KB-331）已建但告警规则未配置
- 结构化日志 slog 部分采用，仍有大量 log.Printf
- 分布式追踪：OTel middleware 存在但未验证 trace 串联
- 部署：K8s 手工 rollout + docker build/push，无 Helm/kustomize 版本化发布流程，无 rollback runbook

---

## 9. 优先级建议

### P0（产品可用性阻塞）
1. auth service 单元测试覆盖率提升到 50%+（防回归）
2. 完成动态 RBAC（role_permissions + Redis cache）替代硬编码前缀
3. 剩余内存态功能持久化（alert rules、composite rules、dashboard widgets、SCIM store）

### P1（企业销售阻塞）
4. SAML 真实 XML-DSig 验签
5. Refresh token 家族轮换 + reuse detection
6. 租户品牌化 UI + Console settings 页真实化（当前多个 404）
7. 审计 tamper-evidence（哈希链）
8. Org 树形管理 UI + B2B invitation flow

### P2（规模化）
9. 审计报表一键导出（SOC2/GDPR 证据包）
10. SDK 发布流程（npm/pypi/maven）+ 功能对齐
11. Helm chart + 发布/回滚 runbook
12. 分布式追踪验证 + 告警规则

---

## 总结

GGID 在 3 周内完成了从"空壳"到"核心可用"的跨越：认证、MFA、多租户 RLS、RBAC、审计、12 语言 SDK 全部打通。当前最大的三个风险是：

1. **auth 服务 17% 测试覆盖率**——认证核心无回归保护，每次改动都是赌博
2. **硬编码权限模型**——每个新角色/新端点都要改代码重新部署，无法产品化交付
3. **状态持久化残留**——约 10% 功能仍是内存态，多副本部署时会出诡异 bug

完成 P0 三项后可称 v1.0；完成 P1 可对企业客户销售。
