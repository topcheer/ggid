# 5-Minute JWT Quickstart

> Register, login, get a JWT, and use it to call a protected API.

---

## Prerequisites

```bash
docker compose -f deploy/docker-compose.yml up -d
sleep 30  # wait for healthchecks
```

---

## curl

```bash
# 1. Register
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","email":"alice@example.com","password":"Secure1Pass!"}' | jq -r .user_id)

# 2. Login → get JWT
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","password":"Secure1Pass!"}' | jq -r .access_token)

# 3. Use JWT
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
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