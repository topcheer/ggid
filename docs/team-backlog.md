# GGID Team Backlog

*Last updated: 2026-07-15 (Round 54 E2E; multi-tenant login + onboarding gaps identified by user)*

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

### P1 — Productization gaps (user-reported)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 13 | Login page lacks tenant selection | frontend | console/src/app/login/page.tsx | [DONE] 53771ccc | Tenant slug input + system init check |
| 14 | No first-deploy onboarding flow | frontend+backend | console/src/app/onboarding/ | [DONE] 15174474 | Bootstrap API + login warning; console wizard can use /api/v1/system/bootstrap |
| 15 | Hardcoded DEFAULT_TENANT_ID | frontend | console/src/lib/api-config.ts | [DONE] 53771ccc | Now uses tenant_slug from login form, DEFAULT_TENANT_ID only as fallback |
| 16 | System init detection API | backend | services/identity/ | [DONE] 6f23b400 | GET /api/v1/system/initialized works |
| 17 | Tenant resolve API | backend | services/identity/ | [DONE] 6f23b400 | GET /api/v1/tenants/resolve works |

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
1. i18n Batch 2+3: internationalize remaining 700 pages (ACTIVE)
2. Login page: add tenant selection (after backend API ready)
3. Onboarding flow: real first-deploy setup wizard
4. Console loading/error states for remaining pages

### Docs/Research
1. OAuth 2.1 / FAPI 2.0 research → DONE
2. Console mock-pages audit (continuing)

## Rules

- Each task owner must report: commit hash + make test result
- No modification of other teammates' directories
- Gap status changes require verification (see docs/gap-maintenance-rules.md)
- Research findings go to docs/research/*.md before entering backlog
- All dependencies use @latest

## Research Pipeline

Active research topics:
- OAuth 2.1 / FAPI 2.0 compliance gap analysis
- PQC migration for IAM systems
- AI agent identity governance patterns
- NIS2 / CRA compliance for IAM vendors
- Console mock data audit

See docs/research/ for full research docs.
---

# IAM 趋势研究 Backlog (2026-07-16 23:25 cron-2 第1小时: 国密+NHI)

## 国密算法 SM2/SM3/SM4 KeyProvider 实现 (2026-07-16) - Priority: P1 - Status: Proposed

**描述**: 为中国市场合规（商用密码应用安全性评估/密评）实现国密算法支持。GGID 已有 KeyProvider 接口（local/AWS KMS/PKCS11/Vault），设计文档 docs/research/kms-hsm-comprehensive-design.md 中 9 处提及 SM2/SM3/SM4，但代码中零实现。

**业务价值**: HIGH
- 密评是政府/国企/关键基础设施的强制合规要求（2025年全面铺开）
- 中国市场 IAM 产品的准入门槛
- 支持 SM2 签名 JWT、SM3 密码杂凑、SM4 数据加密

**实现难度**: Medium
- 成熟 Go 库可用：github.com/emmansun/gmsm（活跃维护，推荐）或 tjfoc/gmsm
- 实现路径：
  1. 新增 `pkg/crypto/key_provider_sm.go` 实现 KeyProvider 接口
  2. JWT 签名算法增加 SM2-with-SM3 (alg: "SM2") 支持（oauth 服务 JWKS）
  3. 密码哈希可选 SM3（替代 Argon2id，用于密评场景）
  4. 数据加密层增加 SM4-GCM 选项（替代 AES-256-GCM）
  5. 密评场景配置开关：tenant 级别选择 crypto suite (international/GM)

**兼容性**: 现有 KeyProvider 接口无需改动，纯增量实现

---

## Agentic Access Management (AAM) / 非人身份治理深化 (2026-07-16) - Priority: P1 - Status: Proposed

**描述**: 2025 年 IAM 最大趋势：AI Agent 自主行动，非人身份（NHI）数量超过人类身份。行业从"管理访问"转向"治理行动"（intent-aware access）。Oasis 等公司推出 AAM 治理框架。GGID console 已有 7 个 agent-* 页面（UI 领先），oauth 服务有 agent_lifecycle_handler.go 和 agent_review_handler.go，但 identity 服务缺乏完整的 NHI 生命周期（service account provisioning/deprovisioning/rotation）。

**业务价值**: HIGH
- 身份将成为 AI 的控制平面（2026 预测）
- GGID 已有先发优势（agent UI 全套页面）
- 差异化竞争点：国内 IAM 产品几乎没有 agent 身份治理

**实现难度**: Medium-High
- 实现路径：
  1. 审查现有 agent-* 后端实现的完整度（oauth 服务已有部分）
  2. identity 服务增加 NHI 类型（agent/service_account/api_key/workload）
  3. 实现 NHI Provisioning：创建即治理（默认最小权限、自动轮换）
  4. intent-aware policy：policy 服务增加"意图"维度评估
  5. 对标 AAM Governance Framework 做成熟度自评

**兼容性**: 基于现有 agent 相关代码扩展

---

## 密评合规套件 (2026-07-16) - Priority: P2 - Status: Proposed

**描述**: 基于国密算法实现的完整密评合规方案：SM2 身份鉴别（USBKey/协同签名）、SM4 传输加密、SM3-MAC 完整性保护、密钥管理合规。依赖国密 KeyProvider 实现完成后进行。

**业务价值**: MEDIUM-HIGH（与国密 KeyProvider 绑定）
**实现难度**: High（涉及硬件对接、密评流程）
**实现路径**: 国密 KeyProvider → 国密 SSL/TLS → 密评文档与工具 → 认证

---

## WebAuthn Signal API 支持 (2026-07-17 第2小时研究) - Priority: P2 - Status: Proposed - Suggested: frontend + backend

**描述**: WebAuthn Level 3 的 Signal API 允许 RP 向客户端认证器同步凭证状态。GGID 当前 passkey 管理页删除凭证后，用户设备上的 passkey 仍残留在自动填充列表中（点击后登录失败，UX 差）。Signal API 三个方法：
- `signalAllAcceptedCredentials()`：登录成功后同步有效凭证全集，自动隐藏已删除的 passkey（推荐每次登录后调用）
- `signalCurrentUserDetails()`：用户改名/改邮箱后更新认证器中的 user.name/displayName
- `signalUnknownCredential()`：无效凭证登录尝试时实时删除

**业务价值**: MEDIUM-HIGH
- Chrome 132+ 默认启用，Safari 26+ (iCloud Keychain) 支持，Android Chrome 143 beta
- FIDO 2025 主推特性，企业级 passkey 部署的运维刚需
- 9 个 passkey 管理页面已就绪，只差同步闭环

**实现难度**: Low-Medium
- 前端（主）：console 登录成功后 + passkey 删除后调用 Signal API（特性检测 `PublicKeyCredential.signalAllAcceptedCredentials`）
- 后端（辅）：新增 GET /api/v1/auth/webauthn/credentials/valid-ids 返回用户有效 credential ID 列表
- 注意 Safari 26 promise bug (WebKit #298951)，需 try/catch 兜底

**兼容性**: 纯增量，go-webauthn v0.17.4 无需升级

---

## Conditional Create 密码无感升级 Passkey (2026-07-17 第2小时研究) - Priority: P2 - Status: Proposed - Suggested: frontend

**描述**: WebAuthn L3 的 conditional create（mediation: "conditional"）让用户密码登录成功后，浏览器自动在密码管理器界面提议创建 passkey —— 无需跳转注册流程。FIDO Passkey Index 2025 显示 93% 账户技术上可升级 passkey，但转化率取决于注册摩擦。这是提升 passkey 采用率的标准做法。

**业务价值**: MEDIUM-HIGH
- 直接提升 passkey 渗透率（GGID 已有 passkey-health dashboard backlog #8，数据会好看）
- 企业 SSO 场景的"静默迁移"路径：员工无感知完成密码→passkey 升级

**实现难度**: Medium
- 前端：密码登录成功后调 navigator.credentials.create({mediation:"conditional"})，需要 Chrome 136+/ Safari 18+
- 后端：webauthn register begin 需支持 conditional 会话（challenge 预生成 + 会话保持）
- 需要 tenant 级开关（部分企业不希望自动提示）

**兼容性**: 增量，现有 passwordless 流程可复用

---

## Credential Exchange Protocol (CXP) 跨平台凭证迁移 (2026-07-17 第2小时研究) - Priority: P3 - Status: Watch - Suggested: IAMExpert 跟踪

**描述**: FIDO 联盟 2025 推进的 Credential Exchange Protocol 允许 passkey 在密码管理器之间安全迁移（1Password ↔ Google Password Manager ↔ iCloud Keychain）。解决企业用户换设备/换密码管理器时 passkey 锁定问题。标准尚在 draft 阶段。

**业务价值**: MEDIUM（当前）→ HIGH（标准落地后）
**实现难度**: High（服务端导出/导入加密凭证包）
**行动**: 跟踪标准进展，标准稳定后评估 GGID 作为 RP 的支持方案。无需立即编码。

---

## ITDR 假数据清理 + 检测引擎落地 (2026-07-17 第3小时研究+审计) - Priority: P0 - Status: Proposed - Suggested: backend + frontend + IAMExpert

**市场背景**: ITDR 市场 2025 年 $5.6B → 2034 年 $29.4B（CAGR 20.3%），是 IAM 增长最快的细分领域。Gartner 将 ITDR 列为身份安全必备能力。

**GGID 现状审计（第3小时深度核查）**:
- 资产：26 个 ITDR 相关 console 页面（itdr-dashboard, threat-hunting-workbench, risk-engine-dashboard, golden-ticket-detect 等）
- **真实检测（4 项）**: impossible travel、credential stuffing（detect+stats+block）、risk-assess、session anomaly-score、HIBP breach check
- **P0 假数据发现**:
  1. `sdk/react/src/useITDRDashboard.ts` — `setTimeout(r,400)` 假延迟 + 100% 硬编码威胁（47 受影响用户、auto_response_enabled: true —— SOC 误以为自动响应已开启，危险）
  2. `services/auth/internal/server/itdr_detections_handler.go` — handleITDRDetections 硬编码 5 威胁+5 规则+3 playbook（已注册路由但无人调用）
  3. **规模扩散：sdk/react 490 个 hooks 中 150 个（31%）存在 setTimeout 假数据模式**；抽样 10 个 ITDR dashboard 页面 8 个零 API 调用

**业务价值**: HIGH
- 安全运营场景展示假威胁数据 = 产品信誉致命伤（客户 POC 必查）
- ITDR 是最高客单价 IAM 模块，真实检测引擎是核心竞争力

**实现难度**: High（检测引擎）/ Medium（假数据清理）
- 实现路径：
  1. **P0 紧急**：150 个假 hooks 分批处理 — 有对应真端点的接线；无端点的页面标注"演示数据"横幅（ frontend 主导，~3 天/批 × 5 批）
  2. **P0**: useITDRDashboard 接 /api/v1/auth/itdr/detections；该端点改为从 audit 事件真实统计（auth 登录失败计数、凭证填充检测记录）
  3. **P1**: ITDR 检测引擎统一 — audit 服务消费 NATS JetStream 事件流 → 规则引擎（已有 4 项真检测逻辑迁移统一）→ 检测结果写 DB → dashboard 查询
  4. **P2**: threat-intelligence-feed 接 abuse.ch/MISP 开源威胁情报

**兼容性**: audit 服务已有 NATS consumer 框架，检测引擎为增量模块

## RFC 8693 标准 Token Exchange Grant (2026-07-17 研究) - Priority: P1 - Status: Proposed - Suggested: backend

**市场背景**: AI agent 委托授权已成为 2026 年 IAM 核心场景。MCP 授权规范基于 OAuth 2.1，agent 代用户操作的标准模式就是 RFC 8693 token exchange（delegation + act claim）。Salesforce Agentforce、Microsoft Copilot Studio、AWS Bedrock Agents 全部收敛到"从用户 token 派生收窄权限 token"模式。

**GGID 现状**:
- 已有：自定义端点 POST /api/v1/oauth/token-exchange-delegation（JSON body，delegation chain 存内存 map）
- 缺失：标准 grant `urn:ietf:params:oauth:grant-type:token-exchange` 在 /oauth/token 中不可用
- 风险：delegationChains 是 sync.Map 内存存储，重启丢失（与 Round 5-6 修复的 6 个内存存储同类问题）

**业务价值**: HIGH — MCP 客户端和标准 OAuth 库（oauth4webapi、AppAuth）期望标准 grant；自定义端点需要客户写定制代码，是集成障碍。

**实现难度**: Medium
- Phase 1（P1）：/oauth/token grant switch 加 token-exchange case；service.TokenExchange() 验证 subject_token、构建 act claim（支持嵌套）、强制 scope ⊆ subject scope、返回 issued_token_type；客户端加 token_exchange_allowed 策略位（~1 天）
- Phase 2（P2）：delegationChains 迁移 PG（参照 backup_codes_pg.go EnsureSchema 模式）；subject token 撤销时级联撤销派生 token（~0.5 天）
- Phase 3（P2）：Go/Node/Python SDK 加 TokenExchange()（~0.5 天）

**兼容性**: 纯增量，不影响现有 grant；研究文档 docs/research/token-exchange-standard-grant-gap.md

## ML-DSA JOSE 后量子 JWT 签名准备 (2026-07-17 研究) - Priority: P2 - Status: Proposed - Suggested: backend（spike 先行）

**市场背景**: draft-ietf-cose-dilithium 已到 draft-10（即将成为 RFC），定义 JOSE alg ML-DSA-44/65/87 和 JWK kty=AKP。CNSA 2.0 要求 2027 年新系统支持 PQC；Auth0/Okta 2025 年已发布 PQC 路线图，企业 RFP 开始询问。

**GGID 现状**:
- 架构已准备好算法敏捷性：KeyProvider 抽象（SM2 已验证模式）、alg 白名单（7cea65ab）、kid 统一派生（a3e29625）
- TLS 层 ML-KEM 混合密钥交换由 Go 1.25 stdlib 免费获得（需验证启用）
- 完全缺失：ML-DSA 签名、AKP JWK、SDK PQC 验证

**业务价值**: MEDIUM（当前）→ HIGH（2027 后）— 政府/金融/密评客户的前置门槛；及早布局成本低（KeyProvider 模式已跑通 SM2）

**实现难度**: Medium（spike 1 天验证 cloudflare/circl FIPS 204 最终测试向量兼容性；纯 Go 无 CGo 符合 CGO_ENABLED=0 构建）
- 顺序：TLS ML-KEM 验证（0.5 天）→ circl spike（1 天）→ AKP JWKS + 白名单（0.5 天）→ Go SDK 验证 feature flag（0.5 天）→ 租户 PQC 迁移指南（0.5 天）
- 注意：ML-DSA-65 签名 3.3KB，JWT 体积 ~4-5KB（HTTP header 8KB 内可行，cookie 场景需评估）

**兼容性**: 增量；spike 未通过前不启动实现；研究文档 docs/research/mldsa-jose-pqc-jwt.md

---

## IGA 闭环：审查决策 → 实际权限回收 (2026-07-17 第4小时研究) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: 审计发现 GGID IGA 工作流"开环"：access review campaign 的 submit 仅把 revoke 决策字符串存入**内存** campaignStore（重启丢失），且不触发任何实际权限回收。IGA 的核心价值闭环是 review→revoke→deprovision 自动化。

**现状审计**: 18 个 IGA console 页面（certification/campaigns/attestation/entitlement/recertification），policy 服务有完整 campaign CRUD 但 (1) 内存存储不持久 (2) revoke 决策无执行器。

**业务价值**: HIGH
- 合规刚需（SOX/等保：定期审查 + 证据 + 及时回收）
- 客户 POC 必演场景："审查员点 revoke → 用户权限立即消失"

**实现难度**: Medium
- 实现路径：
  1. campaigns 迁移 Postgres（新表 access_review_campaigns + items，RLS）
  2. revoke 执行器：submit 时 decision=revoke → policy 服务删除 role assignment（已有 DeleteRoleAssignment）→ audit 事件记录回收证据链
  3. campaign 完成率统计 + 逾期提醒（通知 webhook）

**兼容性**: 复用现有 role assignment 删除 API + audit publisher

---

## GenAI 辅助审查决策 (2026-07-17 第4小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + frontend

**描述**: 2025 IGA 最大趋势（Omada/Saviynt 主推）：LLM 生成审查建议 — "该用户 90 天未使用此权限，建议回收"、"该角色与岗位职责不符"。GGID 已有 MCP server（13 个 LLM 管理 tools）+ audit 使用数据，天然具备数据基础。

**业务价值**: MEDIUM-HIGH（差异化卖点，审查员疲劳是 IGA 落地最大障碍）
**实现难度**: Medium
- 实现路径：
  1. 审查列表 API 增加 recommend 字段：规则先行（last_used>90d → recommend_revoke），LLM 增强（可选，接 MCP）
  2. 前端审查页显示建议标签 + 一键接受
  3. 纯规则版先行（无 LLM 依赖），LLM 版作 v2

**兼容性**: audit 事件有 last_used 数据；MCP 已通 LLM

---

## NHI 纳入访问审查范围 (2026-07-17 第4小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: 现有 campaign scope 仅覆盖人类用户。2025 趋势要求 agent/service account/API key 同样定期审查（与 AAM backlog 项协同）。campaign scope 增加 "agents"、"service_accounts" 类型。

**业务价值**: MEDIUM-HIGH | **实现难度**: Low-Medium（scope 扩展 + agent 数据源接入）

---

## 持续访问评估 CAE (Continuous Access Evaluation) (2026-07-17 第5小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + IAMExpert

**描述**: NIST SP 800-207 零信任核心原则：访问授权不是一次性登录事件，而是持续评估。当会话期间上下文恶化（IP 突变、设备失信任、ITDR 检测命中），访问应立即撤销或降级。Microsoft CAE 已成为事实标准。

**GGID 现状审计**: 数据源全部齐备 — device trust（register/trusted/bind 真实端点）、adaptive MFA 风险引擎、ITDR 检测引擎（刚端到端打通）、session 管理。缺的是**联动层**：ITDR critical detection 产生后，没有任何机制自动撤销该用户会话。

**业务价值**: HIGH
- 零信任是政企采购的 checklist 必备项
- GGID 各组件已建，联动即可讲故事："检测→撤销 <1s"
- 与 ITDR 引擎协同形成完整 ITDR+ZT 叙事

**实现难度**: Medium
- 实现路径：
  1. ITDR playbook 增加内置动作 `revoke_sessions`（v2 接口已预留）：critical/high detection → auth 服务撤销用户全部会话（Redis session 删除 + refresh token 吊销）
  2. Gateway JWT 验证增加"会话存活"检查（Redis 快速查，~0.5ms）或被撤销 jti 黑名单
  3. 会话绑定设备状态变化时触发重评估（device untrust → 降权只读）
  4. 审计链：每次自动撤销写 audit 事件 + ITDR detection.detail.execution

**兼容性**: 全部复用现有组件；ITDR playbook v2 接口设计已留

---

## 零信任统一策略决策点 PDP (2026-07-17 第5小时研究) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: NIST ZTA 参考架构核心：统一 PDP/PEP。GGID 现有 device trust、session risk、ITDR detections、ABAC 属性分散在各服务，策略评估时无法组合使用（如"仅当设备可信 AND 无未处理 critical 检测 AND 工作时间 才允许导出报表"）。

**业务价值**: MEDIUM-HIGH | **实现难度**: Medium-High
- 实现路径：
  1. policy 服务评估输入扩展：注入 device_trust_score（auth API）+ open_detections_count（audit itdr API）+ session_risk
  2. ABAC 条件 DSL 增加内置属性：$device.trusted, $itdr.critical_open, $session.risk_score
  3. 评估缓存（5s TTL）避免每次跨服务调用
- 建议先做 3 个内置属性验证价值，再开放自定义

---

## Zero Trust 姿态评估真实化 (2026-07-17 第5小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + frontend

**描述**: useZeroTrustPosture SDK hook 是 setTimeout 假数据（属 frontend P0 清理范围）。姿态评分应来自真实数据源：设备信任覆盖率（trusted/total devices）、MFA 覆盖率、24h 未处理 critical detections 数、会话绑定率。

**业务价值**: MEDIUM（管理态势感知真实化）| **实现难度**: Low-Medium
- 实现路径：auth 或 audit 新增 GET /api/v1/zt/posture 聚合四项指标 → hook 接真端点（纳入 frontend 清理批次）

---

## K8s Workload Identity 接入（TokenReview → GGID Token） (2026-07-17 第6小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: 云原生标准场景：K8s Pod 内的 workload 需要访问受 GGID 保护的 API。标准模式（GCP/AWS/Azure 一致）：Pod 挂载 ServiceAccount token → 应用用它向 IdP 换访问令牌。GGID 需要：K8s TokenReview API 验证 SA token → 按命名空间/SA 映射策略签发 GGID access token。SPIFFE JWT-SVID 采用同构流程（OIDC 发现 + JWT 验证）。

**市场背景**: CIEM 市场 $3.9B(2025)→$46.3B(2034)，CAGR 31.7%，云安全增长最快细分。GCP 2025 推 managed workload identity（SPIFFE 池化）。

**GGID 现状**: operator 已有（instance/tenant CRD + provisioning controller，Round 87 部署）；RFC 8693 token exchange 已在 backlog（可复用其 grant 机制）；无 TokenReview 支持。

**业务价值**: MEDIUM-HIGH（k8s 部署的客户的刚需，operator 故事补全）
**实现难度**: Medium
- 实现路径：
  1. auth 服务新增 POST /api/v1/auth/workload/token：入参 k8s_sa_token → TokenReview（in-cluster 或 kubeconfig）→ 验证 namespace/SA/audience → 映射 GGID service identity → 签发 access token（短 TTL）
  2. 配置：tenant 级 workload identity pool（允许哪些 cluster/namespace/SA）
  3. SPIFFE 兼容：接受 spiffe:// trust domain 的 JWT-SVID（验签 + trust bundle 配置）
  4. NHI 联动：workload token 的 subject 注册为 NHI（与 AAM backlog 协同）

**兼容性**: 复用 JWT 签发管线；TokenReview 仅出站 HTTPS 调用

---

## CIEM 权限使用分析（Unused/Excessive Entitlements） (2026-07-17 第6小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: CIEM 核心能力：交叉分析"授予的权限"（policy 服务角色/权限数据）与"实际使用"（audit 事件资源访问记录），产出：90 天未使用权限清单、过度授权角色（使用率<10%）、权限右移建议（right-sizing）。与 IGA GenAI 审查建议（审查时点动作）互补：CIEM 是持续分析 dashboard。

**业务价值**: MEDIUM-HIGH（CIEM 独立产品线能力，GGID 数据全有只差计算）
**实现难度**: Medium
- 实现路径：
  1. audit 新增分析 job（每日）：role_assignments × audit_events(resource_type/action) 交叉 → entitlement_usage 表
  2. GET /api/v1/audit/ciem/insights：unused_permissions / excessive_roles / right_sizing_suggestions
  3. console entitlement-review 页面接真数据（当前属假数据清理批次）
  4. 与 ITDR 联动：excessive role + 异常使用 → 风险信号

---

## Operator 扩展：GGIDClient/Application CRD (2026-07-17 第6小时研究) - Priority: P3 - Status: Proposed - Suggested: backend

**描述**: operator 现有 instance/tenant CRD。扩展 GGIDClient CRD 实现 OAuth client 的 K8s 声明式管理（GitOps 场景：应用部署清单中直接声明 redirect_uris/scopes，operator 自动注册 client 并写回 secret）。ArgoCD/Flux 用户的标准期望。

**业务价值**: MEDIUM | **实现难度**: Low-Medium（controller 模式已就绪，新增一种 CRD + reconcile 调 identity admin API）

---

## Per-User 行为基线学习（轻量 UEBA） (2026-07-17 第7小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**描述**: 行业标准 UEBA（Microsoft Sentinel 为参考）核心是"持续学习每个用户的正常模式"。GGID 现状：risk_engine 是全局阈值启发式（velocity>20→+0.4），对异常但合法的用户误报高，对低慢攻击漏报。

**GGID 独特优势**：audit 事件流是完美的 UEBA 训练数据；刚建成的 ITDR 检测引擎是完美投递机制（新增 baseline_deviation 规则类型即可）。

**业务价值**: MEDIUM-HIGH
- 从"阈值告警"升级到"个性化异常检测"，检测精度质变
- 低慢攻击（low-and-slow）检测能力，阈值法盲区
- 无 ML 框架依赖：纯 Go 统计（z-score / modified z-score / 直方图）

**实现难度**: Medium
- 实现路径：
  1. audit 新增 profile builder（每日 job，30 天滑窗）：per-user 登录时段直方图、常用 IP/ASN 集合、资源访问频率分布、典型 UA
  2. 存储：user_behavior_profiles 表（tenant+user+profile JSONB + updated_at）
  3. ITDR 新规则 baseline_deviation：事件与 profile 偏差超 3σ → detection（证据含"通常 9-18 点北京登录，本次 3:15 未知 IP"）
  4. 冷启动：新用户前 50 事件只学习不评估
  5. agent_behavior_handler 的 BehaviorBaseline 已有雏形，可统一到此模型

**兼容性**: ITDR 引擎规则接口直接扩展；复用 NATS 事件流

---

## Peer-Group 对比分析（内部威胁检测） (2026-07-17 第7小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: UEBA 第二支柱：用户与同部门/同角色群体规范对比。检测内部威胁经典场景：财务用户突然访问工程代码库、普通员工访问高管报表。GGID 有完整 org 树（LTREE）+ role 数据。

**业务价值**: MEDIUM | **实现难度**: Medium
- 实现路径：
  1. profile builder 扩展 peer group 聚合（department/role 级资源访问分布）
  2. 新规则 peer_deviation：用户访问资源类型在 peer group 中占比 <1% 且个人历史首次 → medium detection
  3. 与 baseline_deviation 组合加权

---

## LLM 检测解释（Detection Narrative） (2026-07-17 第7小时研究) - Priority: P3 - Status: Watch - Suggested: IAMExpert

**描述**: 2025 趋势：SOC 告警的 LLM 自然语言解释。ITDR detection 产生时调用 LLM 生成分析师友好的叙述（证据摘要+影响评估+建议处置）。GGID MCP 已有 LLM 通道。等 ITDR 引擎上线积累真实 detection 后评估。

---

## OIDC Federation 1.0 支持 (2026-07-17 第8小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: OIDC Federation 1.0 (RFC 9486) 允许 IdP/SP 间无元数据交换自动建立信任（信任链：组织根 CA → 实体 → 子实体）。多组织联邦（高校/政府跨域 SSO）场景的标准方案。GGID 已有 oidc-federation-config 页面和 handler，但需验证是否实现信任链解析（entity statement + authority hints + JWKS 自动发现）。

**GGID 现状审计**: SAML 实现质量高（XMLDSig 签名、assertion 解析、IdP/SP 双向、条件验证全有）。OIDC discovery 有。OIDC Federation handler 存在但深度未验证。这是一个真实的 SAML/OIDC 双协议 IdP（罕见的完整度）。

**业务价值**: MEDIUM-HIGH（高校/政府联邦 SSO 标配，GGID 已有 SAML 基础）| **实现难度**: Medium-High
- 实现路径：验证 trust chain resolution → 补 entity statement 签发/验证 → federation metadata 端点

---

## SAML SP Connector 增强 (2026-07-17 第8小时审计) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: GGID 作为 SP 对接外部 IdP（Okta/Entra/Azure AD）是企业 SSO 主场景。现有 /saml/acs 接收断言并解析（ParseAssertion + ValidateConditions + ExtractAttributes），但用户映射（nameID/email → GGID user）需验证是否有 JIT provisioning（首次 SSO 自动创建用户）。

**业务价值**: HIGH（企业 SSO POC 必演）| **实现难度**: Low-Medium

---

## IssueSAMLToken 命名误导修复 (2026-07-17 第8小时审计) - Priority: P3 - Status: Proposed - Suggested: backend

**描述**: `IssueSAMLToken` 实际签发的是 JWT（issueAccessToken with "saml" grant type），不是 SAML XML 断言。函数名误导。建议重命名 IssueTokenForSAMLUser 或在注释中明确。

---

## Zero Standing Privileges + JIT 权限提升 (2026-07-17 第9小时研究) - Priority: P1 - Status: Proposed - Suggested: backend

**市场背景**: CSA 2025 主推"杀死常驻权限"（Zero Standing Privileges）。现代 PAM 从"永久 admin"转向"需要时才提权"（JIT elevation），提权有时间限制（30min-4h）+ 审批流 + 全审计。CyberArk、Palo Alto XSIAM 全部转向此模式。

**GGID 现状审计**:
- 有 JIT elevation console 页面 + `/api/v1/policy/jit-elevation` 端点（需验证深度）
- 有 break-glass 紧急访问页面，但后端**只有 GET history（内存数组）**，无 activate/审批/到期撤销
- 有 jit-provisioning-config（identity 服务）97 行 handler
- 缺核心：提权请求 → 审批 → 临时角色绑定 → 到期自动解绑 → 审计闭环

**业务价值**: HIGH
- CSA + Gartner 2025 PAM 第一推荐：Zero Standing
- 政企合规必查项（等保要求最小权限 + 审计）
- break-glass 是安全运维最后防线，当前不完整

**实现难度**: Medium
- 实现路径：
  1. policy 服务：POST /api/v1/policies/jit/request — 用户请求临时权限（role + scope + duration + reason）
  2. 审批流：role.assign 临时绑定（expires_at 列）→ 到期自动解绑（goroutine 或 cron 清理）
  3. break-glass activate：POST /api/v1/auth/break-glass/activate — 紧急提权（双人审批或自动 + 强审计 + 告警 webhook）
  4. 全审计链：每次 JIT 提权写 audit 事件（谁、什么权限、多久、为什么）
  5. 迁移：现有 admin 角色从"永久"改为"JIT 默认 8h 续期"（渐进式）

**兼容性**: 复用 role assignment（加 expires_at）；CAE 联动（JIT 到期后 session 自动降权）

---

## PAM Credential Vault 集成 (2026-07-17 第9小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: GGID 有 credential-vault 和 credential-vault-management 两个 console 页面。PAM 标准能力：特权密码/密钥集中保管 → 按需 checkout → 使用完自动轮换 → 审计。与 JIT elevation 互补（JIT 管权限，Vault 管密码）。

**业务价值**: MEDIUM-HIGH | **实现难度**: Medium-High（需要密码代理或 API 集成）
- 简化版：GGID 作为 credential broker（存储/checkout/轮换 API），不做密码代理注入
