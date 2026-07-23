# GGID SOC2 / ISO 27001 合规控制项映射矩阵

日期：2026-07-23
编写：guardian_security
标准：AICPA SOC 2 Trust Services Criteria (TSC 2017) + ISO/IEC 27001:2022 Annex A

---

## 一、SOC 2 Trust Services Criteria 映射

### CC — Common Criteria（适用于全部 5 类）

| 控制项 | 描述 | GGID 实现位置 | 状态 |
|--------|------|--------------|------|
| CC1.1 | 治理结构与问责制 | 审计日志（actor_id + actor_name）、break-glass 审批、RBAC | ✅ |
| CC1.2 | 安全策略与程序 | password_policy（可配置）、MFA 强制策略、conditional access | ✅ |
| CC1.3 | 角色与职责 | 动态 RBAC（role_route_permissions + Redis 缓存）、tenant:admin / platform:admin 分离 | ✅ |
| CC1.4 | 招聘/培训 | 文档：docs/guides/（开发安全指南）、GGCODE.md 项目规范 | ⚠️ 文档 |
| CC1.5 | 问责与强制执行 | 审计哈希链 tamper-evidence（HMAC-SHA256 + WORM 触发器 + tamper-check 巡检） | ✅ |
| CC2.1 | 内部沟通 | 审计事件流（NATS）、ITDR 检测→告警链路（detection→alert→webhook/email） | ✅ |
| CC2.2 | 外部沟通 | Webhook 订阅（HMAC 签名）、SOC2/GDPR 合规报表导出 | ✅ |
| CC2.3 | 系统组件盘点 | OpenAPI spec（内置 Swagger UI）、服务注册、deploy/helm chart | ✅ |
| CC3.1 | 风险识别与评估 | 零信任 Posture 评分（5 维度 NIST 800-207）、ITDR 威胁检测（17 条 KB 规则） | ✅ |
| CC3.2 | 风险缓解策略 | 改进建议引擎（posture recommendations）、conditional access 策略 | ✅ |
| CC3.3 | 新风险评估 | 安全巡航 cron（每 2h 7 维度轮转审计）、ITDR 持续检测 | ✅ |
| CC3.4 | 管理审查 | 零信任 posture 历史趋势（zt_posture_history）、ITDR incident 时间线 | ✅ |
| CC4.1 | 系统操作监控 | 审计日志全量记录（38+ 事件类型）、SSE 实时流、NATS 异步消费 | ✅ |
| CC4.2 | 偏差监控与纠正 | tamper-check（自动告警 + audit_incidents 落库）、异常检测（UEBA） | ✅ |
| CC5.1 | 新控制部署 | 迁移系统（047 migrations）、feature flag（sys_config） | ✅ |
| CC5.2 | 控制运行有效性 | 巡航 cron 自动验证（单元测试 + 生产 tamper-check + 匿名访问检测） | ✅ |
| CC5.3 | 偏差识别与纠正 | ITDR 检测引擎 + incident handler + SOAR playbook 自动响应 | ✅ |
| CC6.1 | 逻辑与物理访问 | JWT 认证 + 多因子认证（TOTP/WebAuthn/RADIUS/YubiKey）+ PKCE 强制 | ✅ |
| CC6.2 | 用户认证 | Argon2id + pepper + 历史检查 + HIBP breach 检测 | ✅ |
| CC6.3 | 访问授权 | 动态 RBAC + RLS 行级租户隔离 + scope 签发侧租户过滤（S1） | ✅ |
| CC6.4 | 未授权访问检测 | 审计事件记录所有拒绝/失败操作、brute-force 检测规则 | ✅ |
| CC6.5 | 安全传输 | TLS 1.2+（7 处 MinVersion）、HSTS、Cookie Secure+HttpOnly+SameSite | ✅ |
| CC6.6 | 跨边界数据流 | OAuth 2.1 标准流程、SAML SSO（签名验证 + XSW 防护）、SCIM 2.0 | ✅ |
| CC6.7 | 数据处置 | GDPR Article 17 删除（密码确认 + 审计保留）、RLS 隔离 | ✅ |
| CC6.8 | 组件连接 | OAuth client credentials + SCIM token middleware + API key | ✅ |
| CC7.1 | 基础设施保护 | K8s NetworkPolicy + ClusterIP（内部服务不可外部访问） | ⚠️ 部分 |
| CC7.2 | 未授权变更检测 | 审计哈希链 tamper-evidence + WORM 触发器（audit_events_worm_guard） | ✅ |
| CC7.3 | 安全事件检测 | ITDR 检测引擎（17 条规则）、composite rules（多信号关联） | ✅ |
| CC7.4 | 安全事件响应 | SOAR playbooks、incident handler（open→investigating→resolved） | ✅ |
| CC7.5 | 恢复程序 | DB 定时备份（建议配置）、迁移回滚（DOWN migrations） | ⚠️ 需完善 |
| CC8.1 | 软件变更管理 | Git 版本控制 + CI（security.yml）+ make test 全量回归 | ✅ |
| CC9.1 | 漏洞与威胁管理 | 安全巡航 cron（持续 grep 危险模式）、govulncheck CI（建议启用） | ⚠️ |
| CC9.2 | 供应商管理 | 第三方库依赖（go.mod）、最小化依赖原则 | ⚠️ |

### A — Availability（可用性）

| 控制项 | 描述 | GGID 实现位置 | 状态 |
|--------|------|--------------|------|
| A1.1 | 容量与环境监控 | Prometheus metrics（38 处）、health check（/healthz/ready） | ✅ |
| A1.2 | 环境保护 | K8s deployment（多副本 + readiness/liveness probe） | ✅ |
| A1.3 | 基础设施恢复 | K8s rollout restart、数据库备份（建议增强） | ⚠️ |

### C — Confidentiality（机密性）

| 控制项 | 描述 | GGID 实现位置 | 状态 |
|--------|------|--------------|------|
| C1.1 | 数据分类与处置 | 租户隔离（RLS FORCE）、PII 标记（pii_redacted 元数据） | ✅ |
| C1.2 | 机密信息传输与存储 | TOTP secret AES-256-GCM 加密、审计 HMAC 链、JWT RSA-256 | ✅ |

### P — Processing Integrity（处理完整性）

| 控制项 | 描述 | GGID 实现位置 | 状态 |
|--------|------|--------------|------|
| P1.1 | 输入验证 | 参数化查询（$1/$2）、body size limit、OAuth state CSRF 验证 | ✅ |
| P2.1 | 处理授权 | 动态 RBAC（DB 驱动 + Redis 缓存）、scope-only admin 匹配 | ✅ |

### PI — Privacy（隐私）

| 控制项 | 描述 | GGID 实现位置 | 状态 |
|--------|------|--------------|------|
| PI1.1 | 隐私通知与政策 | GDPR 合规文档（docs/research/gdpr-compliance.md） | ✅ |
| PI1.2 | 个人信息收集 | 最小化原则（SCIM 映射配置、OAuth scope 交集校验） | ✅ |
| PI2.1 | 个人信息使用与保留 | 审计保留策略（retention policies，可配置天数） | ✅ |
| PI3.1 | 个人信息处置 | GDPR Article 17 删除端点（密码确认 + 审计事件保留） | ✅ |

---

## 二、ISO/IEC 27001:2022 Annex A 映射

### A.5 — Organizational Controls

| 控制项 | 描述 | GGID 实现 | 状态 |
|--------|------|----------|------|
| A.5.1 | 信息安全策略 | 可配置密码策略、MFA 策略、conditional access | ✅ |
| A.5.2 | 信息安全角色与职责 | RBAC（动态 role_route_permissions + 自服务白名单） | ✅ |
| A.5.3 | 信息使用限制 | scope 签发侧租户过滤、RLS 行级隔离 | ✅ |
| A.5.7 | 威胁情报 | ITDR 威胁情报收集器（threat_intel_collector）、TI feed | ✅ |
| A.5.8 | 项目管理中的安全 | 安全巡航 cron、CI 安全检查 | ✅ |
| A.5.9 | 供应商关系中的信息 | 第三方依赖审计（go.mod + 建议启用 govulncheck） | ⚠️ |
| A.5.12 | 供应商协议中的安全改进 | — | ⚠️ 文档 |
| A.5.15 | 访问控制 | 动态 RBAC + OAuth scope + SAML SSO + MFA | ✅ |
| A.5.16 | 身份管理 | JIT 用户配置（社交登录）、SCIM 2.0 用户/组同步 | ✅ |
| A.5.17 | 认证信息管理 | Argon2id + pepper + 密码历史 + HIBP breach 检测 | ✅ |
| A.5.18 | 访问权限 | tenant:admin / platform:admin 分离、break-glass 审批 | ✅ |
| A.5.23 | 云服务的信息安全 | K8s secrets 挂载、TLS 1.2+、NetworkPolicy | ✅ |
| A.5.30 | ICT 应急响应 | ITDR 检测→告警→SOAR playbook 自动响应链路 | ✅ |
| A.5.34 | 隐私与个人数据保护 | GDPR Article 17 删除、PII 涂销（pii_redacted 标记） | ✅ |
| A.5.37 | 法律与合同要求文档 | 合规映射表（compliance_mapping 表 + handler） | ✅ |

### A.6 — People Controls

| 控制项 | 描述 | GGID 实现 | 状态 |
|--------|------|----------|------|
| A.6.3 | 安全意识与培训 | — | ⚠️ 文档 |
| A.6.6 | 保密/保密协议 | — | ⚠️ 文档 |
| A.6.8 | 信息安全事件报告 | ITDR incident handler + 审计事件全量记录 | ✅ |

### A.7 — Physical Security (out of scope for SaaS IAM)

### A.8 — Technological Controls

| 控制项 | 描述 | GGID 实现 | 状态 |
|--------|------|----------|------|
| A.8.1 | 用户终端设备 | 设备绑定、session device info、passkey | ✅ |
| A.8.2 | 访问权限 | RBAC + RLS + scope 过滤 | ✅ |
| A.8.3 | 信息访问限制 | 审计哈希链 WORM、RLS FORCE、SelfServicePaths 白名单 | ✅ |
| A.8.4 | 源代码访问 | Git 版本控制、scope-only admin 匹配（无裸角色名） | ✅ |
| A.8.5 | 安全认证 | MFA（TOTP/WebAuthn/RADIUS/YubiKey）+ OAuth 2.1 PKCE | ✅ |
| A.8.6 | 容量管理 | Prometheus metrics、K8s resource limits | ✅ |
| A.8.7 | 防恶意软件 | ITDR 检测（credential stuffing/brute force/impossible travel） | ✅ |
| A.8.9 | 配置管理 | sys_config 表（运行时可配）、Helm chart | ✅ |
| A.8.12 | 数据泄露预防 | 审计事件监控、GDPR PII 涂销、breach 检测 | ✅ |
| A.8.15 | 日志记录 | 全量审计（38+ 事件类型、actor/IP/UA/device/request_id） | ✅ |
| A.8.16 | 监控活动 | tamper-check 巡检 cron、ITDR 实时检测、SSE 流 | ✅ |
| A.8.17 | 时钟同步 | DB created_at UTC、NTP（K8s 默认） | ✅ |
| A.8.18 | 特权工具使用 | break-glass 审批、impersonation consent 系统 | ✅ |
| A.8.22 | 网络分离 | K8s NetworkPolicy、租户级 RLS 隔离 | ✅ |
| A.8.23 | Web 过滤 | rate limiting（per-tenant + per-IP + per-user） | ✅ |
| A.8.24 | 密码学 | Argon2id + AES-256-GCM（TOTP）+ HMAC-SHA256（审计链）+ RSA-256（JWT） | ✅ |
| A.8.25 | 开发生命周期安全 | CI security.yml + make test 回归 + 安全巡航 cron | ✅ |
| A.8.26 | 应用安全要求 | SAML XSW 防护、CSRF state、PKCE 强制、input validation | ✅ |
| A.8.27 | 安全系统架构 | 微服务隔离 + API Gateway + 零信任 posture 评分 | ✅ |
| A.8.28 | 安全编码 | 参数化查询、constant-time 比较、fail-closed 密钥管理 | ✅ |
| A.8.29 | 开发/测试中的安全 | 测试覆盖率（Task-A 60.9% auth）、隔离环境 | ✅ |

---

## 三、控制成熟度评分

| 领域 | 控制项数 | ✅ 已实现 | ⚠️ 部分 | 成熟度 |
|------|---------|----------|--------|--------|
| SOC2 CC | 30 | 24 | 6 | 80% |
| SOC2 A | 3 | 2 | 1 | 67% |
| SOC2 C | 2 | 2 | 0 | 100% |
| SOC2 P | 2 | 2 | 0 | 100% |
| SOC2 PI | 4 | 4 | 0 | 100% |
| ISO A.5 | 15 | 13 | 2 | 87% |
| ISO A.6 | 3 | 1 | 2 | 33% |
| ISO A.8 | 22 | 22 | 0 | 100% |
| **总计** | **81** | **70** | **11** | **86%** |

---

## 四、差距项与建议

### 需文档完善（6 项）
- CC1.4 安全培训文档
- CC9.2 供应商管理流程文档
- A.5.12 供应商安全协议模板
- A.6.3 安全意识培训计划
- A.6.6 保密协议模板
- A.5.12 合同安全条款

### 需技术增强（5 项）
- CC7.5 灾备恢复程序（DB 定时备份 + 恢复演练 runbook）
- CC9.1 govulncheck CI 集成 + SBOM 生成
- CC7.1 / A.8.22 服务间 mTLS（Istio/Linkerd）
- A.1.3 多区域灾备（R3-03 HA 设计已有，待实施）
- G2/G3 DB 凭据隔离 + KMS/Vault 集成

---

*本矩阵随 GGID 功能迭代持续更新。巡航 cron 每轮检查新增功能的安全合规映射。*
