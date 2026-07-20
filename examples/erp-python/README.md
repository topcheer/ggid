# ERP Python Demo

Cross-board ERP demo using GGID Python SDK.

## Features
- OAuth login via GGID
- JWT permissions claim (fine-grained)
- 7 modules: auth, users, roles, orgs, inventory, orders, audit
- Permission matrix: Viewer / Sales / Manager / Admin

## Run locally

```bash
GGID_URL=https://ggid.iot2.win \
ADMIN_USERNAME=admin \
ADMIN_PASSWORD=q7Rf9Xk2Lm3pW8zBA \
TENANT_ID=00000000-0000-0000-0000-000000000001 \
python3 main.py
```

## API Endpoints

| Method | Path | Permission |
|--------|------|------------|
| GET | /login | public |
| GET | /api/inventory | inventory:read |
| POST | /api/inventory | inventory:write |
| DELETE | /api/inventory/{id} | inventory:delete |
| GET | /api/orders | orders:read |
| POST | /api/orders | orders:write |
| POST | /api/orders/{id}/approve | orders:approve |
| GET | /api/users | users:read |
| GET | /api/roles | roles:read |
| GET | /api/audit | audit:read |
| GET | /api/my-permissions | authenticated |
