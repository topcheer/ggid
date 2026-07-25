# RFC: Platform/Tenant 角色分离架构

## 问题

当前 platform:admin 自动继承所有 tenant:admin 权限，导致：
1. 实例管理员能操作所有租户内部数据（违反最小权限原则）
2. Impersonation/Secrets/Branding 等 per-tenant 功能被错误归到 platform 层
3. tenant:admin 不是按租户赋权（scope_type/scope_id 未被 gateway 使用）

## 目标权限模型

```
Platform Admin (实例管理员)
├── 系统配置/初始化
├── 创建/暂停/激活租户
├── 全局审计（跨租户）
├── 全局威胁仪表盘
└── 系统健康监控
    ※ 不自动获得任何租户内权限
    ※ 除非同时被分配为某租户的 tenant:admin

Tenant Admin (租户管理员, per-tenant)
├── 用户/角色/组织管理
├── 安全策略（CAE/密码/MFA/Passkey）
├── OAuth Clients / Webhooks / API Keys
├── Branding（白标定制）           ← 从 Platform 移入
├── Feature Flags（功能开关）       ← 从 Platform 移入
├── Secrets / Key Rotation / Backup ← 从 Platform 移入
├── Impersonation（租户内模拟）     ← 从 Platform 移入
├── 审计日志（本租户）
├── SCIM / LDAP / SAML 配置
└── 访问审查/申请

Regular User (user:self)
├── Dashboard / 个人资料
├── 我的会话
└── 访问申请
```

## 变更矩阵

### 1. Gateway RBAC (arch_pm)

**router.go:749-753** — 删除 platform→tenant 自动继承：
```go
// BEFORE
if scl == "platform:admin" {
    hasPlatform = true
    hasTenant = true    // ❌ 删除
}

// AFTER
if scl == "platform:admin" {
    hasPlatform = true
    // hasTenant 由独立的 tenant:admin scope 控制
}
```

**platformOnlyPaths** — 缩减为纯平台操作：
```go
var platformOnlyPaths = []string{
    "/api/v1/system/",
    "/api/v1/tenants/create",
    "/api/v1/org/tenants/suspend",
    "/api/v1/org/tenants/activate",
    "/api/v1/admin/audit/global",
    "/api/v1/admin/threats/dashboard",
}
// 移除 impersonate/secrets/key-rotation/backup（这些是 tenant 级）
```

**adminOnlyPaths** — 新增 tenant 级运维操作：
```go
var adminOnlyPaths = []string{
    // 已有
    "/api/v1/users", "/api/v1/roles", "/api/v1/audit/",
    "/api/v1/policies", "/api/v1/webhooks", "/api/v1/oauth/clients",
    "/api/v1/settings/", "/api/v1/identity/dashboard",
    "/api/v1/tenants", "/api/v1/impersonate",
    "/api/v1/api-keys", "/api/v1/access-keys",
    "/oauth/clients",
    // 新增（从 platform 移入）
    "/api/v1/admin/secrets",
    "/api/v1/admin/key-rotation",
    "/api/v1/admin/backup",
    "/api/v1/admin/impersonate",  // 如果有 admin 前缀的路由
}
```

### 2. 前端角色解耦 (shen_frontend)

**api.ts:252-254**：
```ts
// BEFORE
isPlatformAdmin: role === "platform_admin",
isTenantAdmin: role === "tenant_admin" || role === "platform_admin",  // ❌

// AFTER
isPlatformAdmin: role === "platform_admin",
isTenantAdmin: role === "tenant_admin",  // 不再自动包含 platform
```

**sidebar.tsx** — 重组导航分组：

Platform 组（仅 platform:admin 可见）：
```tsx
{
  label: "Platform", icon: Building, requiredScope: "admin", items: [
    { href: "/admin/tenants", label: "Tenants" },
    { href: "/admin/audit/global", label: "Global Audit" },
    { href: "/admin/threats", label: "Threat Dashboard" },
  ],
}
```

Tenant 组（tenant:admin 可见）— 新增"运维"子组或合并到现有组：
```tsx
// Security 组新增
{ href: "/admin/impersonate", label: "Impersonation" },
{ href: "/admin/secrets", label: "Secrets" },
{ href: "/admin/key-rotation", label: "Key Rotation" },
{ href: "/admin/backup", label: "Backup" },

// Applications 组新增
{ href: "/settings/branding", label: "Branding" },
{ href: "/settings/feature-flags", label: "Feature Flags" },
```

**auth-guard.tsx:62-68** — 路由权限映射：
```ts
// BEFORE
"/admin": "platform",

// AFTER — 按路径细分
"/admin/tenants": "platform",
"/admin/audit/global": "platform",
"/admin/threats": "platform",
"/admin/impersonate": "tenant",   // 移到 tenant
"/admin/secrets": "tenant",
"/admin/key-rotation": "tenant",
"/admin/backup": "tenant",
```

### 3. JWT per-tenant 角色加载 (backend)

当前 user_roles 表已有 `scope_type` + `scope_id`，支持按租户赋权：
```sql
-- admin 用户可以同时有 platform 和 tenant 角色
INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
VALUES ('admin-uuid', 'platform-role-uuid', 'global', NULL);

INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
VALUES ('admin-uuid', 'tenant-role-uuid', 'tenant', 'tenant-uuid-here');
```

JWT claims 需要根据登录时选择的 tenant_id 加载对应的 tenant 角色：
- 登录时传 tenant_id → 只加载该 tenant 的角色
- platform:admin 角色全局有效（scope_type='global'）
- tenant:admin 角色只在对应 tenant 内有效

## admin 用户兼容性

setup wizard 创建的 admin 用户同时被分配 platform:admin + tenant:admin（已有，integration_handlers.go:319-320）。
移除自动继承后，admin 仍有显式的 tenant:admin 角色，不会丢失功能。

## 风险

1. **现有 platform:admin 用户如果没有显式 tenant:admin 角色** → 会丢失租户内功能
   - 缓解：setup wizard 已同时分配两个角色
   - 迁移脚本：检查所有 platform:admin 用户是否有对应 tenant 角色

2. **Gateway 测试** — 现有 RBAC 测试假设 platform→tenant 继承
   - 需更新 rbac_scope_test.go
