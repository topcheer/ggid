# 2026 IAM Trends GAP Analysis — GGID vs Industry

> 深度研究 | arch_pm | 2026-07-18 | 来源: cybersecit.net IAM trends 2026 + Gartner + Forrester

## 方法论
1. 提取 2026 年 IAM 行业 10 大趋势
2. 逐一对照 GGID 代码库检查实现状态（grep + 代码行数分析）
3. 评估优先级（enterprise sales blocker / nice-to-have）
4. 输出可执行 backlog（合并已有 + 新增）

## GAP 矩阵

### 1. Non-Human Identity (NHI) 管理 — **PARTIAL**
**行业要求**: 统一管理 service accounts、API tokens、machine credentials 的全生命周期。

**GGID 现状**:
- `nhi_inventory_handler.go` (64行, 3 endpoints) — 基础 inventory
- `nhi_lifecycle.go` — 生命周期管理
- `secret_history_handler.go` — secret 历史
- OAuth client_credentials grant — 已支持

**GAP**:
- ❌ 无 NHI 自动发现（扫描基础设施中的 unmanaged service accounts）
- ❌ 无 secret rotation policy enforcement（有轮换引擎但未对 NHI 强制）
- ❌ 无 NHI 权限审计（NHI 有什么权限？哪些是 orphan？）
- ❌ 无 NHI risk scoring（NHI 的行为分析和异常检测）

**优先级**: P1 — enterprise 客户必问

### 2. Privilege Creep Detection — **PARTIAL**
**行业要求**: 自动检测权限积累（用户调岗后旧权限未清理）。

**GGID 现状**:
- `entitlement_review_handler.go` — entitlement review 存在
- `directory_reconcile_handler.go` — 目录对账
- Access review campaigns — 已实现

**GAP**:
- ❌ 无自动权限基线（用户当前权限 vs 角色标准权限的 diff）
- ❌ 无权限积累告警（同一用户 30 天内权限持续增长 → 告警）
- ⚠️ Access review 是手动触发，无自动定期执行

**优先级**: P0 — 合规审计必须项

### 3. Segregation of Duties (SoD) — **IMPLEMENTED (basic)**
**GGID 现状**: `sod_violation_handler.go` + 测试

**GAP**:
- ⚠️ 无 SoD 规则编辑器（前端）
- ⚠️ 无 SoD 预检（用户被分配角色前检查 SoD 冲突）
- ⚠️ 无 SoD 例外管理（业务需要时允许例外 + 审批）

**优先级**: P1

### 4. IAM-ITDR Metrics (MTTD/MTTR) — **MISSING**
**行业要求**: 可测量的 IAM KPI — 身份威胁的平均检测时间和响应时间。

**GGID 现状**: ITDR 检测规则完善（15 MITRE rules），但无 metrics 追踪。

**GAP**:
- ❌ 无 MTTD 计算（从事件发生到检测的时间差）
- ❌ 无 MTTR 计算（从检测到响应/解决的时间差）
- ❌ 无 IAM 成熟度评分仪表盘
- ❌ 无 identity coverage metric（多少用户受 MFA/ITDR/CAE 保护）

**优先级**: P1 — CISO 汇报需要

### 5. Outcome-Driven IAM Metrics — **MISSING**
**行业要求**: 从 feature checklist 转向 outcome metrics。

**GAP**:
- ❌ 无 "phishing-resistant MFA coverage %" 指标
- ❌ 无 "orphaned service account count" 指标
- ❌ 无 "access review completion rate" 指标
- ❌ 无 "identity incident trend" 趋势图

**优先级**: P2

### 6. AI Agent Identity Governance — **PARTIAL**
**行业要求**: 管理 autonomous AI agents 的身份、权限和审计。

**GGID 现状**:
- `agent_lifecycle_handler.go` — agent 生命周期
- `agent_review_handler.go` — agent review
- AI agent identity 研究（34KB doc）

**GAP**:
- ⚠️ 无 agent 权限 scoping（agent 只能访问特定 API）
- ⚠️ 无 agent 行为审计（agent 调用了什么 API、什么时候）
- ⚠️ 无 agent rate limiting（防止 agent 失控）

**优先级**: P1 — Google A2A / Microsoft Agent ID 竞争压力

### 7. Cloud Identity Federation — **PARTIAL**
**GGID 现状**: SAML IdP + SP, OIDC provider, federation entities table

**GAP**:
- ⚠️ 无 AWS IAM Identity Center federation（只有 SAML 基础）
- ⚠️ 无 Azure AD app roles mapping
- ⚠️ 无 GCP workforce federation

**优先级**: P1 — enterprise cloud 必需

### 8. Continuous Compliance Monitoring — **PARTIAL**
**GGID 现状**: compliance report handler, audit chain, NIS2/CRA dashboard

**GAP**:
- ⚠️ 无实时控制监控（CCM — continuous control monitoring）
- ⚠️ 无自动 evidence collection（手动审计）
- ⚠️ 无 compliance gap alerting

**优先级**: P1

### 9. Identity-First Security Architecture — **IMPLEMENTED (strong)**
**GGID 优势**: PDP + CAE + URE + device posture + ReBAC — 业界领先

**GAP**:
- ✅ 基本完整。可优化 PDP 缓存策略和 UEBA 精度。

### 10. Privileged Access Management (PAM) — **PARTIAL**
**GGID 现状**: JIT elevation + break-glass + delegated admin

**GAP**:
- ❌ 无 session recording（PAM session recording for compliance）
- ❌ 无 command logging（privileged session 中的每个命令记录）
- ❌ 无 credential vaulting for privileged accounts（有 vault 但未对 PAM 特化）

**优先级**: P2

## 新增 Backlog（合并已有）

| ID | Title | Owner | Priority | Est | Note |
|----|-------|-------|----------|-----|------|
| KB-270 | **NHI 自动发现引擎**（扫描 PG/NATS/K8s 中的 service accounts） | backend | P1 | 5d | 合并 KB-146 |
| KB-271 | **NHI risk scoring + behavior analytics** | security | P1 | 4d | 新 |
| KB-272 | **Privilege creep detection**（权限基线 + diff + 告警） | backend | P0 | 4d | 新 |
| KB-273 | **自动 access review scheduling**（季度/年度自动触发） | backend | P1 | 3d | 增强 KB-067 |
| KB-274 | **IAM-ITDR metrics dashboard**（MTTD/MTTR/coverage trends） | frontend | P1 | 3d | 新 |
| KB-275 | **Identity coverage scorecard**（MFA%/ITDR%/CAE%/passkey%） | frontend | P1 | 2d | 新 |
| KB-276 | **SoD 规则编辑器 + 预检**（分配前检查冲突） | frontend+backend | P1 | 4d | 增强 |
| KB-277 | **SoD 例外管理 + 审批流** | backend | P2 | 3d | 新 |
| KB-278 | **Agent 权限 scoping + 行为审计** | backend | P1 | 5d | 合并 KB-234~237 |
| KB-279 | **PAM session recording**（privileged session 录制） | backend | P2 | 8d | 新 |
| KB-280 | **Continuous compliance monitoring (CCM)** | backend | P1 | 5d | 增强 |
| KB-281 | **Cloud federation — AWS IAM Identity Center** | backend | P1 | 4d | 合并 KB-049~053 |

## 优先级排序建议（enterprise sales impact）
1. KB-272 privilege creep detection (P0) — 合规审计必须
2. KB-270 NHI auto-discovery (P1) — enterprise #1 安全问
3. KB-278 agent governance (P1) — AI agent 竞争
4. KB-281 cloud federation AWS (P1) — cloud 必需
5. KB-274 IAM-ITDR metrics (P1) — CISO 汇报
6. KB-280 CCM (P1) — 合规自动化
7. KB-276 SoD editor (P1) — 治理基础
8. KB-273 auto access review (P1) — 合规
9. KB-271 NHI risk scoring (P1) — 安全深化
10. KB-275 identity scorecard (P1) — 运营可视化
