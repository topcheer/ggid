# GGID Cross-Board ERP Demo — Go

A Go implementation of the ERP demo using the GGID Go SDK. Tests all core GGID features.

## Modules

1. **Auth** — Login/logout/token refresh/verify (OAuth 2.1)
2. **Users** — CRUD users + role assignment
3. **Roles & Permissions** — CRUD roles + permission tree
4. **Organizations** — CRUD orgs with hierarchy
5. **Inventory** — CRUD products/stock
6. **Orders** — CRUD orders + approval workflow
7. **Audit** — View audit log

## Permission Matrix

| Role | Permissions |
|------|------------|
| Viewer | dashboard:read, inventory:read, orders:read |
| Sales | + orders:write |
| Manager | + orders:approve, orders:read:all |
| Admin | admin (bypass) |

## Setup

```bash
# Start GGID services first
make docker-run && make migrate-up && make build

# Run ERP demo
cd examples/erp-go
go run .

# Server starts on :9090
```

## API Endpoints

### Auth (public)
- `POST /api/auth/login` — Login with username/password
- `POST /api/auth/refresh` — Refresh token
- `POST /api/auth/verify` — Verify token

### Protected (requires JWT + permission)
- `GET/POST /api/users` — users:read / users:write
- `GET/PUT/DELETE /api/users/:id` — users:read / users:write / users:delete
- `GET/POST /api/roles` — roles:read / roles:write
- `GET/POST /api/orgs` — orgs:read / orgs:write
- `GET/POST /api/inventory` — inventory:read / inventory:write
- `GET/PUT/DELETE /api/inventory/:id` — inventory:read / inventory:write / inventory:delete
- `GET/POST /api/orders` — orders:read / orders:write
- `PUT /api/orders/:id/approve` — orders:approve
- `GET /api/audit` — audit:read

## Testing GGID Features

- ✅ OAuth 2.1 token issuance + refresh
- ✅ JWT permissions claim (fine-grained authorization)
- ✅ RBAC (role assignment)
- ✅ User CRUD via identity API
- ✅ Organization hierarchy
- ✅ Audit log viewing
- ✅ Multi-tenant (X-Tenant-ID header)
- ✅ Go SDK: Login, VerifyToken, CheckPermission, CreateUser, ListUsers, etc.
