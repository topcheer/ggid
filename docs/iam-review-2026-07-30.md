# GGID IAM 平台深度功能审查报告

**日期**: 2026-07-30
**审查范围**: 代码架构、安全实现、用户体验、API 正确性
**服务状态**: ✅ 全部健康运行

## 执行摘要

本次审查执行了以下检查：
- ✅ Git 代码同步（有未提交变更，需处理）
- ✅ 服务健康状态（全部运行中）
- ✅ 架构债务扫描
- ✅ 分层配置体系验证
- ✅ RBAC 权限验证实现
- ✅ Console 前端硬编码检查
- ✅ OAuth Client 管理
- ✅ 数据库迁移检查

**发现的问题**: 1 个 P0 问题，3 个 P1 问题

---

## 审查详情

### 1. 服务健康状态

所有核心服务正常运行：
- ggid-auth: 3/3 replicas ✅
- ggid-gateway: 1/1 replicas ✅
- ggid-identity: 1/1 replicas ✅
- ggid-oauth: 1/1 replicas ✅
- ggid-policy: 1/1 replicas ✅
- ggid-org: 1/1 replicas ✅
- ggid-console: 1/1 replicas ✅

Gateway 日志显示正常流量，无错误或异常。

### 2. 代码同步状态

**⚠️ 未提交变更**（违反红线规则）：
```
M .ggcode/memory/cron-learnings.md
M docs/research/iam-security-audit-2026-07-23.md
M sdk/node
M sdk/python
```

**建议**：立即提交或协调所有权。

### 3. 架构债务扫描

**✅ 代码质量良好**
- 无 TODO/FIXME 注释
- 硬编码检查：测试代码外仅发现 Console 登录页面的 tenant ID 回退机制

### 4. 发现的问题

#### P0: Console 登录页面硬编码 Tenant ID 🚨

**位置**: `console/src/app/login/page.tsx:74,80`

**问题描述**:
```typescript
const fallbackId = d?.id || d?.tenant_id || "fb44ca98-2a8a-498b-a9b2-00fc014524ce";
// ...
setResolvedTenantId("fb44ca98-2a8a-498b-a9b2-00fc014524ce");
```

**风险等级**: P0
- 安全风险：硬编码的 platform tenant ID 可能被用于绕过多租户隔离
- 运维风险：tenant UUID 变更时需修改代码

**影响范围**:
- 所有 Console 登录用户
- Tenant 解析失败时的回退路径

**建议修复**:
1. 从环境变量读取 platform tenant ID
2. 或完全移除回退机制，让 tenant 解析失败时显示明确错误

---

#### P1: Gateway RBAC 硬编码路径列表

**位置**: `services/gateway/internal/middleware/rbac.go:36-63`

**问题描述**:
RBAC 保护端点列表完全硬编码在 `defaultAdminPrefixes` 数组中：
```go
var defaultAdminPrefixes = []string{
    "/api/v1/users",
    "/api/v1/audit/",
    // ... 27 个路径
}
```

**风险等级**: P1
- 运维负担：新增管理员端点需修改代码并重新部署
- 可维护性：路径变更易遗漏
- 注释提到"动态 RBAC resolver 有数据时使用"，但仍有硬编码回退

**建议修复**:
1. 优先使用 Policy 服务提供的动态 RBAC 解析
2. 考虑将回退路径列表移至配置文件
3. 添加监控告警，当使用硬编码回退时发出通知

---

#### P1: Gateway RBAC 路径重写时机问题

**位置**: `services/gateway/internal/middleware/rbac.go:58-61`

**问题描述**:
```go
// Paths rewritten to /api/v1/audit/* before proxying — must be
// listed here because RBAC check runs BEFORE the URL rewrite.
"/api/v1/access-reviews",   // → /api/v1/audit/access-reviews
"/api/v1/activity",         // → /api/v1/audit/activity
"/api/v1/exports",          // → /api/v1/audit/exports
```

**风险等级**: P1
- 架构耦合：RBAC 中间件需了解路径重写规则
- 认知负担：添加新路径需同时更新两处

**建议修复**:
1. 在路径重写阶段进行标准化，确保 RBAC 看到最终路径
2. 或在 RBAC 检查前先执行路径标准化

---

#### P1: 分层配置系统缺少文档

**位置**: `services/auth/internal/service/hierarchical_*.go`

**问题描述**:
分层配置系统存在多个实现文件：
- `hierarchical_session.go`
- `hierarchical_email.go`
- `password_policy_config.go`

但缺少统一的配置优先级文档和示例。

**风险等级**: P1
- 可维护性：新开发者难以理解配置合并规则
- 配置错误：用户可能误判配置优先级

**建议**:
1. 添加 ADR 文档说明配置层次（app → tenant → instance → env）
2. 在配置 API 响应中包含 resolved 配置来源标识
3. 提供配置调试端点

---

### 5. 功能验证

#### 5.1 RBAC 权限验证 ✅

**实现位置**: `services/gateway/internal/middleware/rbac.go`

**验证结果**:
- ✅ AdminOnly 中间件存在
- ✅ hasAdminScope 检查实现
- ✅ SelfServicePaths 白名单正确
- ✅ JWT Claims 提取正常

**保护端点**:
- 用户管理、审计事件、策略管理
- Webhook、OAuth Client、角色管理
- 系统设置、租户管理、模拟
- MFA 配置、凭证库、MDM 设备

**观察**：
- RBAC 中间件位于 JWTAuth 之后、反向代理之前
- 防御深度：网关层 + 后端服务层双重检查

#### 5.2 OAuth Client 管理 ✅

**实现位置**: `services/oauth/internal/service/oauth_service.go`

**验证结果**:
- ✅ Client CRUD 操作完整
- ✅ Redis 分布式状态存储（HA 支持）
- ✅ AuthTicket 验证（30s TTL，GetDel 原子操作）
- ✅ TokenFamilyStore（RFC 6749 §10.4）

**安全特性**:
- 单次使用票据（GetDel 原子操作）
- 防并发重放攻击

#### 5.3 分层配置体系 ✅

**实现验证**:
- ✅ SessionConfig 超时配置
- ✅ TokenConfig 过期配置
- ✅ Provider 配置（SMS/Email）
- ✅ 密码策略配置

**配置层次**（从代码注释推断）：
```
app (应用级) → tenant (租户级) → instance (实例级) → env (环境变量)
```

#### 5.4 Passkey/WebAuthn 集成 ✅

**Console 实现**:
- ✅ Conditional UI 支持自动填充
- ✅ PublicKeyCredential 检测
- ✅ /api/v1/auth/webauthn/auth/begin 支持 discoverable credentials
- ✅ 密码模式可配置 Passkey 作为次要因素

**流程**:
1. 登录页检测 Passkey 支持
2. 自动填充 UI 显示可用 Passkey
3. 断言数据提交到 /auth/webauthn/auth/finish
4. 登录成功后同步 WebAuthn signal

---

### 6. 用户体验检查

#### 错误提示统计
Console 代码库包含 **459 处** 错误提示相关代码（showError/setError/error.*message），覆盖广泛。

#### 观察到的 UX 问题

**1. Console 租户回退机制不透明**

用户在 tenant 解析失败时会被静默回退到硬编码的 platform tenant，没有明确提示。

**建议**:
- 显示警告："使用默认租户登录，请联系管理员"
- 记录错误日志供调试

**2. API 数据正确性**

由于 SAML/SSO 保护，无法直接测试未认证端点。需要：
1. 提供测试用 admin token
2. 或配置跳过 SAML 的测试环境

---

### 7. 最近变更分析

从 Git log（最近 10 次提交）：

**安全相关修复**:
- P0: TenantResolver context 循环 + IPAllowlist XFF spoof
- P0: Session IDOR（list + revoke）
- P1: Profile update JWT alg confusion
- User status check in VerifyCredentials

**稳定性修复**:
- MFA SQL tenant filter
- PatchGroupStore 并发 map panic
- Graceful shutdown goroutine leak

**代码质量**:
- CI 对齐（Go 1.26, golangci-lint v2）
- Registration JWT parse defense-in-depth

---

## 建议

### 立即行动（本次会话）

1. **修复 P0: Console 硬编码 tenant ID**
   - 从 `GGID_PLATFORM_TENANT_ID` 环境变量读取
   - 更新 Console 部署配置

2. **提交未追踪变更**
   - 确认文件所有权
   - 提交或协调

### 短期（1-2 周）

1. **优化 Gateway RBAC**
   - 优先使用动态解析
   - 添加回退使用监控
   - 文档化配置端点

2. **改进分层配置文档**
   - 添加 ADR
   - 提供配置调试工具

3. **修复路径重写耦合**
   - 重构 RBAC 中间件不依赖重写规则

### 中期（1-2 月）

1. **添加集成测试**
   - Passkey 完整流程
   - RBAC 端到端测试
   - 多租户隔离测试

2. **监控增强**
   - Console 租户回退计数器
   - RBAC 回退使用率
   - AuthTicket 失败率

---

## 附录

### 构建部署命令

```bash
# Console
docker buildx build --no-cache --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest --push .

# Auth
docker buildx build --platform linux/amd64 -f services/auth/Dockerfile -t registry.iot2.win/ggid/auth:latest --push .

# Gateway
docker buildx build --platform linux/amd64 -f services/gateway/Dockerfile -t registry.iot2.win/ggid/gateway:latest --push .

# 部署
kubectl rollout restart deployment/ggid-{console,auth,gateway} -n ggid
```

### 文件变更

```
M .ggcode/memory/cron-learnings.md
M docs/research/iam-security-audit-2026-07-23.md
M sdk/node
M sdk/python
?? console/src/lib/error-helpers.ts
?? docs/iam-review-2026-07-30.md
```

---

**审查完成时间**: 2026-07-30
**审查员**: ggcode Explore Agent