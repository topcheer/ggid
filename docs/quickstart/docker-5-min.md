# Docker 5-Minute Quickstart

> From zero to authenticated API call in 5 minutes with Docker Compose.

---

## Step 1: Start GGID

```bash
# Clone
git clone https://github.com/ggid/ggid.git && cd ggid

# Start all 12 containers
cd deploy && docker compose up -d

# Wait for healthchecks (30-60s)
sleep 30

# Verify all containers are healthy
docker compose ps
```

All containers should show `Up (healthy)`.

## Step 2: Register a User

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","email":"alice@example.com","password":"W3lcome-2025!"}' | jq .
```

**Expected output:**
```json
{
  "user_id": "usr_abc123",
  "username": "alice",
  "email": "alice@example.com",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "created_at": "2025-07-11T12:00:00Z"
}
```

## Step 3: Login → Get JWT

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"alice","password":"W3lcome-2025!"}' | jq -r .access_token)

echo "JWT: ${JWT:0:30}..."
echo "Length: ${#JWT} chars"
```

**Expected output:**
```
JWT: eyJhbGciOiJIUzI1NiIsInR5...
Length: 693 chars
```

## Step 4: Use the JWT

```bash
# Call a protected endpoint
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

**Expected:** `200 OK` with user list.

## Step 5: Verify Auth Works

```bash
# Without JWT → should get 401
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
# → 401

# With JWT → should get 200
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
# → 200
```

## Step 6: Admin Console

Open `http://localhost:3000` in your browser. Login with the credentials you just created.

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `connection refused` | Wait 30s for containers to start |
| Register returns 409 | Username taken — use a different one |
| 401 with JWT | JWT expired — login again |
| 429 Too Many Requests | Rate limited — wait 60s or restart auth container |

---

*See: [Full Getting Started](../getting-started.md) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*