# 产品化任务分配计划

> 基于 `docs/research/iam-product-deep-review.md` 验证后的行动计划
> 最后更新：2026-07-21

---

## 总览

| 优先级 | 总数 | 完成 | 状态 |
|--------|------|------|------|
| P0（产品可用性阻塞） | 3 | 3 | **全部完成** |
| P1（企业销售阻塞） | 4 | 4 | **全部完成** |
| P2（规模化） | 3 | 0 | 后续迭代 |

**里程碑：v1.0 基线已达成（P0），企业销售就绪（P1）。**

---

## P0 任务（产品可用性阻塞）— 全部完成 ✅

### Task-A: Auth Service 测试覆盖补强 ✅
- **负责人**: `ggcxf_backend` (backend)
- **验收**: `frontend_qa` — **验证通过** (覆盖率 60.9%, 目标 50%+)
- **Commit**: 575e895e4
- **交付**:
  - VerifyCredentials / Register / CheckBruteForce 全路径测试
  - MFA setup/verify 测试覆盖
  - 新增 4 个测试文件 (miniredis + 内存 mock, ta- 前缀)
  - 覆盖率从 16.9% 提升到 60.9%

### Task-B: 动态 RBAC 替代硬编码前缀 ✅
- **设计**: `arch_pm` — commit edea85e7c (`docs/design/adr-dynamic-rbac.md`)
- **实现**: `ggcxf_backend` — commit a0ab6ea19 + P0 回归修复 ee8132989 + ab762f527
- **交付**:
  - RBACResolver: 全量快照 Redis 60s TTL + 内存 warm-start + 硬编码 fallback
  - RequireAdminScope 签名不变，内部逻辑改为 DB 驱动
  - migration 042 种子数据 (19 个 admin 角色 × 11 个 API 前缀)
  - checkRouteScope 修复: 同时评估 claims.Roles (OAuth token scope 无 admin, 身份在 roles claim)
  - 新增角色插 DB 行即可获 admin 端点访问，无需改代码

### Task-C: 内存态功能持久化 ✅
- **负责人**: `ggcxf_backend` — commit a1010bb7a
- **交付**:
  - composite rules: repo 已存在但从未注入 → 已接线 (audit)
  - dashboard widgets: JSONB schema 修复 (audit)
  - alert rules: anomaly rules CRUD → PG 持久化 (audit)
  - SCIM groups: 新建 scim_groups 表，全 CRUD 落库 (identity)

---

## P1 任务（企业销售阻塞）— 全部完成 ✅

### Task-D: SAML XML-DSig 真实验签 ✅
- **负责人**: `guardian_security` — commit 511656c25
- **交付**:
  - pkg/saml ValidateSignature: 从"仅检查元素存在"升级为真实 RSA/ECDSA 验签
  - VerifySignedAssertion: 内容摘要绑定，篡改 NameID/属性即拒绝
  - oauth /saml/acs handler 接线真实验签，fail-closed (无证书或验签失败 403)
  - 7 项安全测试: 伪造签名、自签伪造、内容篡改、无签名全部拒绝

### Task-E: Refresh Token 家族轮换持久化 ✅
- **负责人**: `ggcxf_backend` — commit 6febb4d12
- **交付**:
  - oidc_refresh_tokens 新增 family_id 列 (migration 044)
  - PGTokenFamilyStore 注入（此前 RegisterTokenFamily 是死代码）
  - RFC 6749 §10.4 reuse detection: 重用 used/revoked token → 精确吊销整个 family
  - 4 项测试: family 建链、多次轮换、重用检测、legacy 兼容

### Task-F: Console Settings 页真实化 ✅
- **负责人**: `shen_frontend` — commit a7584a360
- **交付**:
  - 修复 3 个 404 子页面: branding-config、scim-provisioning、import-enhanced
  - 密码策略连接真实 API (GET/POST/PUT)
  - 品牌化: logo_url + custom_css 连接 tenant 配置

### Task-G: 审计 Tamper-Evidence 完善 ✅
- **负责人**: `guardian_security` — commit b22c2f261
- **交付**:
  - 哈希链根因修复: Go 内先赋值再算 HMAC (µs 精度对齐 timestamptz)
  - 存量修复工具 repair-chain (9084 事件/20 租户重链)
  - WORM 存储: BEFORE UPDATE/DELETE 触发器 (migration 043)
  - Tamper detection: /api/v1/audit/tamper-check 真实验证 + incident 落库
  - k3s 端到端验收: 篡改 → 立即报 hash_chain_break → 恢复后 clean

---

## 待验证项

| 验证项 | 负责人 | 状态 | 阻塞依赖 |
|--------|--------|------|----------|
| Task-A 覆盖率 ≥50% | frontend_qa | **已验证通过** (60.9%) | — |
| Task-F Settings 页 | frontend_qa | 待部署后验证 | Console 重建 |
| RBAC 回归 (Users 页) | frontend_qa | 待验证 | arch_pm 重建 gateway |

---

## 已发现的后续项（非阻塞，记录跟踪）

1. **M2M token 鉴权缺口**: identity 后端自身鉴权不完整，M2M token 可列用户。建议单独立项。
2. **Console 导航前缀劫持**: 动态 RBAC 首次上线时 console 导航路由劫持了 API 流量。已修复，arch_pm 可重建 ADR。

---

## P2 任务（规模化，后续迭代）

### Task-H: SDK 发布流程
- **负责人**: `ggcxf_researcher` (researcher)
- **范围**: npm/pypi/maven 发布管道 + SDK 功能对齐

### Task-I: Helm Chart + 运维 Runbook
- **负责人**: `arch_pm` (arch&pm&research)
- **范围**: 版本化发布、回滚流程、告警规则配置

### Task-J: 分布式追踪验证
- **负责人**: `frontend_qa` (qa&docs)
- **范围**: OTel trace 串联验证 + Grafana 告警规则

---

## Commit 清单

| Task | Commit | 描述 |
|------|--------|------|
| A | 575e895e4 | Auth service 测试覆盖 16.9% → 60.9% |
| B-设计 | edea85e7c | 动态 RBAC ADR 文档 |
| B-实现 | a0ab6ea19 | 动态 RBAC 实现 |
| B-修复1 | ee8132989 | Console 导航前缀劫持 P0 修复 |
| B-修复2 | ab762f527 | checkRouteScope 评估 claims.Roles |
| C | a1010bb7a | 内存态功能持久化 |
| D | 511656c25 | SAML XML-DSig 真实验签 |
| E | 6febb4d12 | Refresh token reuse detection 持久化 |
| F | a7584a360 | Console Settings 页真实化 |
| G | b22c2f261 | 审计 tamper-evidence 完善 |

---

## 共享工作空间规则

- 所有人在 `/Volumes/new/ggai/ggid` 工作
- **不要** git stash/checkout/reset（会覆盖他人改动）
- 只 stage 自己的文件
- 不修复其他人的代码——发现问题 DM 对应负责人
- 使用 worktree 做隔离改动
