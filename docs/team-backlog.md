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
