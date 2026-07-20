# GGID 细粒度权限控制指南

## 概述

GGID 提供四层权限控制，从粗到细：

1. **RBAC（角色级）** — 用户有什么角色，角色有什么权限
2. **菜单可见性** — 前端根据权限显示/隐藏菜单项
3. **按钮可用性** — 有权限的按钮可点击，无权限的禁用（灰色）
4. **ABAC（数据级）** — 行级数据隔离，用户只能看到自己团队/部门的数据

## 权限模型

### RBAC：角色 → 权限

```
用户 → 角色 → 权限
sales_manager → Sales Manager → orders:read, orders:write, orders:approve, inventory:read
warehouse_manager → Warehouse Manager → inventory:read, inventory:write, inventory:delete, orders:read
finance_officer → Finance Officer → orders:read, invoices:read, invoices:write, reports:read
```

### ABAC：属性 → 数据过滤

```
用户属性：
  org_id: "sales-dept"        // 部门
  group_id: "team-a"           // 组
  role: "manager"              // 角色
  all_access: true             // 全量访问

Policy Check 返回：
  allowed: true
  data_filter: {org_id: "sales-dept", group_id: "team-a"}
  
前端用 data_filter 过滤数据：
  GET /api/orders?org_id=sales-dept&group_id=team-a
```

## API

### 检查权限（RBAC + ABAC）

```http
POST /api/v1/policies/check
Content-Type: application/json
X-Tenant-ID: 00000000-0000-0000-0000-000000000001
Authorization: Bearer <token>

{
  "user_id": "363a49b5-...",
  "resource_type": "orders",
  "action": "read",
  "attributes": {
    "org_id": "sales-dept",
    "group_id": "team-a"
  }
}
```

**响应：**
```json
{
  "allowed": true,
  "matched_by": "rbac",
  "data_filter": {
    "org_id": "sales-dept",
    "group_id": "team-a"
  }
}
```

### 创建 ABAC 策略

```http
POST /api/v1/policies
{
  "name": "team-a-orders-read",
  "effect": "allow",
  "actions": ["read"],
  "resources": ["orders"],
  "conditions": {
    "org_id": "sales-dept",
    "group_id": "team-a"
  },
  "priority": 10
}
```

## 前端集成

### 1. 菜单可见性

```typescript
// 有权限才显示菜单
const menuItems = [
  { label: 'Dashboard', path: '/dashboard', show: true },
  { label: 'Inventory', path: '/inventory', show: hasPermission(user, 'inventory:read') },
  { label: 'Orders', path: '/orders', show: hasPermission(user, 'orders:read') },
  { label: 'Admin', path: '/admin', show: hasScope(user, 'admin') },
].filter(item => item.show);
```

### 2. 按钮可用性

```typescript
// 有权限可点击，无权限禁用
<Button 
  disabled={!hasPermission(user, 'inventory:write')}
  title={!hasPermission(user, 'inventory:write') ? 'No write permission' : ''}
>
  New Inventory
</Button>
```

### 3. 行级数据过滤

```typescript
// 调用 policy check 获取 data_filter
const result = await client.policyCheck({
  user_id: user.id,
  resource_type: 'orders',
  action: 'read',
  attributes: { org_id: user.org_id, group_id: user.group_id }
});

// 用 data_filter 过滤 API 请求
const orders = await fetch(`/api/orders?${new URLSearchParams(result.data_filter)}`);

// 显示过滤提示
{result.data_filter.group_id && (
  <Alert message={`仅显示 ${result.data_filter.group_id} 的数据`} />
)}
```

## Demo 应用

### ERP Demo (erp.iot2.win)

3 个角色演示完整权限控制：

| 功能 | Sales Manager | Warehouse Manager | Finance Officer |
|------|:---:|:---:|:---:|
| Dashboard | ✅ | ✅ | ✅ |
| Inventory 查看 | ✅ (只读) | ✅ (读+写+删) | ❌ (403) |
| Orders 查看 | ✅ | ✅ | ✅ (只读) |
| Orders 审批 | ✅ | ❌ | ❌ |
| Orders 发货 | ✅ | ✅ | ❌ |
| Reports | ✅ | ✅ | ✅ (读+写) |
| Admin 页面 | ❌ (403) | ❌ (403) | ❌ (403) |

### SDK Demo (9 语言 × OAuth + SAML)

每个 SDK demo 实现：
- OAuth/SAML 登录 → 获取 JWT
- `hasPermission()` 控制菜单可见性
- 按钮禁用/启用
- `policyCheck()` API 验证
- 403 页面

### 细粒度权限 Demo (高级)

3 个用户演示行级数据隔离：

| 用户 | 部门 | 组 | 数据可见性 |
|------|------|-----|---------|
| alice_sales | sales | team-a | 只看 team-a 订单 |
| bob_sales | sales | team-b | 只看 team-b 订单 |
| manager | - | - | 看所有订单 |

## SDK 集成

### Go SDK

```go
client := ggid.NewClient("https://ggid.example.com")

// 检查权限
result, _ := client.PolicyCheck(ctx, &ggid.PolicyCheckRequest{
    UserID:      userID,
    ResourceType: "orders",
    Action:      "read",
    Attributes:  map[string]any{"org_id": "sales-dept", "group_id": "team-a"},
})

if result.Allowed {
    // 用 data_filter 过滤数据
    orders := client.ListOrders(ctx, result.DataFilter)
}
```

### Node SDK

```typescript
const result = await client.policyCheck({
  user_id: userId,
  resource_type: 'orders',
  action: 'read',
  attributes: { org_id: 'sales-dept', group_id: 'team-a' }
});

if (result.allowed) {
  const orders = await fetchOrders(result.data_filter);
}
```

## 配置

### sys_config

```json
{
  "webauthn_config": {"rp_id": "ggid-console.iot2.win"},
  "radius_config": {"server": "radius.example.com", "secret": "...", "enabled": true},
  "yubico_config": {"client_id": "12345", "secret_key": "...", "enabled": true},
  "saml_config": {"sp_entity_id": "...", "sp_acs_url": "..."}
}
```

### 环境变量

```bash
GGID_URL=https://ggid.example.com
GGID_CLIENT_ID=gcid_xxx
GGID_CLIENT_SECRET=gcs_xxx
GGID_TENANT_ID=00000000-0000-0000-0000-000000000001
```