# GGID Cross-Board ERP Demo — Ruby

ERP demo using GGID Ruby SDK. Tests all core features.

## Setup

```bash
# Install dependencies
gem install sinatra

# Start GGID services first
# Run ERP demo
cd examples/erp-ruby
GGID_URL=http://localhost:8080 ruby app.rb
# Server starts on :9091
```

## Modules
1. Auth — login/refresh/verify
2. Users — CRUD via GGID SDK
3. Roles — CRUD via GGID SDK
4. Orgs — CRUD
5. Inventory — CRUD (inventory:read/write/delete)
6. Orders — CRUD + approval (orders:read/write/approve)
7. Audit — View audit log

## Permission Matrix
| Role | Permissions |
|------|------------|
| Viewer | dashboard:read, inventory:read, orders:read |
| Sales | + orders:write |
| Manager | + orders:approve, orders:read:all |
| Admin | admin (bypass) |
