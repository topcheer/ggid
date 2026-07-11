# 5-Minute JWT Quickstart

> Register, login, get a JWT, and use it to call a protected API.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) 24+ with Docker Compose v2
- `jq` installed (`brew install jq` or `apt install jq`)
- Port 8080 available

```bash
# Clone and start
git clone https://github.com/ggid/ggid.git
cd ggid
docker compose -f deploy/docker-compose.yaml up -d

# Wait for all 12 containers to be healthy (30-60 seconds)
sleep 30
docker compose ps  # all should show "Up (healthy)"

# Verify gateway is up
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

### Variables used below

```bash
export GGID_URL="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"
```

---

## curl

```bash
# 1. Register a new user
curl -s -X POST $GGID_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","email":"alice@example.com","password":"Secure1Pass!"}' | jq .
# → {"user_id":"usr_abc123","username":"alice",...}  (201 Created)

# 2. Login → get JWT
JWT=$(curl -s -X POST $GGID_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Secure1Pass!"}' | jq -r .access_token)

echo "JWT length: ${#JWT} chars"  # should be ~690 chars

# 3. Use JWT to call a protected endpoint
curl -s $GGID_URL/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
# → {"users":[...],"total":1}  (200 OK)

# 4. Verify 401 without JWT
curl -s -o /dev/null -w "%{http_code}" $GGID_URL/api/v1/users \
  -H "X-Tenant-ID: $TENANT"
# → 401
```

---

## Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func main() {
    tenant := "00000000-0000-0000-0000-000000000001"
    base := "http://localhost:8080"

    // 1. Login
    body, _ := json.Marshal(map[string]string{
        "username": "alice", "password": "Secure1Pass!",
    })
    resp, _ := http.Post(base+"/api/v1/auth/login",
        bytes.NewReader(body), )
    defer resp.Body.Close()

    var result struct{ AccessToken string `json:"access_token"` }
    json.NewDecoder(io.Reader(resp.Body)).Decode(&result)

    // 2. Use JWT
    req, _ := http.NewRequest("GET", base+"/api/v1/users", nil)
    req.Header.Set("Authorization", "Bearer "+result.AccessToken)
    req.Header.Set("X-Tenant-ID", tenant)

    resp2, _ := http.DefaultClient.Do(req)
    fmt.Println("Status:", resp2.Status)
}
```

---

## Node.js

```javascript
const base = 'http://localhost:8080';
const tenant = '00000000-0000-0000-0000-000000000001';

// 1. Login
const { access_token } = await fetch(`${base}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenant },
  body: JSON.stringify({ username: 'alice', password: 'Secure1Pass!' })
}).then(r => r.json());

// 2. Use JWT
const users = await fetch(`${base}/api/v1/users`, {
  headers: { Authorization: `Bearer ${access_token}`, 'X-Tenant-ID': tenant }
}).then(r => r.json());

console.log('Users:', users);
```

---

*That's it. You have a working JWT-authenticated API in under 5 minutes.*

*See: [API Reference](../api-reference.md) | [Authentication Guide](../authentication-guide.md)*