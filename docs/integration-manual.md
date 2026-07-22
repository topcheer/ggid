# GGID 接入手册

> GGID IAM Suite · v1.0 · 2026-07-22
> 面向开发者：从零接入 GGID 认证、授权、用户管理的完整指南

---

## 目录

1. [架构概览](#1-架构概览)
2. [快速开始](#2-快速开始5-分钟)
3. [认证流程](#3-认证流程)
4. [Token 验证](#4-token-验证)
5. [SDK 使用](#5-sdk-使用)
6. [权限控制](#6-权限控制)
7. [MFA 集成](#7-mfa-集成)
8. [OAuth 客户端配置](#8-oauth-客户端配置)
9. [ERP Demo 示例](#9-erp-demo-示例)
10. [API 参考](#10-api-参考)
11. [错误处理](#11-错误处理)

---

## 1. 架构概览

```
┌─────────────┐                    ┌──────────┐                    ┌───────────┐
│  你的应用    │  ── OAuth2 ──►     │  GGID     │  ── JWT ──►        │  你的应用  │
│  (前端)     │                    │  Gateway  │                    │  (后端)   │
│             │  ◄── Token ──      │  :8080    │  ── userinfo ──►   │           │
└─────────────┘                    └──────────┘                    └───────────┘
                                         │
                                    ┌────┴────┐
                                    │ Backend │
                                    │ Services│
                                    └─────────┘
```

**关键点**：
- 所有 API 调用通过 **Gateway**（单一入口）
- 后端服务不直接暴露给外部
- JWT 使用 **RS256** 签名，通过 JWKS 验证
- 多租户：每个请求需带 `X-Tenant-ID` header

---

## 2. 快速开始（5 分钟）

### 2.1 获取凭证

```bash
# 1. 获取租户 ID（如果只知道 slug）
curl -s https://ggid.iot2.win/api/v1/tenants/resolve?slug=default
# 返回: {"tenant_id":"28d6fe98-...", "name":"Default", "slug":"default"}

# 2. 登录获取 token（OAuth2 Password Grant）
curl -s -X POST https://ggid.iot2.win/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" \
  -d "grant_type=password&username=admin&password=YOUR_PASSWORD&client_id=YOUR_CLIENT_ID&scope=openid profile email offline_access"
```

**返回**：
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "rt_abc123...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### 2.2 调用 API

```bash
# 使用 access token 调用受保护 API
curl -s https://ggid.iot2.win/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6"
```

### 2.3 刷新 Token

```bash
# Access token 过期后，用 refresh token 自动刷新
curl -s -X POST https://ggid.iot2.win/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=rt_abc123...&client_id=YOUR_CLIENT_ID"
```

---

## 3. 认证流程

GGID 支持 7 种 OAuth2 Grant Type：

### 3.1 Authorization Code + PKCE（推荐，Web/SPA/移动端）

```
浏览器                你的服务器          GGID Gateway
  │                      │                    │
  │── 1. 生成 PKCE ──────────────────────────►│
  │                      │                    │
  │── 2. 重定向到 /oauth/authorize ──────────►│
  │                      │   (用户登录)        │
  │◄── 3. 回调 + code ──────────────────────│
  │                      │                    │
  │── 4. 交换 token ──────────────────────►│
  │   (code + code_verifier)                │
  │◄── 5. access_token + refresh_token ────│
```

**示例**（JavaScript）：
```javascript
// 1. 生成 PKCE
const verifier = base64url(crypto.getRandomValues(new Uint8Array(32)));
const challenge = base64url(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier)));

// 2. 重定向到授权页面
const authUrl = `${GGID_URL}/oauth/authorize?` + new URLSearchParams({
  response_type: 'code',
  client_id: CLIENT_ID,
  redirect_uri: REDIRECT_URI,
  scope: 'openid profile email offline_access',
  state: randomState,
  code_challenge: challenge,
  code_challenge_method: 'S256',
});
window.location.href = authUrl;

// 3. 回调后交换 token
const tokenResp = await fetch(`${GGID_URL}/oauth/token`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({
    grant_type: 'authorization_code',
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: verifier,
  }),
});
const { access_token, refresh_token } = await tokenResp.json();
```

### 3.2 Password Grant（第一方可信应用）

```bash
curl -X POST https://ggid.iot2.win/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -d "grant_type=password&username=USER&password=PASS&client_id=CLIENT_ID&scope=openid profile email offline_access"
```

> 仅用于第一方可信应用（如自家管理后台），不适用于第三方应用。

### 3.3 Client Credentials（M2M 服务间调用）

```bash
curl -X POST https://ggid.iot2.win/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -d "grant_type=client_credentials&client_id=SERVICE_ID&client_secret=SERVICE_SECRET&scope=users:read orders:read"
```

返回的 token 包含 client 的 scope 权限，无需用户参与。

### 3.4 Device Code Flow（IoT/无浏览器设备）

```bash
# 1. 请求 device code
curl -X POST https://ggid.iot2.win/api/v1/oauth/device_authorize \
  -d "client_id=DEVICE_CLIENT_ID&scope=openid profile"

# 返回: {"device_code":"...","user_code":"ABC123","verification_uri":"https://ggid.iot2.win/device"}
# 用户在手机/电脑上访问 verification_uri 输入 user_code 授权

# 2. 轮询 token
curl -X POST https://ggid.iot2.win/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=DEVICE_CODE&client_id=DEVICE_CLIENT_ID"
# 未授权时返回: {"error":"authorization_pending"}
# 授权后返回: {"access_token":"...","refresh_token":"..."}
```

### 3.5 Token Exchange（RFC 8693，降权委派）

```bash
curl -X POST https://ggid.iot2.win/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=ORIGINAL_TOKEN&subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "scope=orders:read"
```

### 3.6 JWT Bearer（RFC 7523）

```bash
curl -X POST https://ggid.iot2.win/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer" \
  -d "assertion=YOUR_JWT_ASSERTION"
```

---

## 4. Token 验证

### 4.1 JWT 结构

GGID 使用 RS256 签名的 JWT，包含以下 claims：

```json
{
  "sub": "user-uuid",
  "username": "admin",
  "email": "admin@example.com",
  "tenant_id": "28d6fe98-...",
  "roles": ["Administrator"],
  "permissions": ["users:read", "users:write", "admin"],
  "scope": "openid profile email offline_access",
  "exp": 1735430400,
  "iat": 1735429500,
  "iss": "https://ggid.iot2.win"
}
```

### 4.2 通过 JWKS 验证（推荐）

```bash
# 获取公钥
curl -s https://ggid.iot2.win/.well-known/jwks.json
```

**Go**：
```go
import "github.com/ggid/ggid/sdk/go"

client := ggid.NewGGIDClient("https://ggid.iot2.win")
info, err := client.VerifyToken(ctx, token)
// info.UserID, info.Roles, info.Permissions, info.TenantID
```

**Node.js**：
```typescript
import { GGIDClient } from '@ggid/sdk';

const client = new GGIDClient({ gatewayUrl: 'https://ggid.iot2.win' });
const claims = await client.verifyToken(token);
// claims.sub, claims.roles, claims.permissions
```

### 4.3 通过 Introspect 验证

```bash
curl -X POST https://ggid.iot2.win/oauth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Tenant-ID: YOUR_TENANT_ID" \
  -d "token=YOUR_TOKEN"
```

返回：
```json
{
  "active": true,
  "sub": "user-uuid",
  "username": "admin",
  "permissions": ["users:read", "admin"],
  "exp": 1735430400
}
```

### 4.4 从 JWT 提取用户信息（无需额外调用）

Access token 的 claims 已包含 `email`、`name`、`roles`、`permissions`，可直接解码使用，无需调用 `/oauth/userinfo`。

---

## 5. SDK 使用

### 5.1 Go SDK

```bash
go get github.com/ggid/ggid/sdk/go
```

```go
package main

import (
    "context"
    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    client := ggid.NewGGIDClient("https://ggid.iot2.win")

    // 登录
    tokens, err := client.Login(ctx, &ggid.LoginRequest{
        Username: "admin",
        Password: "password",
    })

    // M2M 认证
    tokens, err := client.ClientCredentials(ctx, &ggid.ClientCredentialsRequest{
        ClientID:     "my-service",
        ClientSecret: "secret",
        Scope:        "users:read",
        TenantID:     "28d6fe98-...",
    })

    // 验证 token
    info, err := client.VerifyToken(ctx, tokens.AccessToken)
    // info.UserID, info.Roles, info.Permissions

    // 调用 API
    users, err := client.ListUsers(ctx, tokens.AccessToken)
}
```

### 5.2 Node SDK

```bash
npm install @ggid/sdk
```

```typescript
import { GGIDClient } from '@ggid/sdk';

const client = new GGIDClient({ gatewayUrl: 'https://ggid.iot2.win' });

// 登录
const tokens = await client.login({ username: 'admin', password: 'password' });

// M2M
const m2mTokens = await client.clientCredentials({
  clientId: 'my-service',
  clientSecret: 'secret',
  scope: 'users:read',
  tenantId: '28d6fe98-...',
});

// 验证 token
const claims = await client.verifyToken(tokens.access_token);

// Token Manager（自动刷新）
import { TokenManager } from '@ggid/sdk';
const tm = new TokenManager(client);
await tm.login({ username: 'admin', password: 'password' });
const token = await tm.getAccessToken(); // 自动刷新
```

### 5.3 其他 SDK

| 语言 | 路径 | 状态 |
|------|------|------|
| Go | `sdk/go/` | ✅ 完整 |
| Node.js | `sdk/node/` | ✅ 完整 |
| Java | `sdk/java/` | ✅ 完整 |
| Python | `sdk/python/` | ✅ 完整 |
| Rust | `sdk/rust/` | ✅ 完整 |
| Ruby | `sdk/ruby/` | ✅ 完整 |
| C# | `sdk/csharp/` | ✅ 完整 |
| Dart | `sdk/dart/` | ✅ 完整 |
| PHP | `sdk/php/` | ✅ 完整 |
| React | `sdk/react/` | ✅ React Hooks 组件库 |
| cURL | `sdk/curl/` | ✅ Shell 脚本 |

---

## 6. 权限控制

### 6.1 权限格式

```
<resource>:<action>
```

| 权限 | 说明 |
|------|------|
| `users:read` | 查看用户 |
| `users:write` | 创建/修改用户 |
| `users:delete` | 删除用户 |
| `roles:read` | 查看角色 |
| `roles:write` | 创建/修改角色 |
| `audit:read` | 查看审计日志 |
| `orders:approve` | 审批订单 |
| `admin` | 超级权限 |

### 6.2 在后端检查权限

**Go**：
```go
import ggid "github.com/ggid/ggid/sdk/go"

func handler(w http.ResponseWriter, r *http.Request) {
    info, err := client.VerifyToken(r.Context(), token)
    if err != nil { /* 401 */ }

    // 检查权限
    if !ggid.HasPermission(info, "users:write") {
        http.Error(w, "Forbidden", 403)
        return
    }
    // 处理请求...
}
```

**Node.js**：
```typescript
const claims = await client.verifyToken(token);
if (!claims.permissions.includes('users:write')) {
  return res.status(403).json({ error: 'Forbidden' });
}
```

### 6.3 通过 PDP 检查（策略引擎）

```bash
curl -X POST https://ggid.iot2.win/api/v1/policies/check \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "users:delete",
    "resource": "users/550e8400-...",
    "context": {"ip": "192.168.1.1", "time": "2026-07-22T10:00:00Z"}
  }'

# 返回: {"allowed": true, "reason": "permit"}
```

---

## 7. MFA 集成

### 7.1 TOTP 启用流程

```bash
# 1. 开始 TOTP 注册
curl -X POST https://ggid.iot2.win/api/v1/auth/mfa/setup \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'

# 返回: {"device_id":"...","secret":"JBSWY3DP...","qr_code_uri":"otpauth://totp/..."}

# 2. 验证 TOTP 码
curl -X POST https://ggid.iot2.win/api/v1/auth/mfa/verify \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"DEVICE_ID","code":"123456"}'
```

### 7.2 支持的 MFA 方式

| 方式 | 端点 |
|------|------|
| TOTP | `POST /api/v1/auth/mfa/setup` + `POST /api/v1/auth/mfa/verify` |
| Passkey | `POST /api/v1/auth/webauthn/register/begin` + `/finish` |
| YubiKey | `POST /api/v1/auth/mfa/yubikey/verify` |
| Backup Codes | `POST /api/v1/auth/mfa/backup-codes/generate` |

---

## 8. OAuth 客户端配置

### 8.1 在 Console 创建客户端

1. `Console → Settings → OAuth Clients → Create`
2. 配置：
   - Client ID（自动生成或自定义）
   - Client Secret（confidential 客户端需要，public 客户端不需要）
   - Redirect URIs（授权回调地址）
   - Grant Types（选择支持的认证方式）
   - Scopes（允许请求的权限范围）

### 8.2 客户端类型

| 类型 | 说明 | Client Secret |
|------|------|---------------|
| Confidential | 后端服务、M2M | 需要 |
| Public | SPA、移动端、浏览器应用 | 不需要 |

> SPA 使用 Public 客户端 + PKCE，不持有 Client Secret。

---

## 9. ERP Demo 示例

GGID 提供 8 种语言的 ERP Demo，演示完整的认证 + RBAC 流程：

| Demo | 语言 | 认证方式 | 路径 |
|------|------|---------|------|
| ERP Node | Node.js | OAuth2 Password Grant | `examples/erp-node/` |
| ERP React | React/Next.js | OAuth2 Auth Code + PKCE | `examples/erp-react/` |
| ERP Go | Go | OAuth2 Auth Code + PKCE | `examples/erp-go/` |
| ERP Java | Java/Spring | OAuth2 Client Credentials | `examples/erp-java/` |
| ERP Python | Python/FastAPI | OAuth2 Password Grant | `examples/erp-python/` |
| ERP Rust | Rust/Axum | OAuth2 Client Credentials | `examples/erp-rust/` |
| ERP Ruby | Ruby/Sinatra | OAuth2 Password Grant | `examples/erp-ruby/` |
| ERP C# | C#/ASP.NET | OAuth2 Client Credentials | `examples/erp-csharp/` |

### 9.1 运行 ERP Node Demo

```bash
cd examples/erp-node
npm install

# 配置环境变量
export GGID_URL=https://ggid.iot2.win
export GGID_TENANT_ID=28d6fe98-adeb-4c0c-b49b-20c6695bbca6
export ERP_CLIENT_ID=erp-node-m2m
export ERP_CLIENT_SECRET=your-secret

# 启动
npm start
# 服务运行在 http://localhost:3200
```

### 9.2 运行 ERP React Demo

```bash
cd examples/erp-react
npm install

# 配置
export NEXT_PUBLIC_GGID_URL=https://ggid.iot2.win
export NEXT_PUBLIC_CLIENT_ID=erp-react-demo
export NEXT_PUBLIC_REDIRECT_URI=http://localhost:3300/callback

npm run dev
# 服务运行在 http://localhost:3300
```

---

## 10. API 参考

### 10.1 核心 API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/oauth/token` | POST | 获取/刷新 token |
| `/oauth/authorize` | GET | 授权页面 |
| `/oauth/introspect` | POST | Token 内省 |
| `/api/v1/oauth/device_authorize` | POST | Device Code Flow |
| `/.well-known/jwks.json` | GET | JWKS 公钥 |
| `/.well-known/openid-configuration` | GET | OIDC Discovery |
| `/api/v1/users` | GET/POST | 用户 CRUD |
| `/api/v1/users/:id` | GET/PUT/DELETE | 单用户操作 |
| `/api/v1/roles` | GET/POST | 角色 CRUD |
| `/api/v1/orgs` | GET/POST | 组织 CRUD |
| `/api/v1/auth/mfa/setup` | POST | MFA TOTP 注册 |
| `/api/v1/auth/mfa/verify` | POST | MFA 验证 |
| `/api/v1/auth/password/policy` | GET/POST | 密码策略 |
| `/api/v1/tenants/:id/branding` | GET/PUT | 品牌化配置 |
| `/api/v1/policies/check` | POST | PDP 权限检查 |
| `/api/v1/audit/events` | GET | 审计日志 |

### 10.2 请求头

| Header | 必需 | 说明 |
|--------|------|------|
| `Authorization` | 是（受保护端点） | `Bearer <access_token>` |
| `X-Tenant-ID` | 是 | 租户 UUID |
| `Content-Type` | 是（POST/PUT） | `application/json` 或 `application/x-www-form-urlencoded` |

---

## 11. 错误处理

### 11.1 OAuth 错误码

| 错误码 | 说明 | 处理方式 |
|--------|------|---------|
| `invalid_grant` | 凭证无效或过期 | 重新认证 |
| `invalid_client` | Client 认证失败 | 检查 client_id/secret |
| `invalid_request` | 请求参数错误 | 检查请求体 |
| `unsupported_grant_type` | 不支持的 grant type | 检查 client 配置 |
| `authorization_pending` | Device flow 未授权 | 继续轮询 |
| `slow_down` | Device flow 轮询太快 | 降低频率 |
| `mfa_required` | 需要 MFA 验证 | 提交 MFA code |
| `access_denied` | 用户拒绝授权 | 终止流程 |

### 11.2 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证（token 缺失/无效/过期） |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 冲突（如用户名重复） |
| 429 | 请求频率超限 |
| 500 | 服务器内部错误 |

### 11.3 Token 过期处理

```
1. API 返回 401
2. 检查是否有 refresh_token
3. POST /oauth/token with grant_type=refresh_token
4. 获取新 access_token + refresh_token（family rotation）
5. 用新 token 重试原请求
6. 如果刷新也失败 → 重定向到登录页
```

---

## 附录

### A. 环境变量速查

| 变量 | 说明 | 示例 |
|------|------|------|
| `GGID_URL` | Gateway 地址 | `https://ggid.iot2.win` |
| `GGID_TENANT_ID` | 租户 ID | `28d6fe98-adeb-4c0c-b49b-20c6695bbca6` |
| `OAUTH_CLIENT_ID` | OAuth 客户端 ID | `ggid-console` |
| `OAUTH_CLIENT_SECRET` | OAuth 客户端密钥（confidential only） | `your-secret` |
| `OAUTH_REDIRECT_URI` | 授权回调地址 | `http://localhost:3300/callback` |

### B. Docker 快速部署

```bash
cd deploy && docker compose up -d
# Gateway: http://localhost:8080
# Console: http://localhost:3000

# Bootstrap admin
ggid-cli system bootstrap --admin-user admin --admin-password 'SecurePass123!'
```

### C. kubectl 部署

```bash
kubectl apply -f deploy/k8s/
kubectl get pods -n ggid
```

### D. 相关文档

- [用户手册](./user-manual.md) — Console 操作指南
- [部署指南](./deployment-guide.md) — 自托管部署
- [API Cookbook](./api-cookbook.md) — 20 个 curl 示例
- [OAuth Flows Guide](./oauth-flows-guide.md) — OAuth 流程详解
- [MFA Guide](./mfa-guide.md) — MFA 集成详解
- [RBAC Guide](./rbac-guide.md) — 角色权限详解