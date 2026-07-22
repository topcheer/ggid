# GGID v2.0 产品路线图

> 创建日期：2026-07-23
> 状态：实施中
> 驱动方式：定时任务协调 + 团队分工推进

---

## 当前基线

- v1.0 基线已达成（P0×3 + P1×4 全部完成）
- 安全审计基线已建立（30 项发现，P1 进行中）
- 10 个微服务 / 375+ handler / 12 语言 SDK / ERP Demo 8 语言
- 9 个定时巡航任务覆盖 8 个视角

---

## 阶段一：可销售闭环（目标：2-3 周）

让 GGID 从"技术完备"变成"客户可以上手使用的产品"。

### R1-01: B2B 自助 Onboarding 流程
- **状态**: 进行中
- **认领**: arch_pm（流程设计+子域名路由 ✅）+ frontend（UI ✅）+ backend（API 编排，待确认）
- **进度**:
  - frontend: `/onboarding` 5 步向导已完成（连接 self-register API）
  - arch_pm: 子域名→tenant 解析设计中
  - backend: API 编排待认领
- **目标**: 访客 → 注册租户 → 自动配置 → 首个 admin 引导 → 开始使用
- **范围**:
  - 租户注册页（公司名 + admin 信息 + 邮箱验证）
  - 自动创建 tenant + 默认角色 + OAuth client
  - Bootstrap 向导（品牌化 / SCIM / 密码策略引导配置）
  - 子域名路由（tenant.ggid.iot2.win）
- **API 依赖**: org/CreateTenant + identity/CreateUser + policy/SeedRoles
- **验收**: 完整 onboarding 5 分钟内完成，新租户可登录 Console

### R1-02: Social Login 连接器
- **目标**: Google / GitHub / 企业微信 / 飞书 第三方登录
- **范围**:
  - IdP 配置 UI（identity 服务 idpconfig 已有后端）
  - OAuth 连接器实现（Google + GitHub 优先）
  - 登录页 Social 按钮 + 回调处理
  - JIT 用户 provisioning（已有一个 handler）
- **API 依赖**: identity/idpconfig + auth/login callback
- **负责人**: backend（连接器实现）+ frontend（UI）
- **验收**: Google 登录端到端可用

### R1-03: 组织树管理 UI
- **状态**: 前端已完成，待 backend API 验证
- **认领**: frontend ✅ + backend（API 验证，待确认）
- **进度**: `/organizations/tree` 760 行完整实现（树形+拖拽+CRUD+搜索+右键菜单+成员计数）
- **目标**: Console 中可视化组织树 + 拖拽移动 + 成员分配
- **验收**: 组织树 CRUD + 拖拽 + 成员分配端到端

### R1-04: Console 信息架构重构
- **状态**: 进行中（~60%）
- **认领**: frontend ✅ + QA ✅
- **进度**: Ctrl+K CommandPalette 已集成（40+ 命令），三套角色导航 requiredScope 过滤已有，Dashboard 首页优化中
- **目标**: 当前 125 个页面按角色分 dashboard，减少认知负荷
- **验收**: 3 种角色登录后看到不同 dashboard，关键操作 ≤3 次点击

---

## 阶段二：差异化壁垒（目标：3-6 周）

把 GGID 比 Auth0/Okta 多的能力变成可展示的卖点。

### R2-01: ITDR 产品化
- **目标**: 威胁检测仪表盘 + 自动响应 + 告警通知
- **范围**:
  - 安全态势 dashboard（threat_heatmap + kill_chain handler 已有）
  - 检测规则配置 UI（anomaly_detect handler 已有）
  - 自动响应 playbooks（itdr playbooks handler 已有）
  - 告警通知渠道（Slack/Email/Webhook — F3 修复后）
  - Incident 时间线（incident_timeline handler 已有）
- **负责人**: guardian（安全逻辑）+ frontend（dashboard UI）
- **验收**: 模拟攻击 → 自动检测 → 生成 incident → 发送通知

### R2-02: 合规审计包一键导出
- **目标**: SOC2 / GDPR / HIPAA 证据包生成
- **范围**:
  - 合规框架配置（compliance_config handler 已有）
  - 证据收集器（evidence_audit_trail + evidence_integrity handler 已有）
  - PDF/CSV 导出（report handler 已有）
  - 审计日志筛选 + 时间范围
  - 合规评分 auto-score（auto_score handler 已有）
- **负责人**: backend（导出逻辑）+ frontend（配置 UI）
- **验收**: 选择 SOC2 框架 + 时间范围 → 下载包含审计证据的合规报告

### R2-03: Identity Governance 完整流程
- **目标**: Joiner-Mover-Leaver 全自动化
- **范围**:
  - Joiner：创建用户 → 自动分配角色/组织 → MFA 引导
  - Mover：组织变更 → 权限重算 → access review 触发
  - Leaver：禁用账号 → 回收会话 → 归档审计
  - Access Review Campaign 审批工作流（campaign_results handler 已有）
  - Standing Access 报告（standing_access handler 已有）
- **负责人**: backend（流程编排）+ arch_pm（流程设计）
- **验收**: 模拟 joiner-mover-leaver 全链路，权限自动调整

### R2-04: 零信任 Posture 评分产品化
- **目标**: 可视化安全态势 + 改进建议 + benchmark
- **范围**:
  - Posture 评分计算（zt_posture handler 已有）
  - 评分维度可视化（设备/身份/网络/数据/工作负载）
  - 对标 NIST 800-207 框架
  - 改进建议引擎
  - 历史趋势
- **负责人**: guardian（评分逻辑）+ frontend（可视化）
- **验收**: Dashboard 展示当前 posture 评分 + 待改进项 + 历史趋势

---

## 阶段三：规模化和生态（目标：6-12 周）

### R3-01: SDK 正式发布
- **目标**: npm / pypi / maven / go mod / crates.io 发布管道
- **范围**:
  - 功能对齐（所有 SDK 覆盖 OAuth + MFA + Token Management）
  - 版本化 + changelog
  - CI 发布管道
  - SDK 文档站
- **负责人**: researcher

### R3-02: Helm Chart + 运维 Runbook
- **目标**: 一键部署 + 回滚 + 扩缩容
- **范围**:
  - 版本化 Helm Chart
  - values.yaml 分层（dev/staging/prod）
  - 回滚 runbook
  - 扩缩容指南
- **负责人**: arch_pm

### R3-03: 多区域高可用
- **目标**: 读副本 + 跨区域灾备
- **范围**:
  - PostgreSQL 流复制
  - Redis 哨兵/集群
  - 跨区域 DNS 故障转移
  - 数据同步策略
- **负责人**: arch_pm

### R3-04: AI Agent 身份集成
- **目标**: GGID 作为 AI Agent 的身份提供者
- **范围**:
  - MCP 协议身份层（services/mcp 已有基础）
  - Token scope 细粒度授权 for Agent
  - Agent 身份审计
  - Agent-to-Agent 委托（token-exchange 已有）
- **负责人**: backend

---

## 依赖关系

```
R1-01 (Onboarding) ─┬─→ R1-02 (Social Login)
                    ├─→ R1-04 (IA 重构)
                    └─→ R2-03 (Governance)

R1-03 (Org Tree) ────→ R2-03 (Governance)

R2-01 (ITDR) ───────→ R2-04 (Zero Trust)
R2-02 (Compliance) ─→ R2-04 (Zero Trust)

R3-01 (SDK) ────────→ R3-04 (AI Agent)
R3-02 (Helm) ───────→ R3-03 (HA)
```

---

## 协调机制

- **协调者 cron**：每 2 小时检查进度、验收完成项、推进阻塞
- **阶段门控**：阶段一全部完成后解锁阶段二，以此类推
- **团队同步**：每周一次进度广播
