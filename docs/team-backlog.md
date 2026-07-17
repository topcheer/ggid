# GGID Team Backlog

*Last updated: 2026-07-17 (Round 45: ZTNA Broker Integration research complete — 7 new backlog items)*

## Current Stats

- **Docs**: 757 markdown files
- **Console pages**: 629 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 27 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 45/45 packages PASS, 0 FAIL
- **Real productization gaps**: 0

## Gap Closure Priority Queue

### P1 — Real productization gaps (from platform-completeness-report.md)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 1 | OAuth 2.1 compliance audit is a stub | backend | services/oauth/internal/server/oauth21_audit_handler.go | [NEW] | Implement dynamic analyzer reading real client configs |
| 2 | FAPI 2.0 profile not exposed | backend | services/oauth | [NEW] | Add fapi_2_0 client flag and enforce PAR+PKCE+DPoP+response_type=code |
| 3 | FedCM not implemented | backend | services/oauth | [NEW] | Add FedCM config/accounts/login endpoints (future, low priority) |

### P2 — Research-driven competitive/compliance gaps

| # | Feature | Owner | Driver | Notes |
|---|---------|-------|--------|-------|
| 4 | OAuth 2.1 enforcement mode | backend | RFC 9700 / OAuth 2.1 | [DONE] dfcb8a7f |
| 5 | FAPI 2.0 profile | backend | OpenID FAPI | [DONE] ccae234f |
| 6 | FedCM support | backend | Chrome/Edge default | [ACCEPTABLE] future consumer identity |
| 7 | PIPL/NIS2/CRA compliance research | docs/arch | PIPL amended CSL / NIS2 / CRA | [DONE] docs/research/nis2-cra-pipl-compliance.md |
| 8 | Passkey health dashboard | frontend | passkey adoption | Console page showing passkey enrollment status, recovery risk |
| 9 | PQC migration (ML-DSA / ML-KEM) | arch | NIST PQC | Hybrid TLS + JWT signing in pkg/crypto |
| 10 | NIS2 / CRA compliance dashboard | frontend | EU regulation | Security incident reporting, SBOM, vulnerability tracking |
| 11 | AI agent identity lifecycle | backend | agentic AI | Persistent registry, consent flow, credential rotation, drift detection |
| 12 | Fraud: TOR/VPN/proxy detection | backend | ITDR/fraud | IP intelligence integration, geo-velocity anomaly |
| 13 | **ReBAC tuple store** (P0) | backend | Google Zanzibar / fine-grained authz | PostgreSQL tuple store + repository layer. See docs/research/rebac-zanzibar-fine-grained-authz.md §13 |
| 14 | **ReBAC schema DSL parser** (P0) | backend | Zanzibar schema language | Parse `definition type { relation, permission }` syntax |
| 15 | **ReBAC graph traversal engine** (P0) | backend | Relationship graph | Recursive check with depth limiting, memoization |
| 16 | **ReBAC check/write API** (P0) | backend | REST + gRPC | /api/v1/policy/rebac/check, /tuples, /list-objects |
| 17 | **ReBAC evaluator integration** (P0) | backend | Policy service | Wire ReBAC as 3rd layer in evaluator.Check() pipeline |
| 18 | **ReBAC Redis caching** (P1) | backend | Performance | Tuple cache with write invalidation |
| 19 | **ReBAC console UI** (P2) | frontend | Developer experience | Schema editor, tuple browser, check playground |
| 20 | **Journey definition store** (P0) | backend | Configurable auth flows | PostgreSQL tables for journey_definitions, bindings, executions, sessions |
| 21 | **Journey engine + JDL parser** (P0) | backend | Identity orchestration | YAML Journey Definition Language + state machine executor. See docs/research/identity-orchestration-journeys.md |
| 22 | **Journey core nodes** (P0) | backend | Auth extensibility | password_verify, risk_assessment, mfa_orchestrate, issue_tokens, conditional (CEL) |
| 23 | **Journey management + execution API** (P0) | backend | REST + gRPC | CRUD journeys, bindings, start/resume execution, dry-run |
| 24 | **Auth service integration** (P0) | backend | Replace hardcoded Login() | Wire Journey Engine as auth flow dispatcher (backward-compatible default journey) |
| 25 | **Journey templates + dry-run** (P1) | backend | Developer experience | Pre-built templates (login, registration, recovery) + simulation mode |
| 26 | **Journey visual builder** (P2) | frontend | Console UI | Drag-and-drop canvas with React Flow, node config panel, edge labels |
| 27 | **Journey analytics** (P3) | backend | Observability | Conversion rates, drop-off points, per-step latency dashboards |
| 28 | **Cloud federation data model + service** (P0) | backend | Multi-cloud identity | PostgreSQL tables for configs, role mappings, attribute mappings. See docs/research/cloud-iam-federation.md |
| 29 | **Claim mapping engine** (P0) | backend | SAML/OIDC claims | Transform GGID attributes to AWS/Azure/GCP-specific SAML attributes |
| 30 | **AWS SAML federation module** (P0) | backend | AWS IAM | Role ARN generation, https://aws.amazon.com/SAML/Attributes/* attributes |
| 31 | **Azure SAML federation module** (P0) | backend | Azure AD | Azure claim URIs, app role mapping |
| 32 | **Federation login + Terraform snippet API** (P0) | backend | Developer experience | POST /cloud-federation/{id}/login + GET /{id}/terraform |
| 33 | **GCP workforce federation module** (P1) | backend | GCP IAM | SAML attributes for Workforce Identity Pool, CEL attribute mapping |
| 34 | **SCIM client** (P1) | backend | Auto-provisioning | Push user changes to AWS IAM Identity Center via SCIM 2.0 |
| 35 | **Federation health monitoring** (P1) | backend | Operations | Periodic checks: metadata access, cert expiry, SCIM connectivity |
| 36 | **Federation setup wizard** (P2) | frontend | Console UI | Multi-step wizard with metadata download, Terraform copy, test login |
| 37 | **Multi-hash password verifier** (P0) | backend | Migration compatibility | bcrypt, PBKDF2, scrypt, LDAP SSHA, SHA256 verification. See docs/research/data-migration-bulk-import.md |
| 38 | **Transparent rehashing** (P0) | backend | Password security | Auto-upgrade legacy hashes to Argon2id on successful login |
| 39 | **Bulk import pipeline** (P0) | backend | User migration | Async job-based JSON/CSV import with batch processing, progress tracking |
| 40 | **Dry-run validation** (P0) | backend | Safety | Validate import data without committing; return error report |
| 41 | **Lazy migration engine** (P1) | backend | Zero-downtime migration | Legacy DB connector, per-tenant config, JIT user creation on login |
| 42 | **Attribute + role mapping engine** (P1) | backend | Data transformation | Configurable field + role mapping from legacy schema to GGID schema |
| 43 | **Import wizard + dashboard** (P2) | frontend | Console UI | Multi-step wizard (upload, map, validate, import) + migration dashboard with stats |
| 44 | **Device posture API + evaluation engine** (P0) | backend | ZTNA integration | Standardized posture API for ZTNA brokers + configurable policy engine. See docs/research/ztna-broker-integration.md |
| 45 | **Gateway device posture middleware** (P0) | backend | Per-request enforcement | DevicePosture middleware in gateway chain, Redis-cached posture check |
| 46 | **SAML groups claim standardization** (P0) | backend | ZTNA policy | Standardized groups attribute in SAML assertions for ZTNA broker policies |
| 47 | **SCIM outbound client** (P0) | backend | ZTNA provisioning | Push users/groups to Zscaler/Cloudflare/Twingate/Tailscale via SCIM 2.0 |
| 48 | **CAEP event transmitter** (P1) | backend | Continuous verification | Push CAEP SET tokens (session-revoked, credential-change) to ZTNA brokers |
| 49 | **ZTNA provider setup guide generator** (P1) | backend | Developer experience | Auto-generate provider-specific Terraform/config snippets |
| 50 | **ZTNA dashboard + provider wizard** (P2) | frontend | Console UI | Provider status cards, posture compliance, CAEP events, setup wizard |

### P3 — Quality/infrastructure improvements

| # | Feature | Owner | Notes |
|---|---------|-------|-------|
| 10 | Console loading/error states | frontend | Remaining pages: ip-allowlist, tenant-config, branding-custom, settings/page, notifications/templates |
| 11 | i18n extraction | frontend | 1051 hardcoded strings -> messages/en.json, zh.json |
| 12 | SDK parity completion | arch | Node SDK admin extensions, React hooks for risk/SOD/PAR |

## Currently Dispatched (Next 24h)

### Backend
1. (standby)

### Arch
1. Research OAuth 2.1 / FAPI 2.0 / FedCM gaps → DONE (docs/research/oauth21-fapi-fedcm-gap.md)
2. Round 26 E2E regression test
3. Release v0.2.5 verification

### Frontend
1. Console loading/error states for remaining pages
2. Passkey health dashboard

### Docs/Research
1. OAuth 2.1 / FAPI 2.0 research → DONE
2. Console mock-pages audit (continuing)
3. ReBAC / Zanzibar fine-grained authz → DONE (docs/research/rebac-zanzibar-fine-grained-authz.md) — 7 backlog items added
4. Identity Orchestration / Auth Journeys → DONE (docs/research/identity-orchestration-journeys.md) — 8 backlog items added
5. Cloud IAM Federation → DONE (docs/research/cloud-iam-federation.md) — 9 backlog items added
6. Data Migration / Bulk Import → DONE (docs/research/data-migration-bulk-import.md) — 8 backlog items added
7. ZTNA Broker Integration → DONE (docs/research/ztna-broker-integration.md) — 7 backlog items added

## Rules

- Each task owner must report: commit hash + make test result
- No modification of other teammates' directories
- Gap status changes require verification (see docs/gap-maintenance-rules.md)
- Research findings go to docs/research/*.md before entering backlog
- All dependencies use @latest

## Research Pipeline

Active research topics:
- ZTNA Broker Integration → DONE (docs/research/ztna-broker-integration.md)
- Data Migration / Bulk User Import → DONE (docs/research/data-migration-bulk-import.md)
- Cloud IAM Federation (AWS/Azure/GCP) → DONE (docs/research/cloud-iam-federation.md)
- Identity Orchestration / Configurable Auth Journeys → DONE (docs/research/identity-orchestration-journeys.md)
- ReBAC / Google Zanzibar fine-grained authorization → DONE (docs/research/rebac-zanzibar-fine-grained-authz.md)
- OAuth 2.1 / FAPI 2.0 compliance gap analysis
- PQC migration for IAM systems
- AI agent identity governance patterns
- NIS2 / CRA compliance for IAM vendors
- Console mock data audit
- **Next**: Passwordless migration strategies / JIT user provisioning

See docs/research/ for full research docs.
---

## Access Broker / Identity-Aware Proxy (ZTNA) (2026-07-17 第12小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + arch

**市场背景**: Identity-Aware Proxy (IAP) / ZTNA 是 2025 企业安全最高优先级投资。Cloudflare Access、Google IAP、AWS Verified Access 全部增长 40%+。核心价值：**替代 VPN** — 任何内部应用（Jenkins/Grafana/Jupyter/internal dashboard）放在 GGID 身份层后面，按身份+上下文授权访问，无需 VPN。

**GGID 现状审计**: gateway 已有 reverse proxy + route config（path_prefix→backend URL）+ JWT 验证 + ABAC + CAE。**基础设施完全就绪**，差一个"受保护应用注册"层：
- 当前 routes 硬编码为 GGID 自己的 7 个服务（identity/auth/oauth/...）
- 无法注册"外部应用"（如 https://ggid.example.com/grafana/ → 代理到 Grafana 实例 + GGID JWT 验证）
- 无 per-application access policy（"Grafana 只允许 SRE 角色 + 工作时间 + 设备可信"）

**业务价值**: HIGH
- GGID 从"IAM 平台"升级为"Zero Trust Access Platform"（市场天花板提升 3-5x）
- 替代 VPN 是 CISO 2025 第一预算项
- GGID 已有全部组件：reverse proxy + JWT + ABAC + CAE + device trust + ITDR
- 差异化：国内无同类产品（竹云/玉籽数科均无 ZTNA 能力）

**实现难度**: Medium
- 实现路径（完整一步到位，不降级）：
  1. **受保护应用注册**：protected_apps 表（name, path_prefix, backend_url, access_policy_id, tenant_id, auth_mode[jwt/session/anonymous]）
  2. **Gateway 动态路由**：启动时 + 热加载（Redis pub/sub 或 DB 轮询）从 protected_apps 表构建路由表，与现有静态路由合并
  3. **Per-app access policy**：复用 ABAC PDP（$security.* + $data.classification），增加 $app.name / $app.backend 属性
  4. **Session passthrough**：对不支持 OAuth 的传统应用（Jenkins/Grafana），GGID 完成认证后注入 header（X-Authenticated-User + X-Tenant-ID），后端配置 trusted header 模式
  5. **Access logs**：每次受保护应用访问写 audit 事件（与 ITDR 联动：异常访问模式自动检测）
  6. **Console 管理页面**：受保护应用列表 + CRUD + 实时访问日志 + access policy 配置器（可视化条件构建器）

**兼容性**: gateway 已有全部基础设施（reverse proxy + JWT + ABAC + CAE），纯增量注册层 + 动态路由
**工作量**: ~5d（gateway 动态路由 2d + protected_apps API 1d + session passthrough 1d + console UI 1d）

---

## Quantum-Safe Access (后量子密码学准备) (2026-07-17 第12小时研究) - Priority: P3 - Status: Watch - Suggested: IAMExpert

**描述**: Cloudflare Access 2025 已支持 quantum-safe access。NIST PQC 标准（ML-KEM/ML-DSA）2024 年最终化。GGID 已有 gmsm 库（可扩展 ML-DSA JOSE 支持，researcher a07e344c 已标记 gap）。Harvest Now Decrypt Later 威胁要求长寿命凭证提前迁移。跟踪标准落地，不立即编码。

---

## SMS OTP Provider Integration (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: Auth 服务有 SMS OTP 代码验证逻辑（http.go:1598）但只有 DEV 模式日志输出（log.Printf("[DEV] phone OTP..."），没有真正的 SMS 提供商集成。Passwordless 场景需要 SMS 验证码。

**业务价值**: MEDIUM — 补齐 passwordless 方法链 | **实现难度**: Low
- 实现路径：
  1. pkg/notification/sms.go — SMS provider 接口
  2. Twilio provider 实现（或 AWS SNS）
  3. auth/cmd/main.go 注入 provider（env TWILIO_ACCOUNT_SID + TWILIO_AUTH_TOKEN）
  4. 替换 http.go:1598 的 DEV 日志为 provider.Send()
- 参考: docs/research/ciam-2025-trends-gap-analysis.md

---

## Consent Versioning + Withdrawal API (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: OAuth consent 只有一 checkbox 级别。GDPR/CCPA 要求版本化、可撤回、可审计的 consent 记录。当前 consent analytics 端点存在但没有 consent 版本管理或撤回 API。

**业务价值**: MEDIUM-HIGH — 合规要求 | **实现难度**: Medium
- 实现路径：
  1. consent_records 表（user_id, client_id, scopes, version, granted_at, withdrawn_at）
  2. POST /api/v1/oauth/consent/withdraw — 撤回 consent
  3. GET /api/v1/oauth/consent/history — consent 变更历史
  4. Consent 版本绑定到 scope 定义变更
- 参考: docs/research/ciam-2025-trends-gap-analysis.md

---

## B2B CIAM 增强：客户身份管理 + 自助注册 + 品牌定制 (2026-07-17 第13小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + frontend

**市场背景**: Auth0 Customer Identity Trends Report 2025 — CIAM 市场 $27B（2025），4 大趋势：passwordless 默认化、AI agent 身份、身份欺诈防护、B2B CIAM（组织/租户层级管理）。GGID 当前是 workforce IAM，B2B CIAM 扩展是市场天花板翻倍路径。

**GGID 现状审计（CIAM 能力部分就绪）**：
- ✓ 社交登录（social providers stats handler）
- ✓ OAuth consent screen + consent override
- ✓ 租户品牌（branding_pg.go + client_branding.go）
- ✓ 密码自助重置（forgot + reset）
- ✓ 组织管理（org 服务 LTREE 树）
- ✓ 注册页面（register/page.tsx）
- ✗ 缺：客户自助组织管理（B2B 客户自己注册组织 + 管理成员）
- ✗ 缺：渐进式注册（progressive profiling — 首次只收 email，后续逐步补全）
- ✗ 缺：MFA 强制策略（CIAM 场景：高风险操作才要求 MFA，而非每次登录）
- ✗ 缺：客户身份欺诈防护（step-up based on risk score + bot detection）

**业务价值**: HIGH（从 workforce 扩展到 CIAM = 市场翻倍，$27B TAM）
**实现难度**: Medium
- 完整实现路径（不降级）：
  1. **B2B 自助注册**：POST /api/v1/auth/register-organization — 客户自己创建组织（自动分配 tenant + admin 角色 + 组织根节点）
  2. **渐进式注册**：user.profile_completeness 字段 + 登录后检测缺失字段 + 前端引导补全
  3. **品牌深度定制**：登录/注册/MFA 页面主题色 + Logo + CSS + 自定义域名（CNAME 验证 → tenant_branding 表扩展）
  4. **风险驱动 MFA**：risk_engine.Evaluate > 0.5 → step-up MFA（低风险无缝登录，高风险才要求 MFA）
  5. **身份欺诈防护**：bot detection（已有 middleware）+ velocity check（已有 risk_engine）+ disposable email 阻止 + email verification 强制
  6. **客户旅程分析**：/api/v1/auth/journey-analytics — 注册转化漏斗、MFA 放弃率、社交登录偏好

**兼容性**: 全部基于现有组件扩展；oauth 服务已有品牌+consent+社交+注册基础设施

---

## 可验证凭证 (Verifiable Credentials) / EUDI Wallet 支持 (2026-07-17 第13小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: KuppingerCole EIC 2025 CIAM 第一推荐：EUDI Wallet（欧盟数字身份钱包 2026 强制）+ W3C Verifiable Credentials。去中心化身份从概念进入落地期。Auth0 已推出 VC 支持。

**GGID 现状**: SCIM（同步）+ OAuth/OIDC（联邦）+ SAML（企业 SSO）已完整。缺：VC 签发/验证/呈现。

**业务价值**: MEDIUM-HIGH（欧盟市场准入 + 政府/金融场景差异化）
**实现难度**: High（DID 方法选择 + VC JSON-LD 签名 + VP 验证 + Wallet 交互协议）
- 完整路径：DID:web 方法 → VC签发（JSON-LD + Ed25519/SM2 签名）→ VP 验证端点 → statusList 吊销 → Console 签发管理

---

## PKCE Global Mandatory (2026-07-17 研究驱动 OAuth 2.1) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: OAuth 2.1 (draft-15) 要求所有客户端使用 PKCE。当前 PKCE 在 RequirePKCE=true 或 public client 时强制，但不是全局默认。应改为默认强制，仅允许受信任的 confidential client opt-out。

**业务价值**: HIGH — OAuth 2.1 合规基础要求 | **实现难度**: Low
- 实现路径：
  1. oauth/internal/conf: `RequirePKCE` 默认 true
  2. server.go:331-332 — 移除 RequirePKCE 条件检查，改为全局检查（除非 client 显式标记 PKCE 豁免）
  3. 客户端管理增加 `pkce_optional` 字段（仅 trusted confidential clients）
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## RAR (Rich Authorization Requests) (2026-07-17 研究驱动 FAPI 2.0) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: FAPI 2.0 要求 RAR 支持 — `authorization_details` 参数允许细粒度、类型化的授权请求（如 "transfer 100 EUR to account X"），比传统 scope 更精确。GGID 未实现。

**业务价值**: MEDIUM-HIGH — 金融场景必需 | **实现难度**: Medium
- 实现路径：
  1. OAuth authorize 端点接收 `authorization_details` JSON 数组
  2. 类型验证（每种 type 有独立 schema）
  3. consent screen 展示 RAR details
  4. token 中嵌入 authorization_details claim
  5. 资源服务验证 details
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## Agent Token Revocation + Audit (2026-07-17 研究驱动 MCP Auth) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: GGID 有 AI agent 注册和 token 发放，但缺少：(1) agent token 主动撤销 API (2) MCP server 访问审计（哪个 agent 访问了哪个 server）。Agent 安全面需要这些能力。

**业务价值**: MEDIUM — AI agent 安全闭环 | **实现难度**: Medium
- 实现路径：
  1. POST /api/v1/agents/{id}/revoke — 撤销 agent 所有 token
  2. 审计事件：agent token 发放 + MCP server 访问写入 audit_events
  3. Console 页面：agent 活动审计日志
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## Device Posture as Policy Input (2026-07-17 研究驱动 ZT) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: ZT posture score (当前 65/100) 仅作为信息展示，未接入 Access Broker PDP 决策。NIST SP 800-207 要求设备状态作为访问策略输入（如 score < 50 时拒绝访问）。

**业务价值**: HIGH — ZT 闭环核心 | **实现难度**: Medium
- 实现路径：
  1. Access Broker evaluateAccessPolicy() 增加 device_posture 输入
  2. 策略条件支持 `device_posture >= N` 表达式
  3. posture 低于阈值时返回 deny + 原因
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## Compliance Evidence Export API (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend + docs

**描述**: SOC2/ISO27001 审计需要结构化合规证据（用户访问权限、MFA 覆盖率、策略变更历史）。GGID 有全部数据但无导出 API。CCM 工具（Drata/Vanta）需要 API 集成。

**业务价值**: HIGH — 企业合规必需 | **实现难度**: Medium
- 实现路径：
  1. GET /api/v1/audit/compliance/export?framework=soc2&period=Q2-2025
  2. 返回 JSON：access_reviews[], mfa_coverage, policy_changes[], privileged_access_log[]
  3. 支持 SOC2 (Trust Services Criteria) 和 ISO 27001 (Annex A) 框架
  4. 文档：docs/compliance-evidence-guide.md
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## Scheduled IGA Campaigns (2026-07-17 研究驱动) - Priority: P3 - Status: Proposed - Suggested: backend

**描述**: IGA access certification campaigns 需要手动触发。企业合规要求季度自动执行（SOC2 CC6.1, CC6.2）。

**业务价值**: MEDIUM — 合规自动化 | **实现难度**: Low
- 实现路径：
  1. cron/scheduler 在每季度首日自动创建 campaign
  2. 通知所有 certifiers（邮件 + console）
  3. 记录超时未完成审核
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## OWASP MCP Top 10 合规：GGID MCP Server 安全加固 (2026-07-17 第14小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: OWASP MCP Top 10 (2025) 发布 — 2026 年 1-2 月研究者提交 30+ MCP CVE（含 CVSS 9.6 mcp-remote，43.7 万下载）。MCP 已成 AI agent 与企业系统的默认连接协议。GGID 有自己的 MCP Server（13 个 LLM 管理 tools），必须对标 OWASP MCP Top 10 加固。

**OWASP MCP Top 10 与 GGID 对照**：

| OWASP # | 风险 | GGID 现状 | 缺口 |
|---------|------|-----------|------|
| MCP01 | Token 管理 | Client 用静态 Bearer token | 需短时 OAuth 2.1 token + scope |
| MCP02 | 权限蔓延 | Tools 无 scope/per-tool authz | 需 per-tool RBAC + scope 过期 |
| MCP03 | Tool Poisoning | 无签名/版本锁定 | 需 tool 签名 + schema 固定 |
| MCP04 | 供应链 | 无 AIBOM | 需 MCP 依赖清单 + provenance |
| MCP05 | 命令注入 | Tools 参数化较好 | 需输入校验审计 |
| MCP06 | Prompt Injection | 无 context 隔离 | 需 instruction quarantine |
| MCP07 | 认证不足 | **无 MCP auth 中间件**（grep 无结果）| 需 OAuth 2.1 + per-server audience |
| MCP08 | 审计缺失 | 无 tool invocation 审计 | 需 immutable audit log + behavioral monitoring |
| MCP09 | Shadow MCP | 无发现机制 | 需 allowlist + discovery |
| MCP10 | Context 泄露 | 无 context 隔离 | 需 per-session context scope |

**最紧急（MCP07）**：GGID MCP Server **无认证中间件**（grep 零结果）— 任何能访问 MCP 端口的人可调用 13 个管理 tool（创建用户/角色/策略）。这与之前 SCIM 直连同型问题。

**业务价值**: HIGH
- MCP 安全是 AI 时代的 OWASP Top 10 等价物 — 2026 合规刚需
- GGID 作为 IAM 平台保护 AI agent 身份 = 差异化叙事
- MCP07 + MCP02 + MCP08 三项是安全底线

**实现难度**: Medium（基于已有 OAuth 2.1 + RBAC + audit 基础设施）
- 完整实现路径（按 OWASP 优先级）：
  1. **MCP07 认证**：MCP Server 加 JWT 验证（复用 gateway JWTAuth）+ per-tool scope 检查（tool 需声明 required_scope）
  2. **MCP08 审计**：每次 tool invocation 写 audit 事件（tool_name/caller/params/result/duration）
  3. **MCP01 Token**：静态 Bearer → OAuth 2.1 token exchange（RFC 8693 — 已在 backlog B-06 中）
  4. **MCP02 Scope**：per-tool RBAC — tool 注册时声明 required_permissions，调用时检查 caller 是否有权限
  5. **MCP10 Context**：per-session context scope + ephemeral memory
  6. **MCP03/04 供应链**：tool 签名 + AIBOM
  7. **MCP09 Discovery**：MCP allowlist + shadow detection
  8. **MCP06 Isolation**：instruction quarantine（retrieved content 标记 untrusted）
  9. **MCP05 Injection**：输入校验审计（已有基础，补审计）

**兼容性**: 复用 OAuth 2.1 (RFC 8693 B-06)、RBAC (policy 服务)、audit publisher (NATS)
