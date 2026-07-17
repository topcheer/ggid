# RAR (Rich Authorization Requests) 设计 — RFC 9396

> 状态: Proposed | 作者: IAMExpert | 日期: 2026-07-17 | 来源: kanban I-09, FAPI 2.0 gap
> 关联: FAPI 2.0 (docs/research/oauth21-fapi2-mcp-auth-gap-gap.md)、ReBAC、ZT PDP

## 1. 问题

OAuth 2.1 scope 是不透明字符串（如 `read:users`），无法表达精确授权意图："允许客户端**只读取**用户 ID 123 的**邮箱和电话**，**有效期 1 小时**"。

RFC 9396 Rich Authorization Requests (RAR) 引入 `authorization_details` 参数 — 结构化 JSON 数组，每个元素描述一个具体授权请求（type + locations + actions + constraints）。FAPI 2.0、Open Banking、Verifiable Credentials (OID4VCI) 全部依赖 RAR。

**GGID 现状**：
- `AuthorizeRequest` 只有 `Scope []string`（不透明）
- consent screen 只展示 scope 字符串列表
- 无 `authorization_details` 参数支持
- FAPI 2.0 gap 明确标记 RAR 为 P2 缺失

## 2. 数据模型

### 2.1 authorization_details 参数格式

客户端在 `/authorize` 请求中传递：

```json
{
  "authorization_details": [
    {
      "type": "user_profile",
      "locations": ["https://api.ggid.example.com/users"],
      "actions": ["read"],
      "fields": ["email", "phone"],
      "identifier": {
        "user_id": "01234567-89ab-cdef-0123-456789abcdef"
      },
      "constraints": {
        "max_age_seconds": 3600
      }
    },
    {
      "type": "payment_initiation",
      "locations": ["https://api.bank.example.com/payments"],
      "actions": ["initiate"],
      "instructedAmount": {"currency": "EUR", "amount": "100.00"},
      "creditorName": "Merchant GmbH",
      "creditorAccount": {"iban": "DE89..."}
    }
  ]
}
```

### 2.2 RAR Detail 结构

```go
// AuthorizationDetail 是 RAR 数组中的单个授权请求元素。
type AuthorizationDetail struct {
    Type        string         `json:"type"`                   // 类型标识，如 "user_profile"、"payment_initiation"
    Locations   []string       `json:"locations,omitempty"`    // 目标 API 端点
    Actions     []string       `json:"actions,omitempty"`      // 允许的操作 read/write/delete/initiate
    Datatypes   []string       `json:"datatypes,omitempty"`    // 资源类型
    Identifier  map[string]any `json:"identifier,omitempty"`   // 精确资源标识（user_id/account_id）
    Privileges  []string       `json:"privileges,omitempty"`   // 细粒度权限
    Fields      []string       `json:"fields,omitempty"`       // 字段级控制（PII 屏蔽联动）
    Constraints map[string]any `json:"constraints,omitempty"`  // 时间/金额/频率约束
}

// RAR-aware AuthorizeRequest 扩展
type AuthorizeRequest struct {
    // ... existing fields ...
    AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}
```

## 3. GGID RAR Type → 内部权限映射

### 3.1 内置 RAR 类型注册表

```go
// RARTypeHandler 将 RAR type 映射到 GGID 内部权限评估。
type RARTypeHandler interface {
    // Type 返回 RAR type 标识。
    Type() string
    // Validate 验证 detail 的结构合法性。
    Validate(detail *AuthorizationDetail, clientID string) error
    // ToPermission 将 detail 转换为 GGID 内部权限检查请求。
    ToPermission(detail *AuthorizationDetail) (*PermissionCheck, error)
    // RenderForConsent 生成人类可读的描述（consent 页面展示）。
    RenderForConsent(detail *AuthorizationDetail) ConsentLine
}
```

### 3.2 内置类型

| Type | Actions | 映射到 GGID | 示例 |
|------|---------|------------|------|
| `user_profile` | read | identity:GetUser + fields 过滤 | "读取您的邮箱和电话" |
| `user_roles` | read | policy:ListUserRoles | "查看您的角色列表" |
| `audit_events` | read, export | audit:QueryEvents + tenant 限制 | "导出审计日志（只读）" |
| `app_access` | read, manage | gateway:AccessApp + app slug | "访问 Grafana 应用（只读）" |
| `payment_initiation` | initiate | 外部 API 转发（Open Banking 场景）| "发起 100 EUR 转账" |
| `vc_issue` | issue | OID4VCI 签发 verifiable credential | "签发您的身份凭证" |

### 3.3 自定义类型

租户可注册自定义 RAR type + handler（JSON 配置或插件），映射到 ReBAC namespace/relation 或 ABAC policy。

## 4. OAuth 端点集成

### 4.1 /authorize 端点

```go
func (s *OAuthService) CreateAuthorizationCode(ctx context.Context, req *AuthorizeRequest) (string, error) {
    // ... existing validation ...

    // RAR: 解析 + 验证 authorization_details
    if len(req.AuthorizationDetails) > 0 {
        for i, detail := range req.AuthorizationDetails {
            handler, ok := s.rarRegistry.Get(detail.Type)
            if !ok {
                return "", errors.InvalidArgument("unsupported authorization_details[%d].type: %s", i, detail.Type)
            }
            if err := handler.Validate(&detail, req.ClientID); err != nil {
                return "", errors.InvalidArgument("invalid authorization_details[%d]: %w", i, err)
            }
        }
    }

    // 存储到 authorization code（DB）
    code := &domain.AuthorizationCode{
        // ... existing fields ...
        AuthorizationDetails: req.AuthorizationDetails, // JSONB 存储
    }
    // ...
}
```

### 4.2 /token 端点

token exchange 时从 code 取出 AuthorizationDetails → 写入 access token claims：

```json
{
  "authorization_details": [
    {"type": "user_profile", "actions": ["read"], "fields": ["email", "phone"]}
  ]
}
```

**关键**：resource server（API）验证 token 时检查请求是否匹配 authorization_details：
- 请求 `GET /users/123/email` → 匹配 `{type: "user_profile", actions: ["read"], fields: ["email"]}` ✓
- 请求 `DELETE /users/123` → 不匹配（actions 无 delete）→ 403

### 4.3 token introspection

introspection 响应包含 authorization_details（resource server 据此做授权决策）：

```json
{
  "active": true,
  "authorization_details": [...]
}
```

## 5. Consent 页面

### 5.1 展示

当 authorization_details 存在时，consent 页面展示人类可读的授权列表（而非 scope 字符串）：

```
"ACME App" 请求以下权限：

📋 个人信息
   • 读取您的邮箱和电话号码

🔑 角色查询
   • 查看您的角色列表

💳 支付
   • 向 Merchant GmbH 发起 100 EUR 转账

[拒绝]  [同意]
```

### 5.2 consent API

```json
// GET /api/v1/oauth/consent-details?code=...
{
  "client_name": "ACME App",
  "client_logo": "https://...",
  "requested_scopes": ["openid", "profile"],
  "authorization_details": [
    {
      "type": "user_profile",
      "display_text": "读取您的邮箱和电话号码",
      "icon": "📋",
      "category": "personal_info"
    },
    {
      "type": "payment_initiation",
      "display_text": "向 Merchant GmbH 发起 100 EUR 转账",
      "icon": "💳",
      "category": "financial",
      "critical": true
    }
  ]
}
```

**critical=true** 的 detail 要求额外的确认（银行场景法律要求）：用户必须对每条 critical detail 单独同意。

## 6. 与 ReBAC/ABAC/PDP 协同

### 6.1 RAR → PDP 映射

resource server 收到请求时：

```
请求: GET /api/v1/users/123/email
Token authorization_details: [{type: "user_profile", actions: ["read"], fields: ["email"], identifier: {user_id: "123"}}]

→ PDP 评估:
  resource_type = "user_attribute" (from RAR type mapping)
  resource_id = "email" (from fields)
  action = "read" (from RAR actions)
  subject = token sub
  data_classification = "important" (from data_classifications table)

→ PDP result: allow/deny/stepup
```

### 6.2 RAR + ReBAC

RAR type 可映射到 ReBAC namespace：
- `type: "app_access"` → ReBAC check(namespace="app", object=slug, relation="can_view", subject=user)
- `type: "user_roles"` → ReBAC check(namespace="user", object=user_id, relation="can_read_roles", subject=caller)

### 6.3 RAR + 数据安全法

`fields` 联动数据分类：RAR 请求 `fields: ["salary"]` → data_classifications 查到 salary=core → PDP 要求额外条件（device_trusted + JIT + MFA stepup）。

## 7. 安全要求

| 要求 | 实现 |
|------|------|
| client 只能请求注册的 type | client 注册时声明 allowed_rar_types，验证时检查 |
| authorization_details 不可篡改 | 存入 authorization code（DB），token exchange 时从 code 取（非客户端） |
| consent 不可跳过 | authorization_details 非空时强制 consent（即使 scope 仅 openid） |
| critical detail 单独同意 | banking/financial 类 detail 需逐条确认 |
| 过期约束 | constraints.max_age_seconds → token TTL 限制（取 min(max_age, client default_ttl)） |
| 审计 | 每次 RAR 授权写 audit 事件（rar.authorized, type, client, user, details 摘要） |

## 8. 测试计划

| 测试 | 验证点 |
|------|--------|
| authorize + RAR | authorization_details 存入 code |
| token exchange | authorization_details 出现在 token claims |
| introspection | authorization_details 返回 |
| consent 展示 | display_text 人类可读 |
| unsupported type | 400 error |
| critical consent | 逐条确认 |
| PDP 联动 | RAR → resource_type/action → allow/deny |
| data classification | core field → stepup |
| constraints | max_age → token TTL 缩短 |

## 9. 工作量

| Phase | 内容 | 预估 |
|-------|------|------|
| 1 | AuthorizationDetail 模型 + RARTypeHandler 接口 + 注册表 | 0.5d |
| 2 | authorize/token/introspection 集成（存储 + claims + 返回） | 1d |
| 3 | 6 个内置 type handler + ToPermission + RenderForConsent | 1d |
| 4 | consent API + 前端展示 + critical 逐条确认 | 1d |
| 5 | PDP/ReBAC/数据安全法联动 + 测试 | 0.5d |
| **总计** | | **~4d** |
