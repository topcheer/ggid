# 产品化任务分配计划

> 基于 `docs/research/iam-product-deep-review.md` 验证后的行动计划
> 创建日期：2026-07-21

---

## P0 任务（产品可用性阻塞）

### Task-A: Auth Service 测试覆盖补强
- **负责人**: `ggcxf_backend` (backend)（从 QA 转交）
- **验收方**: `frontend_qa` (qa&docs) — 完成后验证覆盖率
- **范围**:
  - 为 `services/auth/internal/service/` 核心方法补充单元测试
  - 重点：`VerifyCredentials`、`Register`、`CheckBruteForce`、MFA setup/verify
  - 目标：auth service 覆盖率从当前（低）提升到 50%+
  - 不含已删除的 token 签发方法（Login/Refresh 等）
- **依赖**: 无
- **验收**: `go test ./services/auth/internal/service/... -cover` 覆盖率 >= 50%

### Task-B: 动态 RBAC 替代硬编码前缀
- **设计**: `arch_pm` (arch&pm&research) — **已完成** (commit edea85e7c, `docs/design/adr-dynamic-rbac.md`)
- **实现**: `ggcxf_backend` (backend) — 待实现
- **范围**:
  - 替换 `services/gateway/internal/middleware/rbac.go` 中的 `adminPrefixes` 硬编码数组
  - 改为从 `role_permissions` 表 + Redis cache 动态查询
  - 保持 `RequireAdminScope` 接口不变，内部逻辑改为 DB 驱动
  - 新增 migration: `role_permissions` 表已有则复用，否则创建
- **依赖**: 无
- **验收**: 新增角色无需改代码即可访问 admin 端点

### Task-C: 内存态功能持久化
- **负责人**: `ggcxf_backend` (backend)
- **范围**:
  - 将以下功能从 `memoryMapRepo` 迁移到 PostgreSQL:
    - alert rules (audit service)
    - composite rules (policy service)
    - dashboard widgets (gateway service)
    - SCIM group store (identity service)
  - 保持 API 不变，仅替换存储层
- **依赖**: 无
- **验收**: Pod 重启后数据不丢失；多副本部署状态一致

---

## P1 任务（企业销售阻塞）

### Task-D: SAML XML-DSig 真实验签
- **负责人**: `guardian_security` (security)
- **范围**:
  - 当前 `server.go:1109` 仅检查 `<ds:Signature>` 元素存在
  - 改为使用 `github.com/crewjam/saml` 或自实现 RSA XML-DSig 验证
  - 添加 SAML 签名验证单元测试
- **依赖**: 无
- **验收**: 伪造签名的 SAML 断言被拒绝；合法签名通过

### Task-E: Refresh Token 家族轮换持久化
- **负责人**: `ggcxf_backend` (backend)
- **范围**:
  - 当前 token family 存储在 memory map，需迁移到 PostgreSQL
  - 实现 RFC 6749 Section 10.4 的 reuse detection
  - token 被重用时自动吊销整个 family
- **依赖**: Task-C 完成更好（但可并行）
- **验收**: refresh token 重用后整个 family 被吊销

### Task-F: Console Settings 页真实化
- **负责人**: `shen_frontend` (frontend)
- **范围**:
  - 修复多个 settings 子页面返回 404 的问题
  - 密码策略配置页连接真实 API
  - 品牌化（logo、颜色、自定义 CSS）连接 tenant 配置
- **依赖**: 无
- **验收**: settings 页面无 404；配置变更持久化

### Task-G: 审计 Tamper-Evidence 完善
- **负责人**: `guardian_security` (security)
- **范围**:
  - 哈希链已有基础实现，完善验证逻辑
  - 添加 WORM（Write Once Read Many）存储策略
  - 添加 tamper detection 报警
- **依赖**: 无
- **验收**: 审计记录被篡改后系统能检测并告警

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

## 执行顺序

1. **第一波（立即可启动）**: Task-A, Task-B, Task-C, Task-D, Task-F
2. **第二波（Task-C 完成后）**: Task-E
3. **第三波（P1 全部完成后）**: Task-G, Task-H, Task-I, Task-J

## 共享工作空间规则

- 所有人在 `/Volumes/new/ggai/ggid` 工作
- **不要** git stash/checkout/reset（会覆盖他人改动）
- 只 stage 自己的文件
- 不修复其他人的代码——发现问题 DM 对应负责人
- 使用 worktree 做隔离改动
