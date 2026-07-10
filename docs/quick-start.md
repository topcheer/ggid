# GGID — 快速入门指南 (5 分钟集成)

## 1. 启动 GGID 平台

```bash
git clone https://github.com/ggid/ggid.git
cd ggid/deploy
docker compose up -d
```

等待 30 秒，所有服务自动初始化。

## 2. 访问

| 服务 | 地址 |
|------|------|
| 托管登录页 | http://127.0.0.1:8080/login |
| 管理后台 | http://127.0.0.1:3000 |
| API 文档 (Swagger) | http://127.0.0.1:8080/docs |
| OIDC Discovery | http://127.0.0.1:8080/oauth/.well-known/openid-configuration |

默认管理员：`admin / Admin@123456`

## 3. 集成你的应用

### 方式 A：托管登录（推荐，最快）

用户访问你的应用 → 重定向到 GGID 登录 → 认证成功返回 JWT

```
https://your-iam:8080/login?redirect_uri=https://yourapp.com/callback
```

### 方式 B：API 调用

```bash
# 1. 注册用户
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","email":"alice@example.com","password":"StrongPass123!"}'

# 2. 登录获取 JWT
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","password":"StrongPass123!"}' | jq -r .access_token)

# 3. 使用 JWT 访问 API
curl http://127.0.0.1:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

### 方式 C：使用 SDK

**Go:**
```go
client := ggid.NewClient("https://your-iam:8080",
    ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))
tokens, _ := client.Login(ctx, "alice", "StrongPass123!")
users, _ := client.ListUsers(ctx, tokens.AccessToken)
```

**Python:**
```python
from ggid import GGIDClient
client = GGIDClient(gateway_url="https://your-iam:8080")
tokens = await client.login("alice", "StrongPass123!")
users = await client.list_users(tokens["access_token"])
```

**Node.js:**
```typescript
import { GGIDClient } from '@ggid/node';
const client = new GGIDClient({ gatewayUrl: 'https://your-iam:8080' });
const tokens = await client.login('alice', 'StrongPass123!');
const { users } = await client.listUsers(tokens.access_token);
```

## 4. 保护你的应用

### Go HTTP 中间件
```go
protected := client.Middleware(yourHandler)
http.ListenAndServe(":3001", protected)
```

### Python FastAPI
```python
from ggid import GGIDMiddleware
app.add_middleware(GGIDMiddleware, jwks_url="https://your-iam:8080/.well-known/jwks.json")
```

### Node.js Express
```typescript
import { expressAuth } from '@ggid/node';
app.use(expressAuth({ jwksUrl: 'https://your-iam:8080/.well-known/jwks.json' }));
```

## 5. OAuth2 / OIDC 集成

GGID 完全兼容 OAuth2 和 OIDC 标准：

```
Authorization endpoint:  /oauth/authorize
Token endpoint:          /oauth/token
UserInfo endpoint:       /oauth/userinfo
JWKS:                    /.well-known/jwks.json
Discovery:               /oauth/.well-known/openid-configuration
```

## 下一步

- [API 文档](http://127.0.0.1:8080/docs)
- [功能矩阵](docs/feature-matrix.md)
- [路线图](docs/roadmap.md)
