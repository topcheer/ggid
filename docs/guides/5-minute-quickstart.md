# 5-Minute Quickstart

> Go from zero to authenticated API call in 5 minutes.

---

## Step 1: Start GGID (30 seconds)

```bash
cd deploy && docker compose up -d
sleep 30  # Wait for healthchecks
curl http://localhost:8080/healthz  # Should return {"status":"ok"}
```

## Step 2: Register a User (10 seconds)

```bash
TENANT="28d6fe98-adeb-4c0c-b49b-20c6695bbca6"

curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","email":"alice@test.com","password":"Secure123!"}'
```

## Step 3: Login & Get JWT (10 seconds)

```bash
JWT=$(curl -s -X POST http://localhost:8080/oauth/token \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Secure123!"}' | jq -r .access_token)

echo "JWT: ${#JWT} chars"
```

## Step 4: Call Protected API (5 seconds)

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

## Step 5: Add JWT Auth to Your App (3 minutes)

### Go

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))
user, _ := client.VerifyToken(ctx, jwt)
```

### React

```jsx
import { GGIDProvider, useAuth } from '@ggid/react';

<GGIDProvider domain="localhost:8080" tenantId="...">
  <App />
</GGIDProvider>

function App() {
  const { isAuthenticated, user } = useAuth();
  return isAuthenticated ? <Dashboard user={user} /> : <Login />;
}
```

### Node.js

```javascript
const { expressAuth, getClaims } = require('@ggid/node');
app.use('/api', expressAuth({ jwksUrl: '.../.well-known/jwks.json' }));
```

---

**Done!** Full JWT auth integrated in 5 minutes.

*See: [3-Line Integration](../quickstart/3-line-integration.md) | [Docker Quickstart](../quickstart/docker-5-min.md) | [SDK Quickstart](../quickstart/sdk-quickstart.md)*

*Last updated: 2025-07-11*
