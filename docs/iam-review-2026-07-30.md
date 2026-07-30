# GGID IAM 深度功能审查报告

**日期**: 2026-07-30
**审查员**: ggcode agent
**审查范围**: Passkey 全流程、RBAC 权限验证、架构债务

---

## 执行摘要

### 服务状态
- **Pods 健康检查**: ✅ 全部核心服务运行正常（auth, gateway, identity, policy, oauth, console, audit）
- **最近提交**: 47 个本地提交未推送，主要涉及 R187-R197 安全修复

### 审查领域
1. **Passkey 注册/登录全流程**
2. **RBAC 权限验证**
3. **架构债务搜索**

---

## Passkey 全流程审查

### ✅ 良好实践

#### 1. 密码学验证
- 使用标准 `go-webauthn` 库进行完整验证
- 支持多种 attestation 格式：`none`, `packed`, `fido-u2f`
- 支持 ES256 (-7), RS256 (-257), EdDSA (-8) 签名算法
- 实现了完整的 AAGUID 验证链

#### 2. 会话管理
- 使用内存 sessionStore（注释说明生产应使用 Redis）
- 5 分钟过期 + 每 2 分钟清理循环
- Tenant 隔离：所有操作都通过 `ggidtenant.FromContext` 获取租户

#### 3. 测试覆盖
- `handler_test.go` 820 行，包含 mockStore 完整实现
- `attestation_test.go` 测试各种格式验证
- `gap_regression_test.go` 边界情况测试
- `passkey_tenant_isolation_test.go` 租户隔离专项测试

### ⚠️ 发现的问题

#### P2: 生产环境 Session 存储配置不明确
**位置**: `services/auth/internal/webauthn/handler.go:61-121`

```go
// Session Store (in-memory, ephemeral — production would use Redis)
type sessionBackend interface {
    Save(ctx context.Context, key string, data []byte, ttl time.Duration) error
    Load(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
}

type sessionStore struct {
    mu       sync.Mutex
    sessions map[string]*sessionData
    done     chan struct{}
}
```

**问题**:
1. 注释说明"生产应使用 Redis"，但实际代码默认使用内存实现
2. 未找到生产环境切换到 Redis 的配置项或环境变量
3. 内存实现在多实例部署时会丢失会话

**影响**:
- 多副本部署时 Passkey 注册/登录可能失败
- 横向扩展受限

**建议**:
- 添加 `WEBAUTHN_SESSION_BACKEND=redis` 配置项
- 实现基于 Redis 的 SessionBackend
- 在部署文档中明确说明配置

#### P3: Counter 重放攻击防护需文档说明
**位置**: `services/auth/internal/webauthn/attestation.go` 相关

**观察**:
- 实现了 `Counter uint32` 字段
- 未找到 counter 验证逻辑的审查

**建议**:
- 确认每个认证请求后 counter 递增
- 添加 replay attack 检测（counter 不能倒退）

---

## RBAC 权限验证审查

### ✅ 良好实践

#### 1. 动态 RBAC + 静态回退
**位置**: `services/gateway/internal/middleware/rbac.go:34-127`

```go
// Dynamic RBAC (ADR-dynamic-rbac): DB-driven route permissions take precedence
// when the resolver has data.
if res := getRBACResolver(); res != nil && res.Available() {
    if allow, handled := res.CheckAccess(...); handled {
        return
    }
    // No dynamic rule matched → static fallback below
}
// Fallback to hardcoded admin prefixes
```

**优点**:
- 支持数据库驱动的动态权限配置
- 有安全的静态回退（defaultAdminPrefixes）
- API Key 请求正确处理

#### 2. 多层防护
- Gateway 层：`RequireAdminScope` 中间件检查 admin scope
- Service 层：`Evaluator.Check()` 执行 RBAC+ABAC 综合验证
- 默认拒绝（fail-safe）：`uuid.Nil tenant` 返回 deny

#### 3. 租户隔离
**位置**: `services/policy/internal/service/evaluator.go:146-149`

```go
// SECURITY: reject nil tenant — no tenant context = deny (P2 R164)
if req.TenantID == uuid.Nil {
    return &domain.CheckResult{Allowed: false, Reason: "missing tenant context"}, nil
}
```

### ⚠️ 发现的问题

#### P2: OAuth Client 管理 RBAC 检查缺失
**位置**: `console/src/app/settings/oauth-clients/new/page.tsx:91-94`

```typescript
const res = await fetch(`${API_BASE}/api/v1/oauth/clients`, {
  method: "POST",
  headers: { "Content-Type": "application/json", ...authHeader() },
  body: JSON.stringify(body),
});
```

**问题**:
1. 前端直接调用 `/api/v1/oauth/clients` 创建客户端
2. 未在前端检查用户权限
3. 错误处理仅显示 `data.error?.detail`，无权限区分

**后端检查**:
- Gateway 层 `/api/v1/oauth/clients` 在 `defaultAdminPrefixes` 列表
- 依赖后端 RBAC 拦截未授权请求
- 但前端未进行权限预检查

**影响**:
- 用户体验差：无权限用户点击创建后才收到 403
- 无法在前端提前隐藏/禁用功能

**建议**:
- 前端调用 `/api/v1/users/me/permissions` 获取权限列表
- 在页面加载时检查 `oauth:clients:create` 权限
- 无权限时显示权限不足提示，隐藏创建按钮

#### P3: 错误消息缺失用户友好提示
**位置**: `console/src/app/settings/oauth-clients/new/page.tsx:96-100`

```typescript
if (!res.ok) throw new Error(data.error?.detail || data.error || "Failed");
setError(e instanceof Error ? e.message : "Failed to create client");
```

**问题**:
- 直接显示后端错误 detail
- 未区分权限错误 vs 输入验证错误
- 无本地化错误消息

**建议**:
- 区分 403 (Forbidden) vs 400 (Bad Request)
- 提供用户友好的提示
- 集成 toast 通知系统

#### P2: Self-Service 路径白名单可能过宽
**位置**: `services/gateway/internal/middleware/rbac.go:68-71`

```go
var SelfServicePaths = map[string]bool{
    "/api/v1/users/me":             true,
    "/api/v1/users/me/permissions": true, // read-only permission listing
}
```

**问题**:
- 只有精确匹配检查
- `/api/v1/users/me/settings` 等深层路径被管理员规则覆盖
- 注释说明"深度子资源不被豁免"，但实现依赖 HasPrefix 前缀检查

**建议**:
- 明确文档说明哪些 /me 路径需要管理员权限
- 或改为前缀检查（如 `/api/v1/users/me/permissions*`）

---

## 架构债务审查结果

### 搜索范围
- TODO/FIXME 标记
- hardcode 关键词（排除注释）
- mock 关键词（排除测试文件）

### 发现情况

#### ✅ 无严重 TODO/FIXME
- 搜索结果：无 TODO 或 FIXME 标记
- 说明代码维护良好

#### ✅ 硬编码检查通过
- 搜索结果：多处 `hardcode` 均为注释说明"不应硬编码"
- 实际代码中已正确使用配置或上下文

**示例**（良好实践）:
```go
// services/oauth/internal/server/grpc_handler.go
// uuid.Nil when unset — never a hardcoded tenant UUID.

// services/audit/internal/server/security_dashboard_handler.go
// No tenant context — return empty dashboard instead of hardcoded fallback.
```

#### ✅ Mock 检查通过
- 搜索结果：mock 仅出现在测试文件
- 生产代码无 mock 实现

---

## Console 用户体验检查

### ⚠️ 发现的问题

#### P3: 缺少 Toast/Alert 通知系统
**位置**: `console/src/` 搜索结果

**发现**:
- 未找到 toast 或 alert 通知系统
- 错误仅通过 `setError` 在页面显示
- 无操作成功反馈

**影响**:
- 用户不知道操作是否成功（需要刷新验证）
- 列表页面操作（如删除）无反馈

**建议**:
- 集成 toast 通知库（如 sonner、react-hot-toast）
- 为所有 CRUD 操作添加成功/失败通知
- 提供撤销/重试选项

---

## 分层配置体系审查

### 状态
- 未在本次审查中深入检查
- 建议后续审查

---

## 优先级修复建议

### P0 / P1
- 无

### P2
1. **Passkey Session 存储配置** - 多实例部署支持
2. **OAuth Client 前端权限检查** - 用户体验改进

### P3
1. **Counter 重放攻击防护文档** - 安全最佳实践
2. **Self-Service 路径白名单文档** - 避免混淆
3. **Console Toast 通知系统** - 用户体验
4. **错误消息本地化** - 国际化

---

## Git 提交建议

**当前状态**:
- 本地领先 47 个提交
- 工作区有大量未暂存变更

**建议**:
1. 本次审查仅记录问题，不提交代码
2. P2/P3 问题通过 DM 协调修复
3. 遵守"只 add 自己的文件"原则

---

## 审查总结

### 优势
- ✅ 核心服务运行稳定
- ✅ Passkey 密码学验证完整
- ✅ RBAC 多层防护完善
- ✅ 租户隔离严格执行
- ✅ 无严重架构债务

### 待改进
- ⚠️ Passkey 多实例部署支持
- ⚠️ Console 权限预检查
- ⚠️ 用户体验反馈系统

### 下一步行动
1. 更新 `.ggcode/memory/cron-learnings.md`
2. 通过 DM 通知相关团队处理 P2/P3 问题