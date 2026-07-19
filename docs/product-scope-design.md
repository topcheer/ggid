# GGID 产品化 Scope 设计

## 问题
当前系统只有硬编码的 3 个角色（admin/manager/user），没有区分实例级和租户级。
所有用户登录后看到的导航完全一样，体验极差。

## Scope 体系设计

### 实例级（Platform Scope）
这些功能属于**整个平台实例的管理**，不属于任何单个租户：

| Scope | 描述 | 可见功能 |
|-------|------|----------|
| `platform:admin` | 实例超级管理员 | Tenants CRUD, Instance Settings, Feature Flags, System Health |
| `platform:operator` | 实例运维 | System Health, Monitoring, Log Viewer |

**实例级功能列表：**
- `/admin/tenants` — 创建/编辑/删除租户
- `/admin/instances` — 实例配置（DB/Redis/NATS/邮件）
- `/admin/feature-flags` — 全局功能开关
- `/admin/system-health` — 系统健康检查
- `/admin/license` — 许可证管理

### 租户级（Tenant Scope）
这些功能属于**租户内部的 IAM 管理**：

| Scope | 描述 | 可见功能 |
|-------|------|----------|
| `tenant:admin` | 租户管理员 | Users, Roles, Policies, Audit, Security, OAuth, Settings |
| `tenant:auditor` | 租户审计员 | Audit (read-only), Compliance, Sessions (read-only) |
| `tenant:helpdesk` | 租户客服 | Users (read + reset password), Sessions |

**租户级功能列表：**
- `/users` — 用户管理
- `/roles` — 角色管理
- `/organizations` — 组织架构
- `/policies` — 策略管理
- `/oauth-clients` — OAuth 应用
- `/webhooks` — Webhook 配置
- `/audit` — 审计日志
- `/security/*` — 安全监控（CAE, Session, Risk）
- `/settings/*` — 租户配置（SCIM, LDAP, Conditional Access）

### 用户级（User Scope）
所有登录用户都能看到的基础功能：

| Scope | 描述 | 可见功能 |
|-------|------|----------|
| `user:self` | 普通用户 | Profile, My Sessions, My Activity, Access Requests |

**用户级功能列表：**
- `/dashboard` — 个人摘要
- `/profile` — 个人信息 + 安全设置（MFA, Passkey, Password）
- `/sessions` — 我的活跃会话
- `/access-requests` — 申请权限

## Bootstrap 流程（从空白实例开始）

### Step 1: 实例初始化
```
POST /api/v1/system/bootstrap
{
  "admin_username": "superadmin",
  "admin_email": "admin@company.com",
  "admin_password": "SecurePassword!"
}
```
→ 创建默认租户 + 平台管理员账户
→ JWT scopes: ["platform:admin", "tenant:admin", "user:self"]

### Step 2: 创建第一个租户（平台管理员操作）
```
POST /api/v1/tenants
{
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "enterprise"
}
```
→ 平台管理员自动成为该租户的 tenant:admin

### Step 3: 租户管理员邀请用户
```
POST /api/v1/users
{
  "username": "john",
  "email": "john@acme.com",
  "roles": ["tenant:helpdesk"]
}
```
→ 用户收到邀请邮件，设置密码后登录
→ JWT scopes: ["user:self", "tenant:helpdesk"]

## JWT Claims 结构
```json
{
  "tenant_id": "uuid",
  "scopes": ["platform:admin", "tenant:admin", "user:self"],
  "sub": "user-uuid",
  "iss": "ggid",
  "exp": 1234567890
}
```

## 前端角色判定逻辑
```typescript
function getRoleLevel(scopes: string[]): "platform" | "tenant" | "user" {
  if (scopes.some(s => s.startsWith("platform:"))) return "platform";
  if (scopes.some(s => s.startsWith("tenant:"))) return "tenant";
  return "user";
}
```
