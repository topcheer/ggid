# Developer Onboarding

> From zero to first authenticated API call. Follow each step in order.

---

## Prerequisites

| Tool | Version | Install |
|------|---------|--------|
| Go | 1.25+ | `brew install go` |
| Docker | 24+ | [docker.com](https://docker.com) |
| Node.js | 20+ | `brew install node` |
| jq | latest | `brew install jq` |

---

## Step 1: Clone

```bash
git clone https://github.com/ggid/ggid.git
cd ggid
```

## Step 2: Build

```bash
make build
# Output:
# bin/gateway
# bin/identity
# bin/auth
# bin/oauth
# bin/policy
# bin/org
# bin/audit
```

## Step 3: Start Docker Stack

```bash
cd deploy && docker compose up -d
sleep 30

# Verify
docker compose ps
# All 12 containers: Up (healthy)
```

## Step 4: Register

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"dev","email":"dev@test.com","password":"W3lcome-2025!"}' | jq .

# Expected: {"user_id":"usr_...","username":"dev",...}
```

## Step 5: Login

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"dev","password":"W3lcome-2025!"}' | jq -r .access_token)

echo "JWT: ${JWT:0:30}... (${#JWT} chars)"
# Expected: JWT: eyJhbGciOiJIUzI1NiIsInR5... (693 chars)
```

## Step 6: Use JWT

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .

# Expected: {"users":[...],"total":1}
```

## Step 7: Verify 401 Without JWT

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
# Expected: 401
```

## Step 8: Add to Your App

### Go

```go
import (
    "context"
    "time"
    ggid "github.com/ggid/ggid/sdk/go"
)

client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))
userInfo, _ := client.VerifyToken(context.Background(), token)
```

### Node.js

```javascript
const { expressAuth } = require('@ggid/node');
app.use(expressAuth({ jwksUrl: 'http://localhost:8080/.well-known/jwks.json', issuer: 'http://localhost:8080' }));
```

### Python

```python
from ggid import GGIDClient
client = GGIDClient(gateway_url="http://localhost:8080", tenant_id="00000000-0000-0000-0000-000000000001")
```

---

## Next Steps

- [5-Minute JWT Quickstart](./5-minute-jwt.md)
- [SDK Quickstart](./sdk-quickstart.md)
- [API Reference](../api-reference.md)
- [Integration Guides](../integration-guides/)

---

*Last updated: 2025-07-11*