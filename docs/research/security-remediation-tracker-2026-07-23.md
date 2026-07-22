# 安全修复追踪表

> 基于 `iam-security-audit-2026-07-23.md` 审计报告
> 创建日期：2026-07-23
> 最后更新：2026-07-23

---

## 总览

| 严重度 | 总数 | 已分配 | 进行中 | 已完成 | 未分配 |
|--------|------|--------|--------|--------|--------|
| P1（安全阻塞） | 7 | 7 | 0 | 7 | 0 |
| P2（企业交付前） | 14 | 8 | 0 | 3 | 3 |
| P3（后续迭代） | 7 | 0 | 0 | 0 | 7 |
| **合计** | **28** | **15** | **0** | **10** | **10** |

> **审计范围说明**：审计报告覆盖 7 个维度共 29 个薄弱点（A1/D1 为同一问题跨维度重复）+ 32 个已落实控制项 = 61 个检查点。guardian 报告中"40+ 发现项"为近似值。本追踪表覆盖全部 28 个待修复行动项（B2/B3 从 P2 提升至 P1 合入 S7）。

---

## 发现 ID 覆盖核对

全部 29 个原始发现 ID 均已追踪：

| 维度 | ID | 严重度 | 追踪编号 |
|------|----|--------|---------|
| 认证 | A1 | P1 | S4 (= D1) |
| 认证 | A2 | P1 | S2 |
| 认证 | A3 | P1 | S6 |
| 认证 | A4 | P2 | P2-1 |
| 认证 | A5 | P2 | P2-2 |
| 认证 | A6 | P3 | P3-1 |
| 授权 | B1 | P1 | S1 |
| 授权 | B2 | P1 | S7 |
| 授权 | B3 | P1 | S7 |
| 授权 | B4 | P3 | P3-6 |
| 会话 | C1 | P1 | S5 |
| 会话 | C2 | P2 | P2-4 |
| 会话 | C3 | P3 | P3-2 |
| 会话 | C4 | P2 | P2-5 |
| 加密 | D1 | P1 | S4 (= A1) |
| 加密 | D2 | P2 | P2-6 |
| 加密 | D3 | P2 | P2-7 |
| 加密 | D4 | P3 | P3-3 |
| 输入 | E1 | P2 | P2-8 |
| 输入 | E2 | P2 | P2-9 |
| 输入 | E3 | P2 | P2-10 |
| 输入 | E4 | P3 | P3-4 |
| 审计 | F1 | P1 | S3 |
| 审计 | F2 | P3 | P3-5 |
| 审计 | F3 | P2 | P2-11 |
| 架构 | G1 | P2 | P2-12 |
| 架构 | G2 | P2 | P2-13 |
| 架构 | G3 | P2 | P2-14 |
| 架构 | G4 | P3 | P3-7 |
| 架构 | G5 | P2 | P2-15 |

---

## P1 项（安全阻塞，立即修复）

| # | 维度 | 问题 | 负责人 | 状态 | 预估 |
|---|------|------|--------|------|------|
| S1 | 授权 | B1: platform:admin scope 签发侧无租户过滤 | guardian | **完成** (787270449) | 2h |
| S2 | 认证 | A2: GGID_INTERNAL_SECRET dev fallback | guardian | **完成** (787270449) | 30min |
| S3 | 审计 | F1: HMAC secret 缺失时静默禁用 | guardian | **完成** (787270449) | 30min |
| S4 | 加密 | A1/D1: LDAP InsecureSkipVerify=true | guardian | **完成** (787270449) | 1h |
| S5 | 会话 | C1: revokedTokens/stateStore sync.Map 内存态 | backend | **完成** (83809081e) | 4h |
| S6 | 认证 | A3: PASSWORD_PEPPER 未强制配置 | backend | **完成** (83809081e) | 15min |
| S7 | 授权 | B2+B3: CORS fallback `*` + OAuth scope 未交集 | backend | **完成** (83809081e) | 1h |

> backend 之前确认过 S5/S6/S7（上一会话），当前会话已发送确认请求，等待回复。

---

## P2 项（企业交付前修复）

### 已认领

| # | 问题 | 负责人 | 状态 |
|---|------|--------|------|
| P2-11 | F3: 告警规则未配置（tamper incident 无通知渠道） | arch_pm | 已认领 |
| P2-12 | G1: 无服务间 mTLS | arch_pm | 已认领 |
| P2-13 | G2: DB 凭据共享（无 per-service 隔离） | arch_pm | 已认领 |
| P2-14 | G3: 无密钥管理服务集成（无 Vault/KMS） | arch_pm | 已认领 |
| P2-15 | G5: 依赖供应链安全缺失（无 govulncheck/SBOM） | arch_pm | 已认领 |

### 未分配

| # | 问题 | 建议负责人 | 维度 |
|---|------|-----------|------|
| P2-1 | A4: TOTP secret 明文存储 | guardian | **完成** (f1920ce55) | 认证 |
| P2-2 | A5: WebAuthn challenge 内存存储，多实例不可共享 | backend | 认证 |
| P2-4 | C2: DPoP token cache 内存态 sync.Map | backend | 会话 |
| P2-5 | C4: Refresh token TOCTOU 竞态 | backend | 会话 |
| P2-6 | D2: 审计 HMAC secret 无版本管理 | guardian | **完成** (63ed9054f) | 加密 |
| P2-7 | D3: 审计 HMAC canonicalization 碰撞风险 | guardian | **完成** (63ed9054f) | 加密 |
| P2-8 | E1: fmt.Sprintf 拼接 SQL 列名 | backend | 输入 |
| P2-9 | E2: map_repo.go 表名 fmt.Sprintf | backend | 输入 |
| P2-10 | E3: 20 处错误吞噬 `_, _ = pool.Exec` | backend | 输入 |

> B2/B3 (CORS + scope) 已提升至 P1 合入 S7。

---

## P3 项（后续迭代）

| # | 问题 | 建议负责人 | 维度 |
|---|------|-----------|------|
| P3-1 | A6: JWT key 仅 2048-bit | guardian | 认证 |
| P3-2 | C3: Session cleanup 依赖定时扫描 | backend | 会话 |
| P3-3 | D4: JWT key 轮换 grace period 过长 | guardian | 加密 |
| P3-4 | E4: OAuth state 参数仅内存校验 | backend | 输入 |
| P3-5 | F2: 审计事件大量 log.Printf（应迁移 slog） | backend | 审计 |
| P3-6 | B4: consent cascade mock 路径 | backend | 授权 |
| P3-7 | G4: 无 Helm/版本化发布 | arch_pm | 架构 |

---

## 系统性风险总结

1. **多实例状态管理**（S5/P2-2/P2-4/P3-4）: 4 处 sync.Map 存储安全状态，K8s 多副本下失效。S5 解决核心项，其余跟进。
2. **密钥管理薄弱**（S2/S3/S4/P2-6/P2-7/P2-14）: dev fallback、未强制配置、无轮换、无 KMS。需生产部署清单强制检查。
3. **签发侧信任链不完整**（S1）: platform:admin scope 不验证角色租户归属，是 RBAC 所有修复的上游根因。

---

## 修复批次

| 批次 | 范围 | 时间 | 状态 |
|------|------|------|------|
| 批次 1 | P1 全部 7 项 | 立即 | 4/7 完成，3 项待 backend 确认 |
| 批次 2 | P2 安全关键（P2-1/4/5/6/7） | P1 完成后 | 待启动 |
| 批次 3 | P2 架构运维（P2-8~15） | 与批次 2 并行 | arch_pm 已认领 5 项 |
| 批次 4 | P3 全部 7 项 | 下一迭代 | 排队 |
