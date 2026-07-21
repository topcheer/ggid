# GGID Built-in Roles & Permission Model

## 内置角色（不可修改，属于系统角色）

这些角色是 GGID 平台级别的角色，管理的是 **console 权限**（管理 GGID 平台本身），与外部应用的业务权限完全无关。

| Role | Key | Scope | 描述 |
|------|-----|-------|------|
| Platform Administrator | `platform:admin` | 全平台 | 管理所有租户、系统配置、平台级审计。相当于超级管理员。 |
| Tenant Administrator | `tenant:admin` | 单个租户 | 管理本租户内的用户、角色、应用、审计。不能访问其他租户。 |
| User | `user:self` | 单个用户 | 管理自己的个人资料。每个用户默认拥有。 |
| Viewer | `viewer` | 单个租户 | 只读访问本租户的审计、用户列表等（不包含修改权限）。 |

### 关键原则

1. **平台管理员 ≠ 应用管理员**：Platform Administrator 管理的是 GGID 平台（创建租户、配置 OAuth Client、查看全局审计等），不拥有任何外部应用的业务权限。

2. **租户管理员 ≠ 应用管理员**：Tenant Administrator 管理的是本租户的用户/角色/审计，但默认不拥有任何应用的业务权限。Tenant Admin 可以给应用创建角色和分配用户。

3. **应用权限是显式的**：每个外部应用（如 ERP demo）有自己的角色和权限（inventory:read、orders:write 等），这些角色由租户管理员创建，由租户管理员显式分配给用户。

4. **无继承、无 fallback**：平台角色不继承应用权限，应用角色不影响平台权限。两套体系完全独立。

## 内置角色权限（Console 权限）

### Platform Administrator (`platform:admin`)
- `platform:admin` — 全平台管理
- `users:read/write` — 管理所有租户的用户
- `roles:read/write` — 管理所有租户的角色
- `security:read/write` — 安全策略管理
- `settings:read/write` — 系统设置
- `audit:read` — 全局审计日志
- `oauth:read/write` — OAuth Client 管理

### Tenant Administrator (`tenant:admin`)
- `users:read/write` — 管理本租户用户
- `roles:read/write` — 管理本租户角色
- `security:read/write` — 本租户安全策略
- `settings:read/write` — 本租户设置
- `audit:read` — 本租户审计日志
- `oauth:read/write` — 本租户 OAuth Client

### User (`user:self`)
- `profile:self` — 管理自己的个人资料

### Viewer (`viewer`)
- `users:read` — 查看用户列表
- `roles:read` — 查看角色
- `security:read` — 查看安全状态
- `audit:read` — 查看审计

## 外部应用权限模型（以 ERP Demo 为例）

外部应用的权限完全独立于上述内置角色，由租户管理员定义：

### ERP 应用角色
| Role | Permissions | 对应 Console 角色 |
|------|-------------|------------------|
| ERP Admin | 所有 ERP 权限 | 通常由 Tenant Admin 创建 |
| ERP Manager | inventory:*, orders:*, audit:read, dashboard:read | 业务岗位 |
| ERP Sales | inventory:read, orders:read/write, dashboard:read | 业务岗位 |
| ERP Viewer | inventory:read, orders:read, dashboard:read | 业务岗位 |

### ERP 权限
- `inventory:read/write/delete`
- `orders:read/write/approve/read:all`
- `audit:read`
- `dashboard:read`

这些权限只出现在 JWT 的 `permissions` claim 中，不会出现在 `scopes` 中。

## 租户生命周期

```
Platform Admin → 创建租户 → Bootstrap 租户管理员
                            ↓
                   租户管理员登录 → 创建应用角色 + 权限
                            ↓
                   租户管理员 → 创建用户 → 分配角色
                            ↓
                   用户登录 → JWT 包含 permissions claim
                            ↓
                   外部应用 → 用 permissions 做授权决策
```

## JWT Claims 结构

```json
{
  "tenant_id": "00000001-0000-0000-0000-000000000001",
  "sub": "user-uuid",
  "scopes": ["user:self"],           // OAuth scopes (openid, profile, email)
  "roles": ["tenant:admin"],          // GGID 内置角色 + 应用角色
  "permissions": [                    // 显式权限（console 权限 + 应用权限）
    "users:read",
    "roles:write",
    "inventory:read",
    "orders:write"
  ],
  "iss": "ggid-auth",
  "exp": 1234567890
}
```
