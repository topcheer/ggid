# GGID API Gateway Middleware Demo

A Python (Flask) API gateway demonstrating JWT verification, RBAC permission checking, and request routing with GGID.

## Quick Start

```bash
pip install -r requirements.txt
python gateway.py
```

## Endpoints

| Endpoint | Method | Auth | Permission | Description |
|----------|--------|------|------------|-------------|
| `/health` | GET | None | — | Health check |
| `/api/v1/profile` | GET | JWT | — | Current user profile |
| `/api/v1/products` | GET | JWT | products:read | List products |
| `/api/v1/products` | POST | JWT | products:create | Create product |
| `/api/v1/admin/users` | GET | JWT | users:read | List users (admin) |

## Architecture

```
Client → API Gateway → GGID JWKS (JWT verification)
                    → GGID Policy Engine (RBAC check)
                    → Backend Service (mocked)
```

## Features

1. **JWKS Caching** — Fetches GGID public keys with 5-minute TTL
2. **RBAC Enforcement** — Calls GGID policy engine for permission checks
3. **User Context Injection** — Adds `request.user_id` and `request.user_roles`
4. **Graceful Fallback** — Local permission check when GGID policy is unavailable

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GGID_URL` | `https://ggid.iot2.win` | GGID gateway URL |
| `GGID_TENANT_ID` | `00000000-...` | Tenant ID |
| `PORT` | `5060` | Gateway listen port |
