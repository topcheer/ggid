# GGID SDK 统一接口规范

所有语言 SDK 必须实现以下接口，保持功能对齐。

## 1. 认证 (Authentication)

### 1.1 JWT 验证
```
verifyToken(token) → Claims { user_id, tenant_id, roles, scope, exp }
```

### 1.2 OAuth/OIDC
```
getDiscovery() → { issuer, authorization_endpoint, token_endpoint, jwks_uri, userinfo_endpoint }
getJWKS() → { keys: [...] }
exchangeCode(code, redirectUri, clientId, clientSecret) → { access_token, refresh_token, id_token, expires_in }
refreshToken(refreshToken, clientId, clientSecret) → { access_token, refresh_token, expires_in }
getUserInfo(accessToken) → { sub, name, email, roles }
revokeToken(token, clientId, clientSecret) → void
```

## 2. RBAC 授权 (Role-Based Access Control)

### 2.1 权限检查
```
checkPermission(token, resource, action) → { allowed: bool, reason: string }
```
调用 `GET /api/v1/policies/check?resource={resource}&action={action}`

### 2.2 角色管理
```
assignRole(userId, roleId) → void
revokeRole(userId, roleId) → void
getUserRoles(userId) → [{ id, key, name }]
listRoles() → [{ id, key, name, permissions }]
```

### 2.3 权限树
```
listPermissions() → [{ resource, actions: [...], description }]
```
调用 `GET /api/v1/policies/permissions/tree`

## 3. ABAC 授权 (Attribute-Based Access Control)

### 3.1 策略评估
```
checkPolicy(request: {
  action: string,
  resource: string,
  subject: { user_id, roles, attributes },
  conditions: [{ field, operator, value }],
  tenant_id: string
}) → { allowed: bool, matched_rules: [...], reason: string }
```
调用 `POST /api/v1/policies/abac/evaluate`

### 3.2 ABAC 条件评估
```
evaluateABAC(request: {
  action: string,
  resource: string,
  conditions: [{ field, operator, value }]
}) → { matched: bool, matched_rules: [...] }
```

## 4. HTTP 中间件

### 4.1 认证中间件
```go
// 每种语言提供框架集成:
// Go:       func AuthMiddleware(next http.Handler) http.Handler
// Node.js:  function authMiddleware(req, res, next)
// Java:     implements HandlerInterceptor
// Python:   @requires_auth decorator
```

### 4.2 权限中间件
```go
// 每种语言提供:
// Go:       func RequirePermission(resource, action string) func(http.Handler) http.Handler
// Node.js:  function requirePermission(resource, action) => (req, res, next)
// Java:     @RequirePermission(resource="products", action="delete")
// Python:   @requires_permission("products", "delete")
```

## 5. 语言支持矩阵

| 功能 | Go | Node.js | Java | Python |
|------|-----|---------|------|--------|
| JWT 验证 | ✅ | ✅ | ✅ | TODO |
| OAuth/OIDC | ✅ | ✅ | ✅ | TODO |
| CheckPermission | ✅ | TODO | TODO | TODO |
| CheckPolicy (ABAC) | ✅ | TODO | TODO | TODO |
| AssignRole/RevokeRole | ✅ | TODO | TODO | TODO |
| GetUserRoles | ✅ | TODO | TODO | TODO |
| ListPermissions | ✅ | TODO | TODO | TODO |
| EvaluateABAC | ✅ | TODO | TODO | TODO |
| HTTP 中间件 | ✅ | TODO | TODO | TODO |
| 权限中间件 | TODO | TODO | TODO | TODO |
