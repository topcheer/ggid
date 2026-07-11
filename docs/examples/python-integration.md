# Python FastAPI Integration Example

> Complete, runnable FastAPI application using the GGID Python SDK for JWT verification and route protection.

---

## Prerequisites

- Python 3.10+
- GGID Gateway running at `http://localhost:8080`

---

## Project Setup

```bash
mkdir ggid-python-demo && cd ggid-python-demo
python -m venv .venv && source .venv/bin/activate
pip install fastapi uvicorn ggid
```

---

## Complete Application

Create `app.py`:

```python
from fastapi import FastAPI, Depends, HTTPException, Request
from pydantic import BaseModel
from ggid import GGIDClient
from ggid.middleware import GGIDMiddleware, get_current_user

import os

app = FastAPI(title="GGID Python Demo")

# ─── Configuration ───────────────────────────────────────────
GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
JWKS_URL = os.getenv("JWKS_URL", f"{GGID_URL}/.well-known/jwks.json")
TENANT_ID = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")

# ─── Middleware ──────────────────────────────────────────────
app.add_middleware(
    GGIDMiddleware,
    gateway_url=GGID_URL,
    jwks_url=JWKS_URL,
    tenant_id=TENANT_ID,
)

# ─── Models ──────────────────────────────────────────────────
class CreateUserRequest(BaseModel):
    username: str
    email: str
    password: str

class PermissionCheck(BaseModel):
    action: str = "read"
    resource: str = "users"

# ─── Public Routes ───────────────────────────────────────────
@app.get("/health")
async def health():
    return {"status": "ok", "service": "python-demo"}

# ─── Protected Routes ────────────────────────────────────────
@app.get("/api/me")
async def me(user=Depends(get_current_user)):
    """Get current user info from JWT."""
    return {
        "user_id": user.get("sub"),
        "tenant_id": user.get("tenant_id"),
        "email": user.get("email"),
        "scope": user.get("scope", "").split(),
    }

@app.get("/api/users")
async def list_users(request: Request, user=Depends(get_current_user)):
    """List users — requires read:users scope."""
    scopes = user.get("scope", "").split()
    if "read:users" not in scopes:
        raise HTTPException(status_code=403, detail="insufficient_scope: read:users")

    client = GGIDClient(
        gateway_url=GGID_URL,
        tenant_id=user.get("tenant_id", TENANT_ID),
    )
    result = await client.list_users()
    return {"users": result.get("users", []), "count": len(result.get("users", []))}

@app.post("/api/users", status_code=201)
async def create_user(req: CreateUserRequest, user=Depends(get_current_user)):
    """Create user — requires admin role."""
    roles = user.get("roles", [])
    if "admin" not in roles:
        raise HTTPException(status_code=403, detail="admin role required")

    client = GGIDClient(
        gateway_url=GGID_URL,
        tenant_id=user.get("tenant_id", TENANT_ID),
    )
    try:
        new_user = await client.create_user({
            "username": req.username,
            "email": req.email,
            "password": req.password,
        })
        return new_user
    except Exception as e:
        if "conflict" in str(e).lower() or "exists" in str(e).lower():
            raise HTTPException(status_code=409, detail="user_exists")
        raise HTTPException(status_code=502, detail=str(e))

@app.post("/api/check-permission")
async def check_permission(req: PermissionCheck, user=Depends(get_current_user)):
    """Check permission via Policy Engine."""
    client = GGIDClient(
        gateway_url=GGID_URL,
        tenant_id=user.get("tenant_id", TENANT_ID),
    )
    result = await client.check_permission(
        user_id=user.get("sub"),
        resource=req.resource,
        action=req.action,
    )
    return result

# ─── Start ───────────────────────────────────────────────────
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", "8000")))
```

---

## Environment Variables

Create `.env`:

```bash
GGID_URL=http://localhost:8080
JWKS_URL=http://localhost:8080/.well-known/jwks.json
TENANT_ID=00000000-0000-0000-0000-000000000001
PORT=8000
```

---

## Run

```bash
uvicorn app:app --reload --port 8000
# → GGID Gateway: http://localhost:8080
```

---

## Test the Endpoints

### Health Check (public)

```bash
curl http://localhost:8000/health
# → {"status":"ok","service":"python-demo"}
```

### Protected Route Without Token (401)

```bash
curl http://localhost:8000/api/me
# → {"detail":"missing or invalid token"}
```

### Get User Info

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin123!"}' | jq -r .access_token)

curl -s http://localhost:8000/api/me \
  -H "Authorization: Bearer $JWT" | jq .
```

### List Users (requires read:users)

```bash
curl -s http://localhost:8000/api/users \
  -H "Authorization: Bearer $JWT" | jq .
```

### Check Permission

```bash
curl -s -X POST http://localhost:8000/api/check-permission \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"action":"write","resource":"users"}' | jq .
```

---

## Key Takeaways

1. **`GGIDMiddleware`** verifies JWT on every request automatically.
2. **`get_current_user`** dependency injects decoded JWT claims into your route.
3. **Scope/role checks** are done inline — raise `HTTPException(403)` for denied access.
4. **`GGIDClient`** calls GGID management APIs server-side.
5. **Tenant isolation** — use `user["tenant_id"]` for all database queries.

---

*See also: [SDK Quickstart](../quickstart/sdk-quickstart.md) | [3-Line Integration](../quickstart/3-line-integration.md) | [RBAC Guide](../guides/role-based-access.md)*

*Last updated: 2025-07-11*
